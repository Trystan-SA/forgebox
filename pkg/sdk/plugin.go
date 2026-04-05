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

const (
	PluginTypeProvider PluginType = "provider"
	PluginTypeChannel  PluginType = "channel"
	PluginTypeTool     PluginType = "tool"
	PluginTypeStorage  PluginType = "storage"
)

// PluginMeta contains metadata about a loaded plugin.
type PluginMeta struct {
	Name    string     `json:"name"`
	Version string     `json:"version"`
	Type    PluginType `json:"type"`
	Builtin bool       `json:"builtin"`
}
