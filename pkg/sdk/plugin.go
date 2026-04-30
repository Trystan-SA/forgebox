// Package sdk defines the public plugin interfaces for ForgeBox.
//
// All ForgeBox plugins — providers, channels, tools, and storage backends —
// implement interfaces defined in this package. Plugin authors import this
// package to build extensions.
package sdk

import "context"

// Plugin is the base interface that all ForgeBox plugins must implement.
type Plugin interface {
	// Name returns a unique identifier for this plugin (e.g., "anthropic", "slack").
	Name() string

	// Version returns the semver version of this plugin.
	Version() string

	// Init initializes the plugin with the given configuration.
	// Called once during startup. Return an error to prevent the plugin from loading.
	Init(ctx context.Context, config map[string]any) error

	// Shutdown gracefully stops the plugin and releases resources.
	Shutdown(ctx context.Context) error
}

// PluginType identifies what kind of plugin this is.
type PluginType string

// Plugin type values.
const (
	PluginTypeProvider PluginType = "provider"
	PluginTypeChannel  PluginType = "channel"
	PluginTypeTool     PluginType = "tool"
	PluginTypeStorage  PluginType = "storage"
)

// PluginMeta contains metadata about a loaded plugin.
//
// ID is set for DB-backed entries (so the UI can target them for deletion) and
// empty for built-ins loaded from config. ProviderType identifies the plugin
// implementation (e.g. "anthropic-subscription") for DB-backed providers whose
// Name is a user-supplied display label distinct from the type.
type PluginMeta struct {
	Name         string     `json:"name"`
	Version      string     `json:"version"`
	Type         PluginType `json:"type"`
	Builtin      bool       `json:"builtin"`
	ID           string     `json:"id,omitempty"`
	ProviderType string     `json:"provider_type,omitempty"`
}
