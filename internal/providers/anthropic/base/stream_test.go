package base

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

const sampleSSE = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"x\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hel\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"lo\"}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

func TestStream_TextDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleSSE))
	}))
	defer srv.Close()

	b := New(Options{Auth: auth.NewAPIKey("x-api-key", "k"), BaseURL: srv.URL})
	resp, err := b.Stream(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)

	var deltas []string
	var sawDone bool
	for ev := range resp.Events {
		switch ev.Type {
		case sdk.EventTextDelta:
			deltas = append(deltas, ev.Delta)
		case sdk.EventDone:
			sawDone = true
		case sdk.EventError:
			t.Fatalf("unexpected error event: %v", ev.Error)
		}
	}
	require.Equal(t, []string{"hel", "lo"}, deltas)
	require.True(t, sawDone, "must end with EventDone")
}
