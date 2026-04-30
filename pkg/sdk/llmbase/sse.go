package llmbase

import (
	"bufio"
	"context"
	"io"
	"strings"
)

// SSEEvent is one parsed Server-Sent Event with its name and raw data payload.
// Multi-line data fields are joined with "\n".
type SSEEvent struct {
	Event string
	Data  []byte
}

// SSEParser reads an io.Reader producing SSE-formatted bytes and surfaces
// events on a channel. The reader is consumed in a goroutine; close it from
// outside to stop, or cancel the context passed to Events().
type SSEParser struct {
	r io.Reader
}

// NewSSEParser wraps an io.Reader.
func NewSSEParser(r io.Reader) *SSEParser {
	return &SSEParser{r: r}
}

// Events returns a buffered channel emitting parsed SSE events. The channel
// closes when the underlying reader hits EOF, the context is canceled, or a
// fatal scanner error occurs.
func (p *SSEParser) Events(ctx context.Context) <-chan SSEEvent {
	out := make(chan SSEEvent, 32)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(p.r)
		// Allow large lines: provider tokens can be 1MB+ in a single delta.
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

		var event string
		var dataParts []string

		flush := func() {
			if event == "" && len(dataParts) == 0 {
				return
			}
			ev := SSEEvent{Event: event, Data: []byte(strings.Join(dataParts, "\n"))}
			select {
			case <-ctx.Done():
			case out <- ev:
			}
			event = ""
			dataParts = nil
		}

		for sc.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := sc.Text()
			switch {
			case line == "":
				flush()
			case strings.HasPrefix(line, ":"):
				// comment, ignore
			case strings.HasPrefix(line, "event:"):
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataParts = append(dataParts, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		flush()
	}()
	return out
}
