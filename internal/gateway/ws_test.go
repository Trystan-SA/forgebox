package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forgebox/forgebox/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/coder/websocket"
)

func newTestWSServer(t *testing.T) (*httptest.Server, *events.Bus, *Hub) {
	t.Helper()

	bus := events.New(16)
	hub := NewHub(bus, 16)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go hub.Run(ctx)

	s := &Server{hub: hub}
	ts := httptest.NewServer(http.HandlerFunc(s.handleWS))
	t.Cleanup(ts.Close)

	return ts, bus, hub
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(url, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	return conn
}

func writeJSONToWS(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, conn.Write(ctx, websocket.MessageText, data))
}

func readJSON(t *testing.T, conn *websocket.Conn, timeout time.Duration) outboundMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var msg outboundMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	return msg
}

func TestWS_AuthOkAndEventDelivery(t *testing.T) {
	ts, bus, hub := newTestWSServer(t)
	conn := dialWS(t, ts.URL)
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeJSONToWS(t, conn, map[string]any{
		"type":    "auth",
		"payload": map[string]string{"token": "abc"},
	})

	got := readJSON(t, conn, 2*time.Second)
	assert.Equal(t, "auth_ok", got.Type)

	// Wait for hub registration to settle.
	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients["tok:abc"]) == 1
	}, time.Second, 10*time.Millisecond)

	bus.Publish(events.Event{
		Type:    "task.updated",
		UserID:  "tok:abc",
		Payload: map[string]string{"id": "t1"},
	})

	got = readJSON(t, conn, 2*time.Second)
	assert.Equal(t, "task.updated", got.Type)
}

func TestWS_RejectsMissingToken(t *testing.T) {
	ts, _, _ := newTestWSServer(t)
	conn := dialWS(t, ts.URL)
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeJSONToWS(t, conn, map[string]any{
		"type":    "auth",
		"payload": map[string]string{"token": ""},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := conn.Read(ctx)
	require.Error(t, err)

	var ce websocket.CloseError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, websocket.StatusCode(wsCloseAuthError), ce.Code)
}

func TestWS_RejectsNonAuthFirstMessage(t *testing.T) {
	ts, _, _ := newTestWSServer(t)
	conn := dialWS(t, ts.URL)
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	writeJSONToWS(t, conn, map[string]any{"type": "pong"})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := conn.Read(ctx)
	require.Error(t, err)

	var ce websocket.CloseError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, websocket.StatusCode(wsCloseAuthError), ce.Code)
}

func TestWS_TwoTabsSameTokenBothReceiveEvent(t *testing.T) {
	ts, bus, hub := newTestWSServer(t)

	conns := make([]*websocket.Conn, 2)
	for i := range conns {
		c := dialWS(t, ts.URL)
		t.Cleanup(func() { _ = c.Close(websocket.StatusNormalClosure, "") })
		writeJSONToWS(t, c, map[string]any{
			"type":    "auth",
			"payload": map[string]string{"token": "shared"},
		})
		got := readJSON(t, c, 2*time.Second)
		require.Equal(t, "auth_ok", got.Type)
		conns[i] = c
	}

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients["tok:shared"]) == 2
	}, time.Second, 10*time.Millisecond)

	bus.Publish(events.Event{
		Type:    "notification",
		UserID:  "tok:shared",
		Payload: map[string]string{"title": "hi"},
	})

	for _, c := range conns {
		got := readJSON(t, c, 2*time.Second)
		assert.Equal(t, "notification", got.Type)
	}
}
