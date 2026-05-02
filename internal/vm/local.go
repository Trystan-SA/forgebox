// Local executor runs tools directly in-process, bypassing Firecracker VMs.
//
// This is used for development and testing when KVM/Firecracker is not available.
// Tools execute in the host process with no isolation — DO NOT use in production.
package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	atools "github.com/forgebox/forgebox/internal/agent/tools"
)

// envContextKey is the context key under which per-allocation env vars
// (e.g. FORGEBOX_API_TOKEN) are propagated to in-process tools running in
// local mode. In firecracker mode the same map is communicated to the guest
// via the VM boot config; this is the in-process equivalent.
type envContextKey struct{}

// EnvFromContext returns the per-allocation env map attached by the local
// executor, or nil if none is present.
func EnvFromContext(ctx context.Context) map[string]string {
	v, _ := ctx.Value(envContextKey{}).(map[string]string)
	return v
}

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
//
// The env map is per-allocation guest env (e.g. FORGEBOX_API_TOKEN). Because
// local mode does not spawn a subprocess, env is propagated via context using
// envContextKey; in-process tools that need it can read it via
// EnvFromContext. In firecracker mode the same map is delivered to the guest
// at boot via the VM config (see orchestrator.bootVM).
func (l *LocalExecutor) Execute(ctx context.Context, toolName string, input json.RawMessage, env map[string]string) (*ExecResult, error) {
	tool, ok := l.tools.Get(toolName)
	if !ok {
		return &ExecResult{
			Content: fmt.Sprintf("unknown tool: %s", toolName),
			IsError: true,
		}, nil
	}

	if len(env) > 0 {
		ctx = context.WithValue(ctx, envContextKey{}, env)
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
