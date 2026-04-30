// Package plugins manages plugin discovery, loading, and lifecycle.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/internal/crypto"
	"github.com/forgebox/forgebox/internal/providers/anthropic"
	anthropicapi "github.com/forgebox/forgebox/internal/providers/anthropic-api"
	anthropicsubscription "github.com/forgebox/forgebox/internal/providers/anthropic-subscription"
	"github.com/forgebox/forgebox/internal/providers/ollama"
	"github.com/forgebox/forgebox/internal/providers/openai"
	"github.com/forgebox/forgebox/pkg/sdk"
)

// providerEntry wraps a registered provider with the metadata needed for
// listing and DB-backed deletion. Built-ins have empty ID/RecordType.
type providerEntry struct {
	plugin     sdk.ProviderPlugin
	id         string // DB row id; empty for config-loaded built-ins
	recordType string // factory key; differs from p.Name() for renamed instances
}

// Registry holds all loaded plugins indexed by type and name.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]*providerEntry
	channels  map[string]sdk.ChannelPlugin
	tools     map[string]sdk.ToolPlugin
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]*providerEntry),
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

// LoadFromStore reads DB-backed providers, decrypts each row's config, and
// registers an instance under the user-supplied display name. Rows whose
// config is corrupt or whose type is unknown are skipped with a warn log;
// startup is not aborted.
func (r *Registry) LoadFromStore(ctx context.Context, store sdk.ProviderStore, sb *crypto.SecretBox) error {
	rows, err := store.ListProviders(ctx)
	if err != nil {
		return fmt.Errorf("list providers: %w", err)
	}
	for _, row := range rows {
		if err := r.registerStoredProvider(ctx, row, sb); err != nil {
			slog.Warn("skipping stored provider", "id", row.ID, "name", row.Name, "error", err)
		}
	}
	return nil
}

// AddStoredProvider decrypts, instantiates, and registers a single DB row.
// Used by the gateway after a successful POST so the new provider is live
// without restart.
func (r *Registry) AddStoredProvider(ctx context.Context, row *sdk.ProviderRecord, sb *crypto.SecretBox) error {
	return r.registerStoredProvider(ctx, row, sb)
}

func (r *Registry) registerStoredProvider(ctx context.Context, row *sdk.ProviderRecord, sb *crypto.SecretBox) error {
	r.mu.RLock()
	_, exists := r.providers[row.Name]
	r.mu.RUnlock()
	if exists {
		return fmt.Errorf("name %q already registered", row.Name)
	}

	plain, err := sb.Decrypt(row.ConfigEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt config: %w", err)
	}
	var cfg map[string]any
	if err = json.Unmarshal(plain, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	p, err := loadBuiltinProvider(row.Type, cfg)
	if err != nil {
		return err
	}
	if err := p.Init(ctx, cfg); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[row.Name] = &providerEntry{
		plugin:     p,
		id:         row.ID,
		recordType: row.Type,
	}
	return nil
}

// RegisterProvider adds a built-in provider plugin (key = p.Name()).
func (r *Registry) RegisterProvider(p sdk.ProviderPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = &providerEntry{plugin: p, recordType: p.Name()}
}

// UnregisterProviderByID removes a DB-backed provider matching id and shuts
// it down. Returns false if no such entry exists. Built-ins (with empty id)
// are never matched.
func (r *Registry) UnregisterProviderByID(ctx context.Context, id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, entry := range r.providers {
		if entry.id == "" || entry.id != id {
			continue
		}
		if err := entry.plugin.Shutdown(ctx); err != nil {
			slog.Warn("provider shutdown failed during unregister", "id", id, "error", err)
		}
		delete(r.providers, name)
		return true
	}
	return false
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

// GetProvider returns a provider by registry key (name or display label).
func (r *Registry) GetProvider(name string) (sdk.ProviderPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return entry.plugin, nil
}

// ListProviders returns all registered providers.
func (r *Registry) ListProviders() []sdk.PluginMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]sdk.PluginMeta, 0, len(r.providers))
	for name, entry := range r.providers {
		out = append(out, sdk.PluginMeta{
			Name:         name,
			Version:      entry.plugin.Version(),
			Type:         sdk.PluginTypeProvider,
			Builtin:      entry.id == "",
			ID:           entry.id,
			ProviderType: entry.recordType,
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
	for name, entry := range r.providers {
		if err := entry.plugin.Shutdown(ctx); err != nil {
			slog.Error("provider shutdown error", "name", name, "error", err)
		}
	}
	for name, c := range r.channels {
		if err := c.Shutdown(ctx); err != nil {
			slog.Error("channel shutdown error", "name", name, "error", err)
		}
	}
}

// providerFactories maps a provider type key to a constructor returning an
// unconfigured plugin. Used both for built-in loading and for validating
// user-submitted POST /providers requests.
var providerFactories = map[string]func() sdk.ProviderPlugin{
	"anthropic":              func() sdk.ProviderPlugin { return anthropic.New() },
	"anthropic-api":          func() sdk.ProviderPlugin { return anthropicapi.New() },
	"anthropic-subscription": func() sdk.ProviderPlugin { return anthropicsubscription.New() },
	"openai":                 func() sdk.ProviderPlugin { return openai.New() },
	"ollama":                 func() sdk.ProviderPlugin { return ollama.New() },
}

// NewProvider returns an unconfigured provider instance for the given type
// key. The caller is responsible for Init.
func NewProvider(typ string) (sdk.ProviderPlugin, error) {
	factory, ok := providerFactories[typ]
	if !ok {
		return nil, fmt.Errorf("unknown built-in provider %q", typ)
	}
	return factory(), nil
}

func loadBuiltinProvider(name string, _ map[string]any) (sdk.ProviderPlugin, error) {
	return NewProvider(name)
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
		&builtinTool{
			name: "brain",
			desc: "Search, read, and write to your persistent memory. Actions: search (query your memory), read (get a specific file), write (create or update a file), list (list all files), delete (remove a file).",
		},
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
