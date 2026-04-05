package sdk

import (
	"context"
	"encoding/json"
)

// ProviderPlugin is the interface for LLM provider integrations.
type ProviderPlugin interface {
	Plugin

	// Models returns the list of models this provider supports.
	Models() []Model

	// Stream sends a request and returns a streaming response.
	// The caller must read from StreamResponse.Events until the channel closes.
	Stream(ctx context.Context, req *CompletionRequest) (*StreamResponse, error)

	// Complete sends a request and returns the full response.
	// Use Stream for real-time output; Complete blocks until done.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}

// Model describes an LLM model available from a provider.
type Model struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	MaxInputTokens  int    `json:"max_input_tokens"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	SupportsTools   bool   `json:"supports_tools"`
	SupportsVision  bool   `json:"supports_vision"`
}

// CompletionRequest is a request to an LLM provider.
type CompletionRequest struct {
	Model        string    `json:"model"`
	Messages     []Message `json:"messages"`
	Tools        []ToolDef `json:"tools,omitempty"`
	MaxTokens    int       `json:"max_tokens,omitempty"`
	Temperature  float64   `json:"temperature,omitempty"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
}

// CompletionResponse is the full result of an LLM call.
type CompletionResponse struct {
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	StopReason string     `json:"stop_reason"`
	Usage      Usage      `json:"usage"`
}

// StreamResponse provides a channel of streaming events from an LLM call.
type StreamResponse struct {
	Events <-chan StreamEvent
}

// StreamEvent is a single event in a streaming LLM response.
type StreamEvent struct {
	Type     StreamEventType `json:"type"`
	Delta    string          `json:"delta,omitempty"`
	ToolCall *ToolCall       `json:"tool_call,omitempty"`
	Usage    *Usage          `json:"usage,omitempty"`
	Error    error           `json:"-"`
}

// StreamEventType identifies the kind of streaming event.
type StreamEventType string

const (
	EventTextDelta StreamEventType = "text_delta"
	EventToolCall  StreamEventType = "tool_call"
	EventDone      StreamEventType = "done"
	EventError     StreamEventType = "error"
)

// Message represents a conversation message.
type Message struct {
	Role        string       `json:"role"` // "user", "assistant", "system"
	Content     string       `json:"content,omitempty"`
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
}

// ToolCall represents an LLM's request to invoke a tool.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult is the output of a tool execution.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// ToolDef describes a tool's schema for the LLM.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Usage tracks token consumption for a completion.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
