// Package base holds the shared Anthropic /v1/messages wire protocol used by
// the anthropic-api and anthropic-subscription providers.
package base

import "encoding/json"

// Request is the JSON payload sent to /v1/messages.
type Request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	System    string    `json:"system,omitempty"`
	MaxTokens int       `json:"max_tokens"`
	Tools     []Tool    `json:"tools,omitempty"`
	Stream    bool      `json:"stream,omitempty"`
	// Extras carries ad-hoc fields (e.g. an inbound anthropic-beta value
	// the gate hook needs to inspect). Not marshaled to the wire.
	Extras map[string]any `json:"-"`
}

// Message is one item in the messages array.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Tool is a tool definition.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Response is the non-streaming /v1/messages response.
type Response struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage      ResponseUsage  `json:"usage"`
}

// ContentBlock represents one item in the response content array.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ResponseUsage carries token counts.
type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
