package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

const (
	wsAuthTimeout    = 10 * time.Second
	wsPingInterval   = 30 * time.Second
	wsPongTimeout    = 10 * time.Second
	wsWriteTimeout   = 10 * time.Second
	wsCloseAuthError = 4001
)

// inboundMessage is what clients send. The auth handshake uses Type="auth" with
// a Token in Payload; subsequent messages use Type="pong" today.
type inboundMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type authPayload struct {
	Token string `json:"token"`
}

// approvalPayload is the body shape of an inbound "tool_approval" message
// from the dashboard.
type approvalPayload struct {
	ApprovalID string `json:"approval_id"`
	Decision   string `json:"decision"` // "approve" or "deny"
}

// handleWS upgrades an HTTP request to a WebSocket, performs the first-message
// auth handshake, registers the client with the hub, and runs read/write
// loops until the connection closes.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Dev: web at :3000, gateway at :8420 — the Vite proxy forwards same-origin,
		// but allow direct dev access too.
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Warn("ws: accept failed", "error", err)
		return
	}
	// We rely on conn.Close (with a status) at the end of every path; defer a
	// safety net that fires only if a deeper path didn't already close.
	defer func() { _ = conn.CloseNow() }()

	ctx := r.Context()

	userID, err := wsAuthenticate(ctx, conn)
	if err != nil {
		slog.Info("ws: auth failed", "error", err)
		_ = conn.Close(wsCloseAuthError, "auth failed")
		return
	}
	if err := wsWriteJSON(ctx, conn, outboundMessage{Type: "auth_ok"}); err != nil {
		return
	}

	client := s.hub.register(userID)
	defer s.hub.unregister(client)

	// onMsg routes non-pong inbound frames. Today the only message we
	// understand here is "tool_approval" from the dashboard.
	onMsg := func(msg inboundMessage) {
		switch msg.Type {
		case "tool_approval":
			if s.approvals == nil {
				return
			}
			var p approvalPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				slog.Info("ws: bad tool_approval payload", "error", err)
				return
			}
			approved := p.Decision == "approve"
			s.approvals.Resolve(p.ApprovalID, approved)
		}
	}

	// pongReceived is touched by the read loop; the ping loop reads it. Use
	// an atomic so we don't need a mutex for one nanosecond timestamp.
	var lastPong atomic.Int64
	lastPong.Store(time.Now().UnixNano())

	// Run read and write/ping loops concurrently. First one to error stops
	// everything via context cancellation.
	loopCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	readErr := make(chan error, 1)
	writeErr := make(chan error, 1)

	go func() { readErr <- wsReadLoop(loopCtx, conn, &lastPong, onMsg) }()
	go func() { writeErr <- wsWriteLoop(loopCtx, conn, client.send, &lastPong) }()

	select {
	case err := <-readErr:
		logWSExit("read", err)
	case err := <-writeErr:
		logWSExit("write", err)
	}
	cancel()
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

// wsAuthenticate reads the first message and returns a userID derived from
// the supplied token. Token validation matches the existing REST middleware:
// any non-empty token is accepted (proper validation is a pre-existing gap
// in the auth layer).
func wsAuthenticate(ctx context.Context, conn *websocket.Conn) (string, error) {
	authCtx, cancel := context.WithTimeout(ctx, wsAuthTimeout)
	defer cancel()

	_, data, err := conn.Read(authCtx)
	if err != nil {
		return "", err
	}

	var msg inboundMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	if msg.Type != "auth" {
		return "", errors.New("first message must be type=auth")
	}
	var p authPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return "", err
	}
	token := strings.TrimSpace(p.Token)
	if token == "" {
		return "", errors.New("empty token")
	}
	// Until proper session/JWT validation lands, the token IS the user key:
	// same token = same fan-out audience across tabs, different token = different audience.
	return "tok:" + token, nil
}

// wsReadLoop consumes inbound messages, updating lastPong on pong messages
// and dispatching everything else through onMsg. Any non-recoverable read
// error returns and ends the connection.
func wsReadLoop(ctx context.Context, conn *websocket.Conn, lastPong *atomic.Int64, onMsg func(inboundMessage)) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		var msg inboundMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			// Tolerate malformed frames; the spec only defines pong + tool_approval today.
			continue
		}
		if msg.Type == "pong" {
			lastPong.Store(time.Now().UnixNano())
			continue
		}
		if onMsg != nil {
			onMsg(msg)
		}
	}
}

// wsWriteLoop drains client.send to the socket and emits pings on a timer.
// If pongs stop arriving (see lastPong) the connection is closed.
func wsWriteLoop(ctx context.Context, conn *websocket.Conn, send <-chan []byte, lastPong *atomic.Int64) error {
	pingTicker := time.NewTicker(wsPingInterval)
	defer pingTicker.Stop()

	pingMsg, _ := json.Marshal(outboundMessage{Type: "ping"})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-send:
			if !ok {
				return errors.New("send channel closed")
			}
			if err := wsWriteRaw(ctx, conn, data); err != nil {
				return err
			}
		case <-pingTicker.C:
			// If no pong within the deadline, abandon the connection.
			deadline := time.Now().Add(-wsPingInterval - wsPongTimeout).UnixNano()
			if lastPong.Load() < deadline {
				return errors.New("pong timeout")
			}
			if err := wsWriteRaw(ctx, conn, pingMsg); err != nil {
				return err
			}
		}
	}
}

func wsWriteJSON(ctx context.Context, conn *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return wsWriteRaw(ctx, conn, data)
}

func wsWriteRaw(ctx context.Context, conn *websocket.Conn, data []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, data)
}

func logWSExit(where string, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	var ce websocket.CloseError
	if errors.As(err, &ce) {
		// Normal close codes are not interesting at warn level.
		slog.Debug("ws: connection closed", "where", where, "code", ce.Code, "reason", ce.Reason)
		return
	}
	slog.Debug("ws: loop exited", "where", where, "error", err)
}
