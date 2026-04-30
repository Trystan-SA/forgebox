package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase"
)

// streamEnvelope is the common structure of every SSE event from
// /v1/messages with stream=true. Field presence depends on Type.
type streamEnvelope struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Stream sends a streaming /v1/messages call.
func (p *Provider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	apiReq := p.BuildRequest(req)
	apiReq.Stream = true
	if p.gate != nil {
		p.gate(apiReq)
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal stream request: %w", err)
	}

	headers := p.headers("")
	headers.Set("Accept", "text/event-stream")

	resp, err := p.runner.Do(ctx, p.baseURL+"/messages", headers, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("anthropic stream: %w", llmbase.ClassifyHTTPError(resp.StatusCode, respBody))
	}

	mapper := newAnthropicEventMapper()
	events := llmbase.StreamPump(ctx, resp.Body, mapper)
	return &sdk.StreamResponse{Events: events}, nil
}

// newAnthropicEventMapper returns an EventMapper that maps Anthropic SSE
// events to sdk.StreamEvent values. It accumulates tool_use input deltas
// across content_block_delta events and emits a single EventToolCall on
// content_block_stop.
func newAnthropicEventMapper() llmbase.EventMapper {
	type pendingTool struct {
		id, name string
		input    []byte
	}
	tools := map[int]*pendingTool{}

	return func(ev llmbase.SSEEvent) ([]sdk.StreamEvent, error) {
		var env streamEnvelope
		if len(ev.Data) == 0 {
			return nil, nil
		}
		if err := json.Unmarshal(ev.Data, &env); err != nil {
			return nil, fmt.Errorf("unmarshal stream event %q: %w", ev.Event, err)
		}
		switch env.Type {
		case "content_block_start":
			if env.ContentBlock.Type == "tool_use" {
				tools[env.Index] = &pendingTool{id: env.ContentBlock.ID, name: env.ContentBlock.Name}
			}
			return nil, nil
		case "content_block_delta":
			switch env.Delta.Type {
			case "text_delta":
				return []sdk.StreamEvent{{Type: sdk.EventTextDelta, Delta: env.Delta.Text}}, nil
			case "input_json_delta":
				if t, ok := tools[env.Index]; ok {
					t.input = append(t.input, env.Delta.PartialJSON...)
				}
				return nil, nil
			}
			return nil, nil
		case "content_block_stop":
			t, ok := tools[env.Index]
			if !ok {
				return nil, nil
			}
			delete(tools, env.Index)
			input := t.input
			if len(input) == 0 {
				input = []byte("{}")
			}
			return []sdk.StreamEvent{{
				Type:     sdk.EventToolCall,
				ToolCall: &sdk.ToolCall{ID: t.id, Name: t.name, Input: input},
			}}, nil
		case "message_stop":
			return []sdk.StreamEvent{{Type: sdk.EventDone}}, nil
		}
		return nil, nil
	}
}
