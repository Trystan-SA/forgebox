// Local executor runs tools directly in-process, bypassing Firecracker VMs.
//
// This is used for development and testing when KVM/Firecracker is not available.
// Tools execute in the host process with no isolation — DO NOT use in production.
package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	atools "github.com/forgebox/forgebox/internal/agent/tools"
)

// LocalExecutor runs tools directly without VM isolation.
type LocalExecutor struct {
	tools   *atools.Registry
	workdir string
}

// NewLocalExecutor creates a local executor with all built-in tools.
func NewLocalExecutor(workdir string) *LocalExecutor {
	registry := atools.NewRegistry()
	registry.Register(&atools.BashTool{})
	registry.Register(&atools.FileReadTool{})
	registry.Register(&atools.FileWriteTool{})
	registry.Register(&atools.FileEditTool{})
	registry.Register(&atools.GlobTool{})
	registry.Register(&atools.GrepTool{})
	registry.Register(&atools.WebFetchTool{})

	return &LocalExecutor{
		tools:   registry,
		workdir: workdir,
	}
}

// Execute runs a tool directly in the current process.
func (l *LocalExecutor) Execute(ctx context.Context, toolName string, input json.RawMessage) (*ExecResult, error) {
	tool, ok := l.tools.Get(toolName)
	if !ok {
		return &ExecResult{
			Content: fmt.Sprintf("unknown tool: %s", toolName),
			IsError: true,
		}, nil
	}

	start := time.Now()

	result, err := tool.Execute(ctx, input)
	if err != nil {
		return &ExecResult{
			Content:    fmt.Sprintf("tool error: %s", err),
			IsError:    true,
			DurationMS: time.Since(start).Milliseconds(),
		}, nil
	}

	return &ExecResult{
		Content:    result.Content,
		IsError:    result.IsError,
		DurationMS: time.Since(start).Milliseconds(),
	}, nil
}
