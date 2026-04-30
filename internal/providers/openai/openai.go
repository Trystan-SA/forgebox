// Package openai implements the OpenAI GPT LLM provider.
package openai

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

const baseURL = "https://api.openai.com/v1"

// Provider implements sdk.ProviderPlugin for OpenAI.
type Provider struct {
	apiKey     string
	httpClient *http.Client
}

func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Name() string    { return "openai" }
func (p *Provider) Version() string { return "1.0.0" }

func (p *Provider) Init(_ context.Context, config map[string]any) error {
	key, ok := config["api_key"].(string)
	if !ok || key == "" {
		return fmt.Errorf("openai: api_key is required")
	}
	p.apiKey = key
	return nil
}

func (p *Provider) Shutdown(_ context.Context) error { return nil }

func (p *Provider) Models() []sdk.Model {
	return []sdk.Model{
		{ID: "gpt-4.1", Name: "GPT-4.1", MaxInputTokens: 1047576, MaxOutputTokens: 32768, SupportsTools: true, SupportsVision: true},
		{ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", MaxInputTokens: 1047576, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
		{ID: "gpt-4.1-nano", Name: "GPT-4.1 Nano", MaxInputTokens: 1047576, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: false},
		{ID: "o3", Name: "o3", MaxInputTokens: 200000, MaxOutputTokens: 100000, SupportsTools: true, SupportsVision: true},
	}
}

func (p *Provider) Complete(ctx context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	apiReq := p.buildRequest(req)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return nil, fmt.Errorf("openai API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp openaiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return p.convertResponse(&apiResp), nil
}

func (p *Provider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	// For now, use Complete and wrap as a single-event stream.
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	events := make(chan sdk.StreamEvent, 4)
	go func() {
		defer close(events)
		if resp.Content != "" {
			events <- sdk.StreamEvent{Type: sdk.EventTextDelta, Delta: resp.Content}
		}
		for _, tc := range resp.ToolCalls {
			events <- sdk.StreamEvent{Type: sdk.EventToolCall, ToolCall: &tc}
		}
		events <- sdk.StreamEvent{
			Type:  sdk.EventDone,
			Usage: &resp.Usage,
		}
	}()

	return &sdk.StreamResponse{Events: events}, nil
}

// --- OpenAI API types ---

type openaiRequest struct {
	Model    string         `json:"model"`
	Messages []openaiMsg    `json:"messages"`
	Tools    []openaiTool   `json:"tools,omitempty"`
	MaxTokens int           `json:"max_completion_tokens,omitempty"`
}

type openaiMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiTool struct {
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *Provider) buildRequest(req *sdk.CompletionRequest) *openaiRequest {
	model := req.Model
	if model == "" {
		model = "gpt-4.1"
	}

	msgs := make([]openaiMsg, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		msgs = append(msgs, openaiMsg{Role: "system", Content: req.SystemPrompt})
	}
	for _, m := range req.Messages {
		content := m.Content
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				content += tr.Content + "\n"
			}
		}
		msgs = append(msgs, openaiMsg{Role: m.Role, Content: content})
	}

	var tools []openaiTool
	for _, t := range req.Tools {
		tools = append(tools, openaiTool{
			Type: "function",
			Function: openaiFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	return &openaiRequest{
		Model:     model,
		Messages:  msgs,
		Tools:     tools,
		MaxTokens: req.MaxTokens,
	}
}

func (p *Provider) convertResponse(resp *openaiResponse) *sdk.CompletionResponse {
	if len(resp.Choices) == 0 {
		return &sdk.CompletionResponse{StopReason: "error"}
	}

	choice := resp.Choices[0]
	var toolCalls []sdk.ToolCall
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, sdk.ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	return &sdk.CompletionResponse{
		Content:    choice.Message.Content,
		ToolCalls:  toolCalls,
		StopReason: choice.FinishReason,
		Usage: sdk.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
}
