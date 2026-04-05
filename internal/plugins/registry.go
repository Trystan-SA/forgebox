// Package plugins manages plugin discovery, loading, and lifecycle.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/internal/providers/anthropic"
	"github.com/forgebox/forgebox/internal/providers/ollama"
	"github.com/forgebox/forgebox/internal/providers/openai"
	"github.com/forgebox/forgebox/pkg/sdk"
)

// Registry holds all loaded plugins indexed by type and name.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]sdk.ProviderPlugin
	channels  map[string]sdk.ChannelPlugin
	tools     map[string]sdk.ToolPlugin
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]sdk.ProviderPlugin),
		channels:  make(map[string]sdk.ChannelPlugin),
		tools:     make(map[string]sdk.ToolPlugin),
	}
}

// LoadBuiltins loads the built-in plugins based on configuration.
func (r *Registry) LoadBuiltins(cfg *config.Config) error {
	// Providers are loaded based on which ones are configured with API keys.
	for name, provCfg := range cfg.Providers {
		slog.Info("loading provider plugin", "name", name)
		p, err := loadBuiltinProvider(name, provCfg)
		if err != nil {
			slog.Warn("skipping provider", "name", name, "error", err)
			continue
		}
		if err := p.Init(context.Background(), provCfg); err != nil {
			return fmt.Errorf("init provider %s: %w", name, err)
		}
		r.RegisterProvider(p)
	}

	// Register built-in tools (these are host-side tool definitions;
	// actual execution happens via fb-agent inside the VM).
	r.registerBuiltinTools()

	return nil
}

// RegisterProvider adds a provider plugin.
func (r *Registry) RegisterProvider(p sdk.ProviderPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// RegisterChannel adds a channel plugin.
func (r *Registry) RegisterChannel(c sdk.ChannelPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[c.Name()] = c
}

// RegisterTool adds a tool plugin.
func (r *Registry) RegisterTool(t sdk.ToolPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// GetProvider returns a provider by name.
func (r *Registry) GetProvider(name string) (sdk.ProviderPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return p, nil
}

// ListProviders returns all registered providers.
func (r *Registry) ListProviders() []sdk.PluginMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]sdk.PluginMeta, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, sdk.PluginMeta{
			Name:    p.Name(),
			Version: p.Version(),
			Type:    sdk.PluginTypeProvider,
			Builtin: true,
		})
	}
	return out
}

// ListTools returns all registered tools.
func (r *Registry) ListTools() []sdk.ToolPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]sdk.ToolPlugin, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// ListChannels returns all registered channels.
func (r *Registry) ListChannels() []sdk.PluginMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]sdk.PluginMeta, 0, len(r.channels))
	for _, c := range r.channels {
		out = append(out, sdk.PluginMeta{
			Name:    c.Name(),
			Version: c.Version(),
			Type:    sdk.PluginTypeChannel,
			Builtin: true,
		})
	}
	return out
}

// Shutdown gracefully stops all plugins.
func (r *Registry) Shutdown(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, p := range r.providers {
		if err := p.Shutdown(ctx); err != nil {
			slog.Error("provider shutdown error", "name", name, "error", err)
		}
	}
	for name, c := range r.channels {
		if err := c.Shutdown(ctx); err != nil {
			slog.Error("channel shutdown error", "name", name, "error", err)
		}
	}
}

// loadBuiltinProvider creates a provider plugin instance by name.
func loadBuiltinProvider(name string, cfg map[string]any) (sdk.ProviderPlugin, error) {
	switch name {
	case "anthropic":
		return anthropic.New(), nil
	case "openai":
		return openai.New(), nil
	case "ollama":
		return ollama.New(), nil
	default:
		return nil, fmt.Errorf("unknown built-in provider %q", name)
	}
}

// registerBuiltinTools registers the host-side tool definitions.
func (r *Registry) registerBuiltinTools() {
	builtins := []sdk.ToolPlugin{
		&builtinTool{name: "bash", desc: "Execute a shell command"},
		&builtinTool{name: "file_read", desc: "Read a file"},
		&builtinTool{name: "file_write", desc: "Write a file"},
		&builtinTool{name: "file_edit", desc: "Edit a file via string replacement"},
		&builtinTool{name: "glob", desc: "Find files matching a pattern"},
		&builtinTool{name: "grep", desc: "Search file contents"},
		&builtinTool{name: "web_fetch", desc: "Fetch a URL"},
	}
	for _, t := range builtins {
		r.RegisterTool(t)
	}
}

// builtinTool is a host-side tool definition. Actual execution is delegated
// to the in-VM agent; this just provides the schema for the LLM.
type builtinTool struct {
	name string
	desc string
}

func (t *builtinTool) Name() string    { return t.name }
func (t *builtinTool) Version() string { return "1.0.0" }

func (t *builtinTool) Init(_ context.Context, _ map[string]any) error { return nil }
func (t *builtinTool) Shutdown(_ context.Context) error               { return nil }

func (t *builtinTool) Schema() sdk.ToolSchema {
	return sdk.ToolSchema{Name: t.name, Description: t.desc}
}

func (t *builtinTool) Execute(_ context.Context, _ json.RawMessage) (*sdk.ToolExecResult, error) {
	// Should never be called on host — execution happens inside the VM.
	return &sdk.ToolExecResult{
		Content: "tool execution must be routed through VM orchestrator",
		IsError: true,
	}, nil
}

func (t *builtinTool) IsReadOnly(_ json.RawMessage) bool    { return t.name == "file_read" || t.name == "glob" || t.name == "grep" }
func (t *builtinTool) IsDestructive(_ json.RawMessage) bool { return false }
