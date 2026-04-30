package gateway

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/forgebox/forgebox/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_DispatchesToAllClientsForUser(t *testing.T) {
	bus := events.New(8)
	h := NewHub(bus, 4)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	// Two tabs for u1, one for u2.
	a := h.register("u1")
	b := h.register("u1")
	other := h.register("u2")

	bus.Publish(events.Event{Type: "task.updated", UserID: "u1", Payload: map[string]string{"id": "t1"}})

	gotA := readEnvelope(t, a.send)
	gotB := readEnvelope(t, b.send)
	assert.Equal(t, "task.updated", gotA.Type)
	assert.Equal(t, "task.updated", gotB.Type)

	select {
	case msg := <-other.send:
		t.Fatalf("u2 client received event for u1: %s", string(msg))
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHub_DropsEventsWithEmptyUserID(t *testing.T) {
	bus := events.New(4)
	h := NewHub(bus, 4)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := h.register("u1")
	bus.Publish(events.Event{Type: "task.updated", UserID: ""})

	select {
	case msg := <-c.send:
		t.Fatalf("expected no delivery for empty UserID, got %s", string(msg))
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHub_UnregisterRemovesClient(t *testing.T) {
	bus := events.New(4)
	h := NewHub(bus, 4)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := h.register("u1")
	h.unregister(c)

	bus.Publish(events.Event{Type: "task.updated", UserID: "u1"})

	// send channel was closed by unregister.
	select {
	case _, ok := <-c.send:
		assert.False(t, ok, "expected closed channel after unregister")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected channel close signal after unregister")
	}
}

func TestHub_EvictsSlowClient(t *testing.T) {
	bus := events.New(8)
	// Per-client buffer of 1 so the second event evicts.
	h := NewHub(bus, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := h.register("u1")

	// First event fills the buffer.
	bus.Publish(events.Event{Type: "first", UserID: "u1"})
	// Wait for delivery to land.
	require.Eventually(t, func() bool {
		return len(c.send) == 1
	}, time.Second, 10*time.Millisecond)

	// Second event has nowhere to go; client must be evicted.
	bus.Publish(events.Event{Type: "second", UserID: "u1"})

	require.Eventually(t, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return len(h.clients["u1"]) == 0
	}, time.Second, 10*time.Millisecond, "expected slow client to be evicted")
}

func TestHub_StopClosesAllClients(t *testing.T) {
	bus := events.New(4)
	h := NewHub(bus, 4)

	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)

	c := h.register("u1")
	cancel()

	select {
	case _, ok := <-c.send:
		assert.False(t, ok, "expected client send channel closed after Hub stop")
	case <-time.After(time.Second):
		t.Fatal("hub did not close clients on stop")
	}
}

func readEnvelope(t *testing.T, ch <-chan []byte) outboundMessage {
	t.Helper()
	select {
	case b := <-ch:
		var msg outboundMessage
		require.NoError(t, json.Unmarshal(b, &msg))
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for outbound message")
		return outboundMessage{}
	}
}
