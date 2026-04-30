// Package engine implements the core LLM tool-calling loop.
//
// The engine orchestrates the conversation between the user, the LLM provider,
// and tools executing inside Firecracker microVMs. It handles context assembly,
// streaming, tool dispatch, permission checking, and cost tracking.
package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/forgebox/forgebox/internal/permissions"
	"github.com/forgebox/forgebox/internal/plugins"
	"github.com/forgebox/forgebox/internal/sessions"
	"github.com/forgebox/forgebox/internal/vm"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Config holds the dependencies for the engine.
type Config struct {
	Registry     *plugins.Registry
	Orchestrator *vm.Orchestrator
	Permissions  *permissions.Checker
	Sessions     *sessions.Manager
}

// Engine executes AI tasks by running the LLM tool-call loop.
type Engine struct {
	registry     *plugins.Registry
	orchestrator *vm.Orchestrator
	permissions  *permissions.Checker
	sessions     *sessions.Manager
}

// New creates a new Engine with the given configuration.
func New(cfg Config) *Engine {
	return &Engine{
		registry:     cfg.Registry,
		orchestrator: cfg.Orchestrator,
		permissions:  cfg.Permissions,
		sessions:     cfg.Sessions,
	}
}

// Task describes a unit of work to execute.
type Task struct {
	ID       string
	Prompt   string
	Provider string
	Model    string
	UserID   string

	// VM configuration overrides.
	MemoryMB      int
	VCPUs         int
	Timeout       time.Duration
	NetworkAccess bool

	// EventSink receives streaming events for real-time output.
	EventSink chan<- Event
}

// Result is the output of a completed task.
type Result struct {
	Output   string
	ToolUses int
	Cost     Cost
	Duration time.Duration
}

// Cost tracks token usage and monetary cost.
type Cost struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

// Event is a streaming event emitted during task execution.
type Event struct {
	Type     string          `json:"type"` // "text", "tool_call", "tool_result", "error", "done"
	Text     string          `json:"text,omitempty"`
	ToolCall *sdk.ToolCall   `json:"tool_call,omitempty"`
	Result   *sdk.ToolResult `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Run executes a task through the full LLM tool-call loop.
func (e *Engine) Run(ctx context.Context, task *Task) (*Result, error) {
	start := time.Now()

	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	provider, err := e.registry.GetProvider(task.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", task.Provider, err)
	}

	tools := e.registry.ListTools()
	toolDefs := make([]sdk.ToolDef, len(tools))
	for i, t := range tools {
		toolDefs[i] = sdk.ToolDef(t.Schema())
	}

	// Boot a VM for this task.
	vmID, err := e.orchestrator.Allocate(ctx, &vm.AllocRequest{
		MemoryMB:      task.MemoryMB,
		VCPUs:         task.VCPUs,
		Timeout:       task.Timeout,
		NetworkAccess: task.NetworkAccess,
	})
	if err != nil {
		return nil, fmt.Errorf("allocate VM: %w", err)
	}
	defer e.orchestrator.Release(ctx, vmID)

	// Build the conversation.
	messages := []sdk.Message{
		{Role: "user", Content: task.Prompt},
	}

	var totalCost Cost
	var toolUseCount int
	const maxIterations = 50

	for i := 0; i < maxIterations; i++ {
		req := &sdk.CompletionRequest{
			Model:       task.Model,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   4096,
			SystemPrompt: buildSystemPrompt(),
		}

		resp, err := provider.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("provider call: %w", err)
		}

		totalCost.InputTokens += resp.Usage.InputTokens
		totalCost.OutputTokens += resp.Usage.OutputTokens

		// Emit text if present.
		if resp.Content != "" {
			e.emit(task, Event{Type: "text", Text: resp.Content})
		}

		// If no tool calls, we're done.
		if len(resp.ToolCalls) == 0 {
			return &Result{
				Output:   resp.Content,
				ToolUses: toolUseCount,
				Cost:     totalCost,
				Duration: time.Since(start),
			}, nil
		}

		// Process tool calls.
		assistantMsg := sdk.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		var toolResults []sdk.ToolResult
		for _, tc := range resp.ToolCalls {
			e.emit(task, Event{Type: "tool_call", ToolCall: &tc})

			// Permission check.
			allowed, reason := e.permissions.Check(task.UserID, tc.Name, tc.Input)
			if !allowed {
				slog.Warn("tool call denied", "tool", tc.Name, "user", task.UserID, "reason", reason)
				result := sdk.ToolResult{
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Permission denied: %s", reason),
					IsError:    true,
				}
				toolResults = append(toolResults, result)
				continue
			}

			// Execute in VM.
			execResult, err := e.orchestrator.Execute(ctx, vmID, tc.Name, tc.Input)
			if err != nil {
				result := sdk.ToolResult{
					ToolCallID: tc.ID,
					Content:    fmt.Sprintf("Execution error: %s", err),
					IsError:    true,
				}
				toolResults = append(toolResults, result)
				continue
			}

			toolUseCount++
			result := sdk.ToolResult{
				ToolCallID: tc.ID,
				Content:    execResult.Content,
				IsError:    execResult.IsError,
			}
			toolResults = append(toolResults, result)
			e.emit(task, Event{Type: "tool_result", Result: &result})
		}

		messages = append(messages, sdk.Message{
			Role:        "user",
			ToolResults: toolResults,
		})
	}

	return nil, errors.New("task exceeded maximum iteration limit")
}

// emit sends an event to the task's event sink if one is configured.
func (e *Engine) emit(task *Task, event Event) {
	if task.EventSink == nil {
		return
	}
	select {
	case task.EventSink <- event:
	default:
		slog.Warn("event sink full, dropping event", "type", event.Type)
	}
}

func buildSystemPrompt() string {
	return `You are ForgeBox, an AI assistant that helps users accomplish tasks by using tools.
You run inside an isolated environment. Use the available tools to complete the user's request.
Be direct and efficient. Execute the task, don't just describe what you would do.`
}
