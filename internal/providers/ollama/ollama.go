// Package ollama implements the Ollama local LLM provider.
package ollama

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

// Provider implements sdk.ProviderPlugin for Ollama.
type Provider struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *Provider) Name() string    { return "ollama" }
func (p *Provider) Version() string { return "1.0.0" }

func (p *Provider) Init(_ context.Context, config map[string]any) error {
	p.baseURL = "http://localhost:11434"
	if url, ok := config["base_url"].(string); ok && url != "" {
		p.baseURL = url
	}
	return nil
}

func (p *Provider) Shutdown(_ context.Context) error { return nil }

func (p *Provider) Models() []sdk.Model {
	return []sdk.Model{
		{ID: "llama3.3", Name: "Llama 3.3 70B", MaxInputTokens: 128000, MaxOutputTokens: 4096, SupportsTools: true},
		{ID: "qwen3", Name: "Qwen 3", MaxInputTokens: 128000, MaxOutputTokens: 4096, SupportsTools: true},
		{ID: "codestral", Name: "Codestral", MaxInputTokens: 32000, MaxOutputTokens: 4096, SupportsTools: false},
	}
}

func (p *Provider) Complete(ctx context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "llama3.3"
	}

	prompt := req.SystemPrompt + "\n"
	for _, m := range req.Messages {
		prompt += m.Role + ": " + m.Content + "\n"
	}

	apiReq := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &sdk.CompletionResponse{
		Content:    ollamaResp.Response,
		StopReason: "end_turn",
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	events := make(chan sdk.StreamEvent, 2)
	go func() {
		defer close(events)
		events <- sdk.StreamEvent{Type: sdk.EventTextDelta, Delta: resp.Content}
		events <- sdk.StreamEvent{Type: sdk.EventDone}
	}()

	return &sdk.StreamResponse{Events: events}, nil
}
