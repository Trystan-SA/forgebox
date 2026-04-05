package sdk

import (
	"context"
	"encoding/json"
)

// ToolPlugin is the interface for tool implementations.
//
// Tools are the actions an AI agent can perform: running shell commands,
// reading files, searching code, fetching web pages, etc.
type ToolPlugin interface {
	Plugin

	// Schema returns the tool's JSON Schema definition for the LLM.
	Schema() ToolSchema

	// Execute runs the tool with the given input and returns the result.
	Execute(ctx context.Context, input json.RawMessage) (*ToolExecResult, error)

	// IsReadOnly returns true if the given input would not modify any state.
	IsReadOnly(input json.RawMessage) bool

	// IsDestructive returns true if the given input could cause irreversible changes.
	IsDestructive(input json.RawMessage) bool
}

// ToolSchema describes a tool for the LLM to understand and invoke.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// ToolExecResult is the output of a tool execution.
type ToolExecResult struct {
	// Content is the text output of the tool.
	Content string `json:"content"`

	// IsError indicates the tool execution failed.
	IsError bool `json:"is_error"`

	// Metadata contains optional structured data about the execution.
	Metadata map[string]any `json:"metadata,omitempty"`
}
