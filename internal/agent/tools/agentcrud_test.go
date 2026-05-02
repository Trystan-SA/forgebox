package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAgents_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/agents", r.URL.Path)
		assert.Equal(t, "Bearer fbtask_test", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`[{"id":"a-1","name":"Helper"}]`))
	}))
	defer srv.Close()

	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")

	tool := &ListAgentsTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, `"id":"a-1"`)
}

func TestListAgents_MissingEnvIsError(t *testing.T) {
	t.Setenv("FORGEBOX_API_URL", "")
	t.Setenv("FORGEBOX_API_TOKEN", "")
	tool := &ListAgentsTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestListAgents_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &ListAgentsTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "invalid token")
}

func TestGetAgent_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/agents/a-42", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"a-42","name":"Helper"}`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &GetAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"a-42"}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, `"a-42"`)
}

func TestGetAgent_MissingID(t *testing.T) {
	t.Setenv("FORGEBOX_API_URL", "http://unused")
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &GetAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content, "id")
}

func TestCreateAgent_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/agents", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"name":"My Agent"`)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"a-99","name":"My Agent"}`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &CreateAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"name":"My Agent","sharing":"personal"}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, `"a-99"`)
}

func TestUpdateAgent_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/v1/agents/a-7", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), `"name":"Renamed"`)
		assert.NotContains(t, string(body), `"id"`) // id stays in URL, not body
		_, _ = w.Write([]byte(`{"id":"a-7","name":"Renamed"}`))
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &UpdateAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"a-7","name":"Renamed"}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "Renamed")
}

func TestUpdateAgent_MissingID(t *testing.T) {
	t.Setenv("FORGEBOX_API_URL", "http://unused")
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &UpdateAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"name":"X"}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestDeleteAgent_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v1/agents/a-13", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("FORGEBOX_API_URL", srv.URL)
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &DeleteAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"a-13"}`))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.JSONEq(t, `{"deleted":true,"id":"a-13"}`, res.Content)
}

func TestDeleteAgent_MissingID(t *testing.T) {
	t.Setenv("FORGEBOX_API_URL", "http://unused")
	t.Setenv("FORGEBOX_API_TOKEN", "fbtask_test")
	tool := &DeleteAgentTool{}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
