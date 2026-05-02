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
	"github.com/forgebox/forgebox/internal/tasktoken"
	"github.com/forgebox/forgebox/internal/vm"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// managementToolNames is the set of in-VM tools that need an authenticated
// callback to the gateway. The engine consults this on every task to decide
// whether to grant ControlPlaneAccess and inject the API token. Keep in sync
// with internal/plugins/management.go and specs/5.0.0-management-tools.md §5.1.0.
var managementToolNames = map[string]bool{
	"list_agents":              true,
	"get_agent":                true,
	"create_agent":             true,
	"update_agent":             true,
	"delete_agent":             true,
	"list_providers":           true,
	"list_models_for_provider": true,
}

// Config holds the dependencies for the engine.
type Config struct {
	Registry     *plugins.Registry
	Orchestrator *vm.Orchestrator
	Permissions  *permissions.Checker
	Sessions     *sessions.Manager
	TaskTokens   *tasktoken.Store // optional; nil disables token issuance for one-shot CLI mode
	APIBaseURL   string           // gateway base URL injected as FORGEBOX_API_URL (e.g. "http://127.0.0.1:8420")
	Approvals    *Approvals       // optional; nil disables the destructive-action gate (one-shot CLI mode)
}

// Engine executes AI tasks by running the LLM tool-call loop.
type Engine struct {
	registry     *plugins.Registry
	orchestrator *vm.Orchestrator
	permissions  *permissions.Checker
	sessions     *sessions.Manager
	taskTokens   *tasktoken.Store
	apiBaseURL   string
	approvals    *Approvals
}

// New creates a new Engine with the given configuration.
func New(cfg Config) *Engine {
	return &Engine{
		registry:     cfg.Registry,
		orchestrator: cfg.Orchestrator,
		permissions:  cfg.Permissions,
		sessions:     cfg.Sessions,
		taskTokens:   cfg.TaskTokens,
		apiBaseURL:   cfg.APIBaseURL,
		approvals:    cfg.Approvals,
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
	Type       string          `json:"type"` // "text", "tool_call", "tool_result", "error", "done", "tool_pending_approval", "tool_approval_resolved"
	Text       string          `json:"text,omitempty"`
	ToolCall   *sdk.ToolCall   `json:"tool_call,omitempty"`
	Result     *sdk.ToolResult `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	ApprovalID string          `json:"approval_id,omitempty"`
	Approved   bool            `json:"approved,omitempty"`
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
	toolByName := make(map[string]sdk.ToolPlugin, len(tools))
	for i, t := range tools {
		toolDefs[i] = sdk.ToolDef(t.Schema())
		toolByName[t.Name()] = t
	}

	// If the task didn't specify a timeout, adopt the orchestrator's default
	// up-front so the VM, the API token, and any downstream lifetime all agree
	// (spec 5.3.0: token lifetime is bounded by the task timeout).
	if task.Timeout <= 0 {
		task.Timeout = e.orchestrator.DefaultTimeout()
	}

	apiToken := ""
	if e.taskTokens != nil {
		apiToken = e.taskTokens.Issue(task.UserID, task.ID, task.Timeout)
		defer e.taskTokens.Revoke(apiToken)
	}

	controlPlane := false
	for _, t := range tools {
		if managementToolNames[t.Schema().Name] {
			controlPlane = true
			break
		}
	}

	// Boot a VM for this task.
	vmID, err := e.orchestrator.Allocate(ctx, &vm.AllocRequest{
		MemoryMB:           task.MemoryMB,
		VCPUs:              task.VCPUs,
		Timeout:            task.Timeout,
		NetworkAccess:      task.NetworkAccess,
		ControlPlaneAccess: controlPlane,
		EnvVars: map[string]string{
			"FORGEBOX_API_URL":   e.apiBaseURL,
			"FORGEBOX_API_TOKEN": apiToken,
		},
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
			Model:        task.Model,
			Messages:     messages,
			Tools:        toolDefs,
			MaxTokens:    4096,
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

			// Destructive-action confirmation gate (spec 5.4.0). If the tool's
			// IsDestructive(input) returns true, pause and wait for the user to
			// approve via the dashboard WebSocket. Timeout / cancel / deny all
			// short-circuit with a synthesized "user declined" tool result.
			if toolPlugin, ok := toolByName[tc.Name]; ok && toolPlugin.IsDestructive(tc.Input) && e.approvals != nil {
				approvalID, ch := e.approvals.Register()
				tcCopy := tc
				e.emit(task, Event{
					Type:       "tool_pending_approval",
					ToolCall:   &tcCopy,
					ApprovalID: approvalID,
				})
				approved := e.approvals.Await(ctx, approvalID, ch, 60*time.Second)
				e.emit(task, Event{
					Type:       "tool_approval_resolved",
					ApprovalID: approvalID,
					Approved:   approved,
				})
				if !approved {
					toolResults = append(toolResults, sdk.ToolResult{
						ToolCallID: tc.ID,
						Content:    "User declined to approve this action.",
						IsError:    true,
					})
					continue
				}
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
