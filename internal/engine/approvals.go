package engine

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Approvals tracks pending destructive-action approvals keyed by random id.
// The dashboard responds via the WebSocket; the engine's Run loop blocks on
// Await until the reply, a timeout, or context cancellation. See spec 5.4.0.
//
// Memory hygiene: an approval entry lives until Resolve or Cancel is called;
// callers that Register must always pair with one of the two (typically via
// defer Cancel and explicit Resolve from the WS handler).
type Approvals struct {
	mu sync.Mutex
	m  map[string]chan bool
}

// NewApprovals returns an empty registry.
func NewApprovals() *Approvals {
	return &Approvals{m: make(map[string]chan bool)}
}

// Register allocates an id and channel for a new pending approval. The
// channel is buffered (cap 1) so Resolve never blocks even if Await has not
// reached its select yet.
func (a *Approvals) Register() (string, <-chan bool) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Errorf("approvals: crypto/rand: %w", err))
	}
	id := base64.RawURLEncoding.EncodeToString(raw[:])
	ch := make(chan bool, 1)
	a.mu.Lock()
	a.m[id] = ch
	a.mu.Unlock()
	return id, ch
}

// Resolve delivers the user's decision to the waiter and removes the entry.
// Unknown ids are dropped silently (the WS handler may report after timeout).
func (a *Approvals) Resolve(id string, approved bool) {
	a.mu.Lock()
	ch, ok := a.m[id]
	delete(a.m, id)
	a.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- approved:
	default:
	}
}

// Cancel removes a pending approval without delivering. Idempotent.
func (a *Approvals) Cancel(id string) {
	a.mu.Lock()
	delete(a.m, id)
	a.mu.Unlock()
}

// Await blocks until the user decides, the timeout elapses, or ctx is done.
// Per spec 5.4.0: timeout = deny, ctx-cancel = deny.
func (a *Approvals) Await(ctx context.Context, id string, ch <-chan bool, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ok := <-ch:
		return ok
	case <-timer.C:
		a.Cancel(id)
		return false
	case <-ctx.Done():
		a.Cancel(id)
		return false
	}
}
