// Package anthropic implements the Anthropic Claude LLM provider.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
)

const baseURL = "https://api.anthropic.com/v1"

// Provider implements sdk.ProviderPlugin for Anthropic's Claude API.
type Provider struct {
	apiKey     string
	httpClient *http.Client
}

// New creates a new Anthropic provider with default HTTP settings.
func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "anthropic" }

// Version returns the provider version.
func (p *Provider) Version() string { return "1.0.0" }

// Init configures the provider with the given settings.
func (p *Provider) Init(_ context.Context, config map[string]any) error {
	key, ok := config["api_key"].(string)
	if !ok || key == "" {
		return fmt.Errorf("anthropic: api_key is required")
	}
	p.apiKey = key
	return nil
}

// Shutdown is a no-op for the HTTP-based Anthropic provider.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models returns the list of supported Anthropic models.
func (p *Provider) Models() []sdk.Model { return Models() }

// Complete sends a completion request to the Anthropic API.
func (p *Provider) Complete(ctx context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	apiReq := p.buildRequest(req)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return p.convertResponse(&apiResp), nil
}

// Stream sends a streaming completion request to the Anthropic API.
func (p *Provider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	apiReq := p.buildRequest(req)
	apiReq.Stream = true

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(httpReq) //nolint:bodyclose // closed inside the goroutine below
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("anthropic API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	events := make(chan sdk.StreamEvent, 64)

	go func() {
		defer close(events)
		defer func() { _ = resp.Body.Close() }()
		// TODO: Parse SSE stream from Anthropic API.
		// For now, read full response and emit as single event.
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			events <- sdk.StreamEvent{Type: sdk.EventError, Error: err}
			return
		}
		events <- sdk.StreamEvent{Type: sdk.EventTextDelta, Delta: string(respBody)}
		events <- sdk.StreamEvent{Type: sdk.EventDone}
	}()

	return &sdk.StreamResponse{Events: events}, nil
}

// --- Anthropic API types ---

type anthropicRequest struct {
	Model     string          `json:"model"`
	Messages  []anthropicMsg  `json:"messages"`
	System    string          `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens"`
	Tools     []anthropicTool `json:"tools,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *Provider) buildRequest(req *sdk.CompletionRequest) *anthropicRequest {
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	msgs := make([]anthropicMsg, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		content := m.Content
		// For tool results, format them as text.
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				content += tr.Content + "\n"
			}
		}
		msgs = append(msgs, anthropicMsg{Role: m.Role, Content: content})
	}

	tools := make([]anthropicTool, 0, len(req.Tools))
	for _, t := range req.Tools {
		// Anthropic rejects tools with a missing or non-object input_schema
		// (HTTP 400: "tools.0.custom.input_schema: Input does not match the
		// expected shape"), so fall back to a valid empty-object schema if
		// the plugin didn't supply one.
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	return &anthropicRequest{
		Model:     model,
		Messages:  msgs,
		System:    req.SystemPrompt,
		MaxTokens: maxTokens,
		Tools:     tools,
	}
}

func (p *Provider) convertResponse(resp *anthropicResponse) *sdk.CompletionResponse {
	var content string
	var toolCalls []sdk.ToolCall

	for _, c := range resp.Content {
		switch c.Type {
		case "text":
			content += c.Text
		case "tool_use":
			toolCalls = append(toolCalls, sdk.ToolCall{
				ID:    c.ID,
				Name:  c.Name,
				Input: c.Input,
			})
		}
	}

	return &sdk.CompletionResponse{
		Content:    content,
		ToolCalls:  toolCalls,
		StopReason: resp.StopReason,
		Usage: sdk.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}
