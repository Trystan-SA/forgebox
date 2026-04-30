// Package agent implements the in-VM agent that executes tools.
//
// The agent binary (fb-agent) runs inside each Firecracker microVM. It listens
// for tool execution requests from the host via vsock, executes them using
// the sandboxed tool implementations, and reports results back.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	atools "github.com/forgebox/forgebox/internal/agent/tools"
)

// Config holds the agent configuration.
type Config struct {
	Tools   *atools.Registry
	Workdir string
}

// Agent executes tools inside a sandboxed VM environment.
type Agent struct {
	tools   *atools.Registry
	workdir string
}

// New creates a new in-VM agent.
func New(cfg Config) *Agent {
	return &Agent{
		tools:   cfg.Tools,
		workdir: cfg.Workdir,
	}
}

// Request is a tool execution request received from the host.
type Request struct {
	ToolName       string          `json:"tool_name"`
	Input          json.RawMessage `json:"input"`
	TimeoutSeconds int             `json:"timeout_seconds,omitempty"`
}

// Response is the result sent back to the host.
type Response struct {
	Output     string `json:"output"`
	IsError    bool   `json:"is_error"`
	DurationMS int64  `json:"duration_ms"`
}

// HandleConnection processes a single connection from the host.
// Each connection carries one request-response pair.
func (a *Agent) HandleConnection(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req Request
	if err := decoder.Decode(&req); err != nil {
		if err != io.EOF {
			slog.Error("failed to decode request", "error", err)
		}
		return
	}

	slog.Info("executing tool", "tool", req.ToolName)
	start := time.Now()

	// Apply timeout.
	timeout := 60 * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp := a.executeTool(execCtx, &req)
	resp.DurationMS = time.Since(start).Milliseconds()

	if err := encoder.Encode(resp); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func (a *Agent) executeTool(ctx context.Context, req *Request) *Response {
	tool, ok := a.tools.Get(req.ToolName)
	if !ok {
		return &Response{
			Output:  fmt.Sprintf("unknown tool: %s", req.ToolName),
			IsError: true,
		}
	}

	result, err := tool.Execute(ctx, req.Input)
	if err != nil {
		return &Response{
			Output:  fmt.Sprintf("tool error: %s", err),
			IsError: true,
		}
	}

	return &Response{
		Output:  result.Content,
		IsError: result.IsError,
	}
}
