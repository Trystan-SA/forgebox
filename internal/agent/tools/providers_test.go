package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProviders_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/providers", r.URL.Path)
		assert.Equal(t, "Bearer fbtask_test", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`[{"name":"Anthropic (API)","provider_type":"anthropic-api","builtin":false,"models":[{"id":"claude-opus-4","display_name":"Claude Opus 4"}]}]`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")

	tool := &ListProvidersTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "anthropic-api")
	// list_providers returns name/provider_type/builtin only — should NOT
	// include the models slice in the LLM-facing payload.
	assert.NotContains(t, res.Content, "claude-opus-4")
}

func TestListProviders_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &ListProvidersTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "invalid token")
}

func TestListModelsForProvider_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/providers", r.URL.Path)
		_, _ = w.Write([]byte(`[
			{"name":"OpenAI","provider_type":"openai","builtin":false,"models":[{"id":"gpt-4.1","display_name":"GPT-4.1"}]},
			{"name":"Anthropic (API)","provider_type":"anthropic-api","builtin":false,"models":[{"id":"claude-opus-4","display_name":"Claude Opus 4"}]}
		]`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")

	tool := &ListModelsForProviderTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"provider":"OpenAI"}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "gpt-4.1")
	assert.NotContains(t, res.Content, "claude-opus-4") // wrong provider's models filtered out
}

func TestListModelsForProvider_MissingProvider(t *testing.T) {
	t.Setenv("FORGEBOX_API_URL", "http://unused")
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &ListModelsForProviderTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "provider")
}

func TestListModelsForProvider_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"OpenAI","provider_type":"openai","builtin":false}]`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")

	tool := &ListModelsForProviderTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"provider":"Unknown"}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "not found")
}
