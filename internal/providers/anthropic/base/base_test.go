package base

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

func TestBuildRequest(t *testing.T) {
	b := New(Options{
		Auth:    auth.NewAPIKey("x-api-key", "k"),
		Betas:   APIKeyBetas,
		BaseURL: "http://example.invalid",
	})

	req := &sdk.CompletionRequest{
		Model:        "claude-sonnet-4-6",
		SystemPrompt: "you are helpful",
		Messages: []sdk.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello"},
			{Role: "system", Content: "ignored"},
		},
		MaxTokens: 1000,
		Tools: []sdk.ToolDef{{
			Name: "echo", Description: "echo", InputSchema: map[string]any{"type": "object"},
		}},
	}
	out := b.BuildRequest(req)
	require.Equal(t, "claude-sonnet-4-6", out.Model)
	require.Equal(t, "you are helpful", out.System)
	require.Equal(t, 1000, out.MaxTokens)
	require.Len(t, out.Messages, 2, "system messages must be excluded")
	require.Equal(t, "user", out.Messages[0].Role)
	require.Len(t, out.Tools, 1)
	require.Equal(t, "echo", out.Tools[0].Name)
}

func TestBuildRequest_Defaults(t *testing.T) {
	b := New(Options{Auth: auth.NewAPIKey("x-api-key", "k"), BaseURL: "http://x"})
	out := b.BuildRequest(&sdk.CompletionRequest{Messages: []sdk.Message{{Role: "user", Content: "hi"}}})
	require.Equal(t, "claude-sonnet-4-6", out.Model, "default model")
	require.Equal(t, 4096, out.MaxTokens, "default max tokens")
}

func TestComplete_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "k", r.Header.Get("x-api-key"))
		require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		require.Contains(t, r.Header.Get("anthropic-beta"), BetaFineGrainedToolStreaming)
		body, _ := io.ReadAll(r.Body)
		var rq Request
		require.NoError(t, json.Unmarshal(body, &rq))
		require.Equal(t, "claude-sonnet-4-6", rq.Model)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"x","content":[{"type":"text","text":"hello"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":2}}`))
	}))
	defer srv.Close()

	b := New(Options{
		Auth:    auth.NewAPIKey("x-api-key", "k"),
		Betas:   APIKeyBetas,
		BaseURL: srv.URL,
		Timeout: 2 * time.Second,
	})

	resp, err := b.Complete(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, "hello", resp.Content)
	require.Equal(t, "end_turn", resp.StopReason)
	require.Equal(t, 1, resp.Usage.InputTokens)
	require.Equal(t, 2, resp.Usage.OutputTokens)
	require.Equal(t, 3, resp.Usage.TotalTokens)
}

func TestComplete_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer srv.Close()

	b := New(Options{Auth: auth.NewAPIKey("x-api-key", "bad"), BaseURL: srv.URL})
	_, err := b.Complete(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, sdk.ErrAuth)
}

func TestComplete_GateRequest(t *testing.T) {
	var seenBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{}}`))
	}))
	defer srv.Close()

	b := New(Options{
		Auth:    auth.NewAPIKey("x-api-key", "k"),
		BaseURL: srv.URL,
		GateRequest: func(rq *Request) {
			rq.System = ""
		},
	})
	_, err := b.Complete(context.Background(), &sdk.CompletionRequest{
		Messages:     []sdk.Message{{Role: "user", Content: "hi"}},
		SystemPrompt: "should be stripped by gate",
	})
	require.NoError(t, err)
	require.NotContains(t, string(seenBody), "should be stripped by gate")
}
