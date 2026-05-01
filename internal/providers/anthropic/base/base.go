package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	apiVersion     = "2023-06-01"
	defaultModel   = "claude-sonnet-4-6"
	defaultMaxTok  = 4096
)

// Options configures a Provider.
type Options struct {
	Auth    auth.Auth     // required
	Betas   []string      // default-empty; merged into anthropic-beta header
	BaseURL string        // optional; defaults to https://api.anthropic.com/v1
	Timeout time.Duration // optional; defaults to 120s
	// GateRequest is an optional hook called after BuildRequest. The
	// subscription provider uses it to strip disallowed fields. May be nil.
	GateRequest func(*Request)
}

// Provider implements the Anthropic /v1/messages call shape. Embed it in a
// concrete provider (anthropic-api, anthropic-subscription) which adds Plugin
// metadata and Models().
type Provider struct {
	auth    auth.Auth
	betas   []string
	baseURL string
	gate    func(*Request)
	runner  *llmbase.HTTPRunner
}

// New constructs a base Provider.
func New(opts Options) *Provider {
	url := opts.BaseURL
	if url == "" {
		url = defaultBaseURL
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &Provider{
		auth:    opts.Auth,
		betas:   opts.Betas,
		baseURL: url,
		gate:    opts.GateRequest,
		runner:  llmbase.NewHTTPRunner(llmbase.HTTPOptions{Timeout: timeout}),
	}
}

// BuildRequest converts an sdk.CompletionRequest to the Anthropic wire format.
// System messages in the messages array are dropped; the SystemPrompt field is
// hoisted to the top-level "system" key.
func (p *Provider) BuildRequest(req *sdk.CompletionRequest) *Request {
	model := req.Model
	if model == "" {
		model = defaultModel
	}
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = defaultMaxTok
	}

	msgs := make([]Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		raw, _ := json.Marshal(m.Content)
		msgs = append(msgs, Message{Role: m.Role, Content: raw})
	}

	tools := make([]Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		// Anthropic rejects tools whose input_schema is missing or not a
		// valid object schema (HTTP 400 "tools.0.custom.input_schema: Input
		// does not match the expected shape"). Fall back to a valid
		// empty-object schema when the plugin didn't supply one.
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	return &Request{
		Model:     model,
		Messages:  msgs,
		System:    req.SystemPrompt,
		MaxTokens: maxTok,
		Tools:     tools,
	}
}

// Complete sends a non-streaming /v1/messages call.
func (p *Provider) Complete(ctx context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	apiReq := p.BuildRequest(req)
	if p.gate != nil {
		p.gate(apiReq)
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.runner.Do(ctx, p.baseURL+"/messages", p.headers(""), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic complete: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if cerr := llmbase.ClassifyHTTPError(resp.StatusCode, respBody); cerr != nil {
		return nil, fmt.Errorf("anthropic complete: %w", cerr)
	}

	var ar Response
	if err := json.Unmarshal(respBody, &ar); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return p.convertResponse(&ar), nil
}

func (p *Provider) headers(extraBeta string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("anthropic-version", apiVersion)
	if beta := MergeBetaHeader(extraBeta, p.betas); beta != "" {
		h.Set("anthropic-beta", beta)
	}
	stub := &http.Request{Header: h}
	p.auth.Apply(stub)
	return stub.Header
}

func (p *Provider) convertResponse(ar *Response) *sdk.CompletionResponse {
	var content string
	var toolCalls []sdk.ToolCall
	for _, c := range ar.Content {
		switch c.Type {
		case "text":
			content += c.Text
		case "tool_use":
			toolCalls = append(toolCalls, sdk.ToolCall{ID: c.ID, Name: c.Name, Input: c.Input})
		}
	}
	return &sdk.CompletionResponse{
		Content:    content,
		ToolCalls:  toolCalls,
		StopReason: ar.StopReason,
		Usage: sdk.Usage{
			InputTokens:  ar.Usage.InputTokens,
			OutputTokens: ar.Usage.OutputTokens,
			TotalTokens:  ar.Usage.InputTokens + ar.Usage.OutputTokens,
		},
	}
}
