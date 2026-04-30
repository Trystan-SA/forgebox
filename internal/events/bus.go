// Package events provides an in-process typed event bus used to decouple
// event producers (engine, sessions, etc.) from event consumers (the
// WebSocket hub, future audit/notification subscribers).
package events

import (
	"sync"
)

// Event is the unit of delivery on the bus. Type is a routing key like
// "task.updated" or "notification". UserID is the audience: only the hub
// connections registered under that ID will receive the event. Payload is
// the type-specific body, marshaled to JSON before delivery.
type Event struct {
	Type    string `json:"type"`
	UserID  string `json:"-"`
	Payload any    `json:"payload,omitempty"`
}

// Bus is a fan-out event bus. Publishers call Publish; subscribers receive
// every published event on the channel returned by Subscribe.
//
// Publish never blocks: if a subscriber's channel is full the event is
// dropped for that subscriber. Subscribers are expected to drain their
// channels promptly and size their buffers to absorb expected bursts.
type Bus struct {
	mu          sync.RWMutex
	subscribers []chan Event
	bufSize     int
}

// New returns a Bus where each new subscriber gets a channel buffered to
// bufSize events. A reasonable default is 256.
func New(bufSize int) *Bus {
	if bufSize <= 0 {
		bufSize = 256
	}
	return &Bus{bufSize: bufSize}
}

// Subscribe returns a channel that receives every event published after
// the call. Pass the returned channel to Unsubscribe when done.
func (b *Bus) Subscribe() <-chan Event {
	ch := make(chan Event, b.bufSize)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a previously subscribed channel and closes it.
// It is safe to call with a channel that is not (or no longer) registered.
func (b *Bus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, sub := range b.subscribers {
		if sub == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(sub)
			return
		}
	}
}

// Publish delivers ev to every subscriber. Delivery is non-blocking:
// subscribers whose channels are full miss the event.
func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subscribers {
		select {
		case sub <- ev:
		default:
		}
	}
}
