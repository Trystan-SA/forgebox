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

// ListProviders returns all registered providers, including the models each
// provider exposes. The model list is in the provider's preferred display
// order (most powerful first) so dashboards can default to the first entry.
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
			Models:       entry.plugin.Models(),
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

// registerBuiltinTools registers the host-side tool definitions. Each tool
// includes a JSON Schema for its input — Anthropic's /v1/messages rejects
// any tool whose `input_schema` is missing or not a valid object schema, so
// the LLM-facing definitions must always carry one even when execution is
// delegated to the in-VM agent.
func (r *Registry) registerBuiltinTools() {
	stringProp := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	objectSchema := func(props map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}

	builtins := []sdk.ToolPlugin{
		&builtinTool{
			name:   "bash",
			desc:   "Execute a shell command inside the task VM.",
			schema: objectSchema(map[string]any{"command": stringProp("The shell command to run.")}, "command"),
		},
		&builtinTool{
			name:   "file_read",
			desc:   "Read a file from the task VM filesystem.",
			schema: objectSchema(map[string]any{"path": stringProp("Absolute path of the file to read.")}, "path"),
		},
		&builtinTool{
			name: "file_write",
			desc: "Write a file in the task VM filesystem, replacing existing content.",
			schema: objectSchema(map[string]any{
				"path":    stringProp("Absolute path of the file to write."),
				"content": stringProp("Full file contents to write."),
			}, "path", "content"),
		},
		&builtinTool{
			name: "file_edit",
			desc: "Edit a file via exact string replacement.",
			schema: objectSchema(map[string]any{
				"path":       stringProp("Absolute path of the file to edit."),
				"old_string": stringProp("Existing text to replace; must match exactly once."),
				"new_string": stringProp("Replacement text."),
			}, "path", "old_string", "new_string"),
		},
		&builtinTool{
			name:   "glob",
			desc:   "Find files matching a glob pattern.",
			schema: objectSchema(map[string]any{"pattern": stringProp("Glob pattern, e.g. **/*.go.")}, "pattern"),
		},
		&builtinTool{
			name: "grep",
			desc: "Search file contents for a pattern.",
			schema: objectSchema(map[string]any{
				"pattern": stringProp("Regular expression to search for."),
				"path":    stringProp("Optional path or glob to scope the search."),
			}, "pattern"),
		},
		&builtinTool{
			name:   "web_fetch",
			desc:   "Fetch a URL and return its body as text.",
			schema: objectSchema(map[string]any{"url": stringProp("Absolute http(s) URL to fetch.")}, "url"),
		},
		&builtinTool{
			name: "brain",
			desc: "Search, read, and write to your persistent memory. Actions: search (query your memory), read (get a specific file), write (create or update a file), list (list all files), delete (remove a file).",
			schema: objectSchema(map[string]any{
				"action":  map[string]any{"type": "string", "enum": []string{"search", "read", "write", "list", "delete"}, "description": "Memory operation to perform."},
				"query":   stringProp("Search query (action=search)."),
				"path":    stringProp("File path (action=read|write|delete)."),
				"content": stringProp("File contents (action=write)."),
			}, "action"),
		},
		&builtinTool{
			name: "list_agents",
			desc: "List ForgeBox agents visible to the calling user. Optional filter by sharing scope.",
			schema: objectSchema(map[string]any{
				"sharing": map[string]any{"type": "string", "enum": []string{"personal", "team", "org"}, "description": "Optional filter by sharing scope."},
			}),
		},
		&builtinTool{
			name: "get_agent",
			desc: "Return the full record for one ForgeBox agent by id.",
			schema: objectSchema(map[string]any{
				"id": stringProp("Agent id."),
			}, "id"),
		},
		&builtinTool{
			name: "create_agent",
			desc: "Create a new ForgeBox agent. Required: name. Optional: description, system_prompt, provider, model, tools (JSON-encoded array of tool names), sharing (personal|team|org), team_id.",
			schema: objectSchema(map[string]any{
				"name":          stringProp("Display name (non-empty)."),
				"description":   stringProp("Optional free-form description."),
				"system_prompt": stringProp("Optional system prompt prepended to the agent's conversations."),
				"provider":      stringProp("Provider registry name (call list_providers to see options)."),
				"model":         stringProp("Model id from the chosen provider's catalog (call list_models_for_provider to see options)."),
				"tools":         stringProp("JSON-encoded array of allowed tool names, e.g. '[\"bash\",\"file_read\"]'."),
				"sharing":       map[string]any{"type": "string", "enum": []string{"personal", "team", "org"}, "default": "personal", "description": "Visibility scope."},
				"team_id":       stringProp("Required when sharing=team."),
			}, "name"),
		},
		&builtinTool{
			name:        "update_agent",
			desc:        "Patch an existing ForgeBox agent by id. Only fields present in the input are updated. This is a destructive action — the dashboard will ask the user to approve before it runs.",
			destructive: true,
			schema: objectSchema(map[string]any{
				"id":            stringProp("Agent id (required)."),
				"name":          stringProp("New display name."),
				"description":   stringProp("New description."),
				"system_prompt": stringProp("New system prompt."),
				"provider":      stringProp("New provider registry name."),
				"model":         stringProp("New model id."),
				"tools":         stringProp("New JSON-encoded array of allowed tool names."),
				"sharing":       map[string]any{"type": "string", "enum": []string{"personal", "team", "org"}, "description": "New sharing scope."},
				"team_id":       stringProp("New team id; required when sharing=team."),
			}, "id"),
		},
		&builtinTool{
			name:        "delete_agent",
			desc:        "Permanently delete a ForgeBox agent by id. Hard delete; not recoverable. The dashboard will ask the user to approve before it runs.",
			destructive: true,
			schema: objectSchema(map[string]any{
				"id": stringProp("Agent id to delete."),
			}, "id"),
		},
		&builtinTool{
			name:   "list_providers",
			desc:   "List configured LLM providers (name, provider_type, builtin). Use list_models_for_provider to fetch the model catalog for one of them.",
			schema: objectSchema(map[string]any{}),
		},
		&builtinTool{
			name: "list_models_for_provider",
			desc: "List models available on a configured provider, most-powerful-first. Pass the provider's name as returned by list_providers.",
			schema: objectSchema(map[string]any{
				"provider": stringProp("Provider name as returned by list_providers."),
			}, "provider"),
		},
	}
	for _, t := range builtins {
		r.RegisterTool(t)
	}
}

// builtinTool is a host-side tool definition. Actual execution is delegated
// to the in-VM agent; this just provides the schema for the LLM.
type builtinTool struct {
	name        string
	desc        string
	schema      map[string]any
	destructive bool
}

func (t *builtinTool) Name() string    { return t.name }
func (t *builtinTool) Version() string { return "1.0.0" }

func (t *builtinTool) Init(_ context.Context, _ map[string]any) error { return nil }
func (t *builtinTool) Shutdown(_ context.Context) error               { return nil }

func (t *builtinTool) Schema() sdk.ToolSchema {
	return sdk.ToolSchema{Name: t.name, Description: t.desc, InputSchema: t.schema}
}

func (t *builtinTool) Execute(_ context.Context, _ json.RawMessage) (*sdk.ToolExecResult, error) {
	// Should never be called on host — execution happens inside the VM.
	return &sdk.ToolExecResult{
		Content: "tool execution must be routed through VM orchestrator",
		IsError: true,
	}, nil
}

func (t *builtinTool) IsReadOnly(_ json.RawMessage) bool {
	return t.name == "file_read" || t.name == "glob" || t.name == "grep"
}
func (t *builtinTool) IsDestructive(_ json.RawMessage) bool { return t.destructive }
