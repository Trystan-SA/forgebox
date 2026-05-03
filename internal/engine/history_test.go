//go:build integration

// Integration test for Task.History replay (specs/7.0.0).
//
// Verifies that when a Task carries a non-empty History, the engine
// prepends it to the message slice fed to the provider, so multi-turn
// conversations see prior context. Also verifies that Result.NewMessages
// returns only the messages produced during this run (excluding the
// replayed history and the initial user prompt) — see plan Task 5.
package engine_test

import (
	"context"
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

// captureProvider records the messages it receives on each Complete call
// and returns a final no-tool-calls response so the engine exits the loop
// after one iteration. It is independent of the destructive-tool
// fakeProvider used in integration_management_test.go.
type captureProvider struct {
	calls    atomic.Int32
	captured atomic.Pointer[[]sdk.Message]
}

func (p *captureProvider) Name() string    { return "capture" }
func (p *captureProvider) Version() string { return "0.0.0" }
func (p *captureProvider) Init(_ context.Context, _ map[string]any) error {
	return nil
}
func (p *captureProvider) Shutdown(_ context.Context) error { return nil }
func (p *captureProvider) Models() []sdk.Model {
	return []sdk.Model{{ID: "m1", Name: "capture-model", SupportsTools: false}}
}

func (p *captureProvider) Complete(_ context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	p.calls.Add(1)
	msgs := append([]sdk.Message{}, req.Messages...)
	p.captured.Store(&msgs)
	return &sdk.CompletionResponse{
		Content:    "ok",
		StopReason: "end_turn",
	}, nil
}

func (p *captureProvider) Stream(_ context.Context, _ *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	panic("Stream not used in this test")
}

func TestEngine_ReplaysHistoryToProvider(t *testing.T) {
	registry := plugins.NewRegistry()
	provider := &captureProvider{}
	registry.RegisterProvider(provider)

	orch, err := vm.NewOrchestrator(config.VMConfig{
		Mode:            "local",
		DefaultMemoryMB: 128,
		DefaultVCPUs:    1,
		DefaultTimeout:  30 * time.Second,
	})
	require.NoError(t, err)
	defer orch.Shutdown(context.Background())

	eng := engine.New(engine.Config{
		Registry:     registry,
		Orchestrator: orch,
		Permissions:  permissions.NewChecker(config.AuthConfig{Method: "local"}, stubAuditStore{}),
		Sessions:     sessions.NewManager(nil),
	})

	history := []sdk.Message{
		{Role: "user", Content: "earlier"},
		{Role: "assistant", Content: "earlier reply"},
	}
	res, err := eng.Run(context.Background(), &engine.Task{
		Prompt:   "now",
		Provider: "capture",
		Model:    "m1",
		History:  history,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	captured := provider.captured.Load()
	require.NotNil(t, captured)
	require.Len(t, *captured, 3, "provider should see history (2) + new prompt (1)")

	msgs := *captured
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "earlier", msgs[0].Content)
	assert.Equal(t, "assistant", msgs[1].Role)
	assert.Equal(t, "earlier reply", msgs[1].Content)
	assert.Equal(t, "user", msgs[2].Role)
	assert.Equal(t, "now", msgs[2].Content)

	// NewMessages should hold ONLY the new turns produced during this
	// run, not the replayed history. With a no-tool-calls response that
	// is exactly one entry: the final assistant message.
	require.Len(t, res.NewMessages, 1)
	assert.Equal(t, "assistant", res.NewMessages[0].Role)
	assert.Equal(t, "ok", res.NewMessages[0].Content)
}
