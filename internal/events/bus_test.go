package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBus_PublishDeliversToAllSubscribers(t *testing.T) {
	b := New(8)

	a := b.Subscribe()
	c := b.Subscribe()

	ev := Event{Type: "task.updated", UserID: "u1", Payload: map[string]string{"id": "t1"}}
	b.Publish(ev)

	got1 := receiveOrFail(t, a)
	got2 := receiveOrFail(t, c)
	assert.Equal(t, ev, got1)
	assert.Equal(t, ev, got2)
}

func TestBus_UnsubscribeStopsDeliveryAndClosesChannel(t *testing.T) {
	b := New(4)
	ch := b.Subscribe()

	b.Unsubscribe(ch)

	// Channel should be closed.
	_, ok := <-ch
	assert.False(t, ok, "expected channel to be closed after Unsubscribe")

	// Subsequent publishes do not panic on the unsubscribed channel.
	assert.NotPanics(t, func() {
		b.Publish(Event{Type: "x"})
	})
}

func TestBus_PublishDoesNotBlockWhenSubscriberFull(t *testing.T) {
	b := New(1)
	ch := b.Subscribe()

	// Fill the buffer.
	b.Publish(Event{Type: "first"})

	// Second publish would block if delivery were synchronous; it must drop.
	done := make(chan struct{})
	go func() {
		b.Publish(Event{Type: "second"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Publish blocked when subscriber channel was full")
	}

	// First event still readable; second was dropped.
	got := receiveOrFail(t, ch)
	assert.Equal(t, "first", got.Type)
	select {
	case ev := <-ch:
		t.Fatalf("expected dropped second event, got %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBus_NewDefaultsBufferSizeWhenNonPositive(t *testing.T) {
	b := New(0)
	require.NotNil(t, b)
	assert.Equal(t, 256, b.bufSize)
}

func TestBus_ConcurrentPublishAndSubscribe(t *testing.T) {
	b := New(64)
	const subscribers = 4
	const events = 32

	subs := make([]<-chan Event, subscribers)
	for i := 0; i < subscribers; i++ {
		subs[i] = b.Subscribe()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < events; i++ {
			b.Publish(Event{Type: "task.updated"})
		}
	}()
	wg.Wait()

	for _, ch := range subs {
		count := 0
		for count < events {
			select {
			case <-ch:
				count++
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("subscriber received only %d/%d events", count, events)
			}
		}
	}
}

func receiveOrFail(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}
