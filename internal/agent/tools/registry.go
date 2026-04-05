// Package tools contains the tool implementations that run inside Firecracker VMs.
package tools

import (
	"context"
	"encoding/json"
	"sync"
)

// Tool is the interface for in-VM tool implementations.
type Tool interface {
	Name() string
	Execute(ctx context.Context, input json.RawMessage) (*Result, error)
}

// Result is the output of a tool execution.
type Result struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// Registry holds the set of available tools inside the VM.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}
