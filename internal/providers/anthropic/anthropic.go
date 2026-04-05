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

func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string    { return "anthropic" }
func (p *Provider) Version() string { return "1.0.0" }

func (p *Provider) Init(_ context.Context, config map[string]any) error {
	key, ok := config["api_key"].(string)
	if !ok || key == "" {
		return fmt.Errorf("anthropic: api_key is required")
	}
	p.apiKey = key
	return nil
}

func (p *Provider) Shutdown(_ context.Context) error { return nil }

func (p *Provider) Models() []sdk.Model {
	return []sdk.Model{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
		{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", MaxInputTokens: 200000, MaxOutputTokens: 8192, SupportsTools: true, SupportsVision: true},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
	}
}

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
	defer resp.Body.Close()

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

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	events := make(chan sdk.StreamEvent, 64)

	go func() {
		defer close(events)
		defer resp.Body.Close()
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
	Model     string            `json:"model"`
	Messages  []anthropicMsg    `json:"messages"`
	System    string            `json:"system,omitempty"`
	MaxTokens int               `json:"max_tokens"`
	Tools     []anthropicTool   `json:"tools,omitempty"`
	Stream    bool              `json:"stream,omitempty"`
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

	var tools []anthropicTool
	for _, t := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
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
