package llmbase

import (
	"context"
	"io"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// EventMapper converts a single raw SSE event into zero or more sdk.StreamEvents.
// Returning a nil slice means "skip this SSE event". An error terminates the stream
// (an EventError is emitted by the pump).
type EventMapper func(ev SSEEvent) ([]sdk.StreamEvent, error)

// StreamPump reads SSE events from r, maps each through mapper, and forwards
// results to a buffered sdk.StreamEvent channel. The channel closes when the
// reader ends, ctx is canceled, or mapper returns an error. The pump owns
// closing r.
//
// On any error, an EventError event is sent before the channel closes.
func StreamPump(ctx context.Context, r io.ReadCloser, mapper EventMapper) <-chan sdk.StreamEvent {
	out := make(chan sdk.StreamEvent, 64)
	go func() {
		defer close(out)
		defer func() { _ = r.Close() }()

		parser := NewSSEParser(r)
		for ev := range parser.Events(ctx) {
			events, err := mapper(ev)
			if err != nil {
				select {
				case <-ctx.Done():
				case out <- sdk.StreamEvent{Type: sdk.EventError, Error: err}:
				}
				return
			}
			for _, sev := range events {
				select {
				case <-ctx.Done():
					return
				case out <- sev:
				}
			}
		}
	}()
	return out
}
