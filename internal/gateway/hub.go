package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/forgebox/forgebox/internal/events"
	"github.com/google/uuid"
)

// outboundMessage is the wire envelope used for every server→client message.
// The shape matches the design spec: a string Type and an optional Payload.
type outboundMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// hubClient is one connected WebSocket. The handler owns the writer goroutine
// that drains send; the hub only enqueues bytes onto send.
type hubClient struct {
	id     string
	userID string
	send   chan []byte
}

// Hub fans events from the EventBus out to all WebSocket clients registered
// for the event's UserID. Each user can have many clients (one per browser
// tab). Clients are kept in send-channel form so the hub never blocks on a
// slow socket — if a client's send channel fills up, the hub closes it.
type Hub struct {
	bus      *events.Bus
	sub      <-chan events.Event
	mu       sync.RWMutex
	clients  map[string]map[string]*hubClient // userID → connID → client
	sendBuf  int
	stopOnce sync.Once
	stopped  chan struct{}
}

// NewHub creates a Hub that fans events from bus to registered clients.
// sendBuf is the per-client outbound channel buffer; 64 is a sensible default.
// Subscription is established synchronously so callers can Publish before
// Run starts without losing events.
func NewHub(bus *events.Bus, sendBuf int) *Hub {
	if sendBuf <= 0 {
		sendBuf = 64
	}
	return &Hub{
		bus:     bus,
		sub:     bus.Subscribe(),
		clients: make(map[string]map[string]*hubClient),
		sendBuf: sendBuf,
		stopped: make(chan struct{}),
	}
}

// Run dispatches subscribed events until ctx is canceled.
func (h *Hub) Run(ctx context.Context) {
	defer h.bus.Unsubscribe(h.sub)

	for {
		select {
		case <-ctx.Done():
			h.stop()
			return
		case ev, ok := <-h.sub:
			if !ok {
				h.stop()
				return
			}
			h.dispatch(ev)
		}
	}
}

func (h *Hub) stop() {
	h.stopOnce.Do(func() {
		close(h.stopped)
		h.mu.Lock()
		for _, conns := range h.clients {
			for _, c := range conns {
				close(c.send)
			}
		}
		h.clients = map[string]map[string]*hubClient{}
		h.mu.Unlock()
	})
}

// register adds a client for a userID and returns the assigned connection ID
// along with the send channel the handler should drain.
func (h *Hub) register(userID string) *hubClient {
	c := &hubClient{
		id:     uuid.New().String(),
		userID: userID,
		send:   make(chan []byte, h.sendBuf),
	}
	h.mu.Lock()
	conns, ok := h.clients[userID]
	if !ok {
		conns = make(map[string]*hubClient)
		h.clients[userID] = conns
	}
	conns[c.id] = c
	h.mu.Unlock()
	return c
}

// unregister removes a client and closes its send channel. Safe to call
// multiple times for the same client.
func (h *Hub) unregister(c *hubClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns, ok := h.clients[c.userID]
	if !ok {
		return
	}
	if existing, ok := conns[c.id]; ok && existing == c {
		delete(conns, c.id)
		close(c.send)
	}
	if len(conns) == 0 {
		delete(h.clients, c.userID)
	}
}

// dispatch encodes ev once and enqueues the bytes for every client of the
// target user. Clients whose send channel is full are evicted: they're slow
// or disconnected and would otherwise stall delivery for everyone.
func (h *Hub) dispatch(ev events.Event) {
	if ev.UserID == "" {
		return
	}
	msg := outboundMessage{Type: ev.Type, Payload: ev.Payload}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("hub: marshal event", "type", ev.Type, "error", err)
		return
	}

	h.mu.RLock()
	conns := h.clients[ev.UserID]
	targets := make([]*hubClient, 0, len(conns))
	for _, c := range conns {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	var slow []*hubClient
	for _, c := range targets {
		select {
		case c.send <- data:
		default:
			slow = append(slow, c)
		}
	}
	for _, c := range slow {
		slog.Warn("hub: slow client evicted", "user_id", c.userID, "conn_id", c.id)
		h.unregister(c)
	}
}
