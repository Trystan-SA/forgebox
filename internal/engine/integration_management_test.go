//go:build integration

// Integration test for the destructive-action approval gate (spec 5.4.0).
//
// Exercises the real *engine.Engine end-to-end at the engine layer (no
// gateway / WebSocket) with:
//   - a real *plugins.Registry (fake provider + fake destructive tool)
//   - a real *vm.Orchestrator in local mode
//   - a real *engine.Approvals
//   - a real *permissions.Checker
//
// Verifies both branches of the gate:
//   - deny  → tool is NOT executed; LLM gets a synthesized "User declined" result
//   - approve → tool IS dispatched into the orchestrator
//
// Both branches must also emit tool_pending_approval and tool_approval_resolved
// events with matching ApprovalIDs.
package engine_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/internal/engine"
	"github.com/forgebox/forgebox/internal/permissions"
	"github.com/forgebox/forgebox/internal/plugins"
	"github.com/forgebox/forgebox/internal/sessions"
	"github.com/forgebox/forgebox/internal/vm"
	"github.com/forgebox/forgebox/pkg/sdk"
)

// fakeProvider returns a canned tool call on the first Complete and a final
// text response on subsequent Completes. The call counter lets the test assert
// how many times the engine called back into the provider, which is how we
// detect whether the loop continued past the approval gate.
type fakeProvider struct {
	calls    atomic.Int32
	toolCall sdk.ToolCall
}

func (p *fakeProvider) Name() string    { return "fake" }
func (p *fakeProvider) Version() string { return "0.0.0" }
func (p *fakeProvider) Init(_ context.Context, _ map[string]any) error {
	return nil
}
func (p *fakeProvider) Shutdown(_ context.Context) error { return nil }

func (p *fakeProvider) Models() []sdk.Model {
	return []sdk.Model{{ID: "m1", Name: "fake-model", SupportsTools: true}}
}

func (p *fakeProvider) Complete(_ context.Context, _ *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	n := p.calls.Add(1)
	if n == 1 {
		return &sdk.CompletionResponse{
			Content:    "calling tool",
			ToolCalls:  []sdk.ToolCall{p.toolCall},
			StopReason: "tool_use",
			Usage:      sdk.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		}, nil
	}
	return &sdk.CompletionResponse{
		Content:    "done",
		StopReason: "end_turn",
		Usage:      sdk.Usage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	}, nil
}

func (p *fakeProvider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	// Engine.Run uses Complete, not Stream — but the interface requires this.
	ch := make(chan sdk.StreamEvent)
	close(ch)
	return &sdk.StreamResponse{Events: ch}, nil
}

// fakeDestructiveTool implements sdk.ToolPlugin and reports IsDestructive=true
// for all inputs. Execute should never be invoked (the orchestrator dispatches
// to its own in-VM tool registry; Execute on the host plugin is unused).
type fakeDestructiveTool struct{}

func (t *fakeDestructiveTool) Name() string    { return "fake_destructive" }
func (t *fakeDestructiveTool) Version() string { return "0.0.0" }
func (t *fakeDestructiveTool) Init(_ context.Context, _ map[string]any) error {
	return nil
}
func (t *fakeDestructiveTool) Shutdown(_ context.Context) error { return nil }

func (t *fakeDestructiveTool) Schema() sdk.ToolSchema {
	return sdk.ToolSchema{
		Name:        "fake_destructive",
		Description: "destructive test tool",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *fakeDestructiveTool) IsReadOnly(_ json.RawMessage) bool    { return false }
func (t *fakeDestructiveTool) IsDestructive(_ json.RawMessage) bool { return true }
func (t *fakeDestructiveTool) Execute(_ context.Context, _ json.RawMessage) (*sdk.ToolExecResult, error) {
	return &sdk.ToolExecResult{Content: "should not run gateway-side"}, nil
}

// stubAuditStore satisfies sdk.AuditStore for the permission checker — the
// engine's permission checker writes one audit entry per tool call.
type stubAuditStore struct{}

func (stubAuditStore) LogAuditEntry(_ context.Context, _ *sdk.AuditEntry) error { return nil }
func (stubAuditStore) ListAuditEntries(_ context.Context, _ sdk.AuditFilter) ([]*sdk.AuditEntry, error) {
	return nil, nil
}

func TestDestructiveToolApprovalGate(t *testing.T) {
	tests := []struct {
		name     string
		decision bool
	}{
		{name: "deny blocks execution", decision: false},
		{name: "approve allows execution", decision: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			provider := &fakeProvider{toolCall: sdk.ToolCall{
				ID:    "tc-1",
				Name:  "fake_destructive",
				Input: json.RawMessage(`{}`),
			}}

			registry := plugins.NewRegistry()
			registry.RegisterProvider(provider)
			registry.RegisterTool(&fakeDestructiveTool{})

			orch, err := vm.NewOrchestrator(config.VMConfig{
				Mode:            "local",
				DefaultMemoryMB: 128,
				DefaultVCPUs:    1,
				DefaultTimeout:  5 * time.Second,
			})
			require.NoError(t, err)
			defer orch.Shutdown(context.Background())

			approvals := engine.NewApprovals()
			eng := engine.New(engine.Config{
				Registry:     registry,
				Orchestrator: orch,
				Permissions:  permissions.NewChecker(config.AuthConfig{Method: "local"}, stubAuditStore{}),
				Sessions:     sessions.NewManager(nil),
				Approvals:    approvals,
			})

			eventCh := make(chan engine.Event, 64)
			task := &engine.Task{
				ID:        "t-1",
				UserID:    "u-1",
				Provider:  "fake",
				Model:     "m1",
				Prompt:    "do something destructive",
				Timeout:   5 * time.Second,
				EventSink: eventCh,
			}

			type runResult struct {
				result *engine.Result
				err    error
			}
			done := make(chan runResult, 1)
			go func() {
				r, e := eng.Run(ctx, task)
				done <- runResult{result: r, err: e}
			}()

			// Drain events into a slice on a side goroutine so the engine never
			// blocks on a full channel; the test reads from the slice under a
			// mutex. This avoids the easy deadlock where the test is blocked on
			// `<-done` while the engine is blocked on `EventSink <- event`.
			collected := make(chan engine.Event, 64)
			go func() {
				for {
					select {
					case ev, ok := <-eventCh:
						if !ok {
							return
						}
						collected <- ev
					case <-ctx.Done():
						return
					}
				}
			}()

			// Wait for tool_pending_approval to arrive, then capture its id.
			var (
				approvalID string
				seen       []engine.Event
			)
		waitLoop:
			for {
				select {
				case ev := <-collected:
					seen = append(seen, ev)
					if ev.Type == "tool_pending_approval" {
						approvalID = ev.ApprovalID
						break waitLoop
					}
				case <-time.After(3 * time.Second):
					t.Fatalf("did not see tool_pending_approval; saw %d events: %+v", len(seen), seen)
				}
			}
			require.NotEmpty(t, approvalID, "tool_pending_approval must carry an ApprovalID")

			// Deliver the user's decision.
			approvals.Resolve(approvalID, tc.decision)

			// Engine must finish the loop after the decision.
			var rr runResult
			select {
			case rr = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("engine.Run did not return after approval decision")
			}
			assert.NoError(t, rr.err)
			require.NotNil(t, rr.result)
			assert.Equal(t, "done", rr.result.Output, "second Complete should produce the final text")

			// Provider must have been called twice: once for the tool call, once
			// for the follow-up after the tool result was injected. This holds
			// for both branches: on deny the synthesized "User declined" result
			// is fed back into the loop, on approve the orchestrator's result is.
			assert.Equal(t, int32(2), provider.calls.Load(), "provider.Complete should be called exactly twice")

			// Drain remaining events.
			drained := append([]engine.Event{}, seen...)
		drainLoop:
			for {
				select {
				case ev := <-collected:
					drained = append(drained, ev)
				case <-time.After(200 * time.Millisecond):
					break drainLoop
				}
			}

			// Always: a tool_approval_resolved event with our id and a matching
			// Approved field.
			var sawResolution bool
			for _, ev := range drained {
				if ev.Type == "tool_approval_resolved" && ev.ApprovalID == approvalID {
					sawResolution = true
					assert.Equal(t, tc.decision, ev.Approved, "tool_approval_resolved.Approved must match the decision")
				}
			}
			assert.True(t, sawResolution, "expected tool_approval_resolved with id=%s", approvalID)

			// Branch-specific assertions.
			if !tc.decision {
				// Deny: the engine appends a synthesized "User declined"
				// ToolResult to the next user message but does not emit a
				// tool_result event for it — only the resolution event
				// reaches subscribers. The strongest observable signal that
				// Execute was skipped is that ToolUses (counter incremented
				// only on successful orchestrator dispatch) stays zero.
				assert.Equal(t, 0, rr.result.ToolUses, "deny path must not increment ToolUses")
			} else {
				// Approve: the orchestrator dispatches to the local executor,
				// which doesn't know "fake_destructive" and returns a wrapped
				// IsError result without an error. The engine then emits
				// tool_result and increments ToolUses. Verify both.
				var sawToolResult bool
				for _, ev := range drained {
					if ev.Type == "tool_result" && ev.Result != nil && ev.Result.ToolCallID == "tc-1" {
						sawToolResult = true
					}
				}
				assert.True(t, sawToolResult, "approve path must emit a tool_result event")
				assert.Equal(t, 1, rr.result.ToolUses, "approve path must increment ToolUses")
			}
		})
	}
}
