package plugins

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider is a minimal sdk.ProviderPlugin implementation for testing.
type mockProvider struct {
	name         string
	shutdownDone bool
}

func (m *mockProvider) Name() string                                   { return m.name }
func (m *mockProvider) Version() string                                { return "1.0.0" }
func (m *mockProvider) Init(_ context.Context, _ map[string]any) error { return nil }
func (m *mockProvider) Shutdown(_ context.Context) error {
	m.shutdownDone = true
	return nil
}
func (m *mockProvider) Models() []sdk.Model { return nil }
func (m *mockProvider) Stream(_ context.Context, _ *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	return nil, nil
}

func (m *mockProvider) Complete(_ context.Context, _ *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	return nil, nil
}

// mockChannel satisfies sdk.ChannelPlugin.
type mockChannel struct{ name string }

func (m *mockChannel) Name() string                                         { return m.name }
func (m *mockChannel) Version() string                                      { return "1.0.0" }
func (m *mockChannel) Init(_ context.Context, _ map[string]any) error       { return nil }
func (m *mockChannel) Shutdown(_ context.Context) error                     { return nil }
func (m *mockChannel) Listen(_ context.Context, _ sdk.MessageHandler) error { return nil }
func (m *mockChannel) Send(_ context.Context, _ *sdk.OutboundMessage) error { return nil }

func TestNewRegistry_IsEmpty(t *testing.T) {
	r := NewRegistry()
	assert.Empty(t, r.ListProviders())
	assert.Empty(t, r.ListTools())
	assert.Empty(t, r.ListChannels())
}

func TestRegisterProvider_And_GetProvider(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "my-provider"}
	r.RegisterProvider(p)

	got, err := r.GetProvider("my-provider")
	require.NoError(t, err)
	assert.Equal(t, "my-provider", got.Name())
}

func TestGetProvider_NotFound_ReturnsError(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetProvider("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestListProviders_ReturnsAllRegistered(t *testing.T) {
	r := NewRegistry()
	r.RegisterProvider(&mockProvider{name: "prov-a"})
	r.RegisterProvider(&mockProvider{name: "prov-b"})

	metas := r.ListProviders()
	assert.Len(t, metas, 2)
}

func TestListProviders_BuiltinsHaveEmptyID(t *testing.T) {
	r := NewRegistry()
	r.RegisterProvider(&mockProvider{name: "builtin"})

	metas := r.ListProviders()
	require.Len(t, metas, 1)
	assert.True(t, metas[0].Builtin)
	assert.Empty(t, metas[0].ID)
}

func TestUnregisterProviderByID_RemovesProvider(t *testing.T) {
	r := NewRegistry()
	// Add a stored (DB-backed) provider entry manually.
	p := &mockProvider{name: "stored"}
	r.mu.Lock()
	r.providers["stored"] = &providerEntry{plugin: p, id: "db-row-1", recordType: "anthropic"}
	r.mu.Unlock()

	removed := r.UnregisterProviderByID(context.Background(), "db-row-1")
	assert.True(t, removed)
	assert.True(t, p.shutdownDone)

	_, err := r.GetProvider("stored")
	require.Error(t, err)
}

func TestUnregisterProviderByID_SkipsBuiltins(t *testing.T) {
	r := NewRegistry()
	// Built-ins have empty id.
	r.RegisterProvider(&mockProvider{name: "builtin"})

	removed := r.UnregisterProviderByID(context.Background(), "")
	assert.False(t, removed, "built-ins must not be unregistered by empty id")
}

func TestUnregisterProviderByID_ReturnsFalseForMissingID(t *testing.T) {
	r := NewRegistry()
	removed := r.UnregisterProviderByID(context.Background(), "unknown-id")
	assert.False(t, removed)
}

func TestRegisterChannel_And_ListChannels(t *testing.T) {
	r := NewRegistry()
	r.RegisterChannel(&mockChannel{name: "slack"})

	channels := r.ListChannels()
	require.Len(t, channels, 1)
	assert.Equal(t, "slack", channels[0].Name)
}

func TestRegisterTool_And_ListTools(t *testing.T) {
	r := NewRegistry()

	// Use a builtinTool since sdk.ToolPlugin requires more methods.
	r.RegisterTool(&builtinTool{name: "bash", desc: "run commands"})
	r.RegisterTool(&builtinTool{name: "glob", desc: "find files"})

	tools := r.ListTools()
	assert.Len(t, tools, 2)
}

func TestNewProvider_KnownTypes(t *testing.T) {
	knownTypes := []string{
		"anthropic",
		"anthropic-api",
		"anthropic-subscription",
		"openai",
		"ollama",
	}
	for _, typ := range knownTypes {
		t.Run(typ, func(t *testing.T) {
			p, err := NewProvider(typ)
			require.NoError(t, err)
			assert.NotNil(t, p)
		})
	}
}

func TestNewProvider_UnknownType_ReturnsError(t *testing.T) {
	_, err := NewProvider("not-a-real-provider")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not-a-real-provider")
}

func TestBuiltinTool_Execute_ReturnsHostDelegationError(t *testing.T) {
	t.Helper()
	bt := &builtinTool{name: "bash", desc: "run commands"}
	res, err := bt.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestBuiltinTool_ReadOnlyTools(t *testing.T) {
	tests := []struct {
		name   string
		wantRO bool
	}{
		{"file_read", true},
		{"glob", true},
		{"grep", true},
		{"bash", false},
		{"file_write", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := &builtinTool{name: tt.name}
			assert.Equal(t, tt.wantRO, bt.IsReadOnly(nil))
		})
	}
}
