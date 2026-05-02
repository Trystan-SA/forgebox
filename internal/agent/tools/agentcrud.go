// Package tools — agentcrud.go provides the in-VM tools that manage
// ForgeBox agents by calling the gateway's /api/v1/agents API with a
// per-task fbtask_… token. See specs/5.0.0-management-tools.md.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type agentAPIClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func newAgentAPIClient() (*agentAPIClient, error) {
	base := os.Getenv("FORGEBOX_API_URL")
	tok := os.Getenv("FORGEBOX_API_TOKEN")
	if base == "" || tok == "" {
		return nil, fmt.Errorf("FORGEBOX_API_URL and FORGEBOX_API_TOKEN must be set")
	}
	return &agentAPIClient{
		baseURL: strings.TrimRight(base, "/"),
		token:   tok,
		client:  &http.Client{},
	}, nil
}

func (c *agentAPIClient) do(ctx context.Context, method, path string, body any) (respBody []byte, status int, err error) {
	var buf io.Reader
	if body != nil {
		var b []byte
		b, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %w", err)
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, buf)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		// Scrub the URL from any error string returned to the LLM, since the
		// Authorization header is on the request and a poorly-formatted error
		// could leak it. The transport never includes the header in err itself
		// today, but be defensive.
		return nil, 0, fmt.Errorf("api call %s %s: transport error", method, path)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ = io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// ListAgentsTool returns the agents visible to the calling user.
type ListAgentsTool struct{}

// Name returns the tool identifier.
func (t *ListAgentsTool) Name() string { return "list_agents" }

// Execute calls GET /api/v1/agents and returns the raw JSON response.
func (t *ListAgentsTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodGet, "/api/v1/agents", nil)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	return &Result{Content: string(body)}, nil
}

// GetAgentTool returns one agent by id.
type GetAgentTool struct{}

// Name returns the tool identifier.
func (t *GetAgentTool) Name() string { return "get_agent" }

// Execute calls GET /api/v1/agents/{id} and returns the raw JSON response.
func (t *GetAgentTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(input, &args); err != nil || args.ID == "" {
		return &Result{Content: "missing required field: id", IsError: true}, nil
	}
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodGet, "/api/v1/agents/"+args.ID, nil)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	return &Result{Content: string(body)}, nil
}

// CreateAgentTool creates a new agent. The input is forwarded to the gateway
// verbatim; the gateway enforces validation per spec 1.2.3.
type CreateAgentTool struct{}

// Name returns the tool identifier.
func (t *CreateAgentTool) Name() string { return "create_agent" }

// Execute calls POST /api/v1/agents with the input as the request body.
func (t *CreateAgentTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodPost, "/api/v1/agents", input)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	return &Result{Content: string(body)}, nil
}

// UpdateAgentTool patches an existing agent.
type UpdateAgentTool struct{}

// Name returns the tool identifier.
func (t *UpdateAgentTool) Name() string { return "update_agent" }

// Execute calls PUT /api/v1/agents/{id} with the remaining input fields as the
// request body. The id is taken from the input and removed before forwarding.
func (t *UpdateAgentTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var args map[string]json.RawMessage
	if err := json.Unmarshal(input, &args); err != nil {
		return &Result{Content: "invalid input: " + err.Error(), IsError: true}, nil
	}
	idRaw, ok := args["id"]
	if !ok {
		return &Result{Content: "missing required field: id", IsError: true}, nil
	}
	var id string
	if err := json.Unmarshal(idRaw, &id); err != nil || id == "" {
		return &Result{Content: "missing required field: id", IsError: true}, nil
	}
	delete(args, "id")
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodPut, "/api/v1/agents/"+id, args)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	return &Result{Content: string(body)}, nil
}

// DeleteAgentTool permanently deletes an agent. On success returns
// {deleted:true,id:<id>}; the gateway's response body is ignored.
type DeleteAgentTool struct{}

// Name returns the tool identifier.
func (t *DeleteAgentTool) Name() string { return "delete_agent" }

// Execute calls DELETE /api/v1/agents/{id} and returns a synthetic confirmation
// envelope on success.
func (t *DeleteAgentTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(input, &args); err != nil || args.ID == "" {
		return &Result{Content: "missing required field: id", IsError: true}, nil
	}
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodDelete, "/api/v1/agents/"+args.ID, nil)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	out, _ := json.Marshal(map[string]any{
		"deleted": true,
		"id":      args.ID,
	})
	return &Result{Content: string(out)}, nil
}
