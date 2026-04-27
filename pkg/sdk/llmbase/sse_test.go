package llmbase

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSEParser_Stream(t *testing.T) {
	stream := "event: message_start\n" +
		"data: {\"id\":\"abc\"}\n" +
		"\n" +
		"event: content_block_delta\n" +
		"data: {\"delta\":{\"text\":\"hello\"}}\n" +
		"\n" +
		"event: content_block_delta\n" +
		"data: {\"delta\":{\"text\":\" world\"}}\n" +
		"\n" +
		"event: message_stop\n" +
		"data: {}\n" +
		"\n"

	p := NewSSEParser(strings.NewReader(stream))
	ch := p.Events(context.Background())

	var got []SSEEvent
	for ev := range ch {
		got = append(got, ev)
	}
	require.Len(t, got, 4)
	require.Equal(t, "message_start", got[0].Event)
	require.JSONEq(t, `{"id":"abc"}`, string(got[0].Data))
	require.Equal(t, "content_block_delta", got[1].Event)
	require.Equal(t, "message_stop", got[3].Event)
}

func TestSSEParser_MultilineData(t *testing.T) {
	stream := "event: x\ndata: line1\ndata: line2\n\n"
	p := NewSSEParser(strings.NewReader(stream))
	ch := p.Events(context.Background())
	ev := <-ch
	require.Equal(t, "x", ev.Event)
	require.Equal(t, "line1\nline2", string(ev.Data))
}

func TestSSEParser_IgnoresComments(t *testing.T) {
	stream := ": this is a comment\nevent: ping\ndata: ok\n\n"
	p := NewSSEParser(strings.NewReader(stream))
	ev := <-p.Events(context.Background())
	require.Equal(t, "ping", ev.Event)
	require.Equal(t, "ok", string(ev.Data))
}
