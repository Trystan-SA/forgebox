// Package config handles configuration loading and validation.
//
// ForgeBox uses a layered configuration system:
//   1. Built-in defaults
//   2. /etc/forgebox/forgebox.yaml (system)
//   3. ./forgebox.yaml (project)
//   4. Environment variables (FORGEBOX_*)
//   5. CLI flags
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level ForgeBox configuration.
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	VM        VMConfig                  `yaml:"vm"`
	Providers map[string]map[string]any `yaml:"providers"`
	Channels  map[string]map[string]any `yaml:"channels"`
	Auth      AuthConfig                `yaml:"auth"`
	Storage   StorageConfig             `yaml:"storage"`
	Telemetry TelemetryConfig           `yaml:"telemetry"`
}

// ServerConfig configures the gateway server.
type ServerConfig struct {
	Listen     string `yaml:"listen"`
	GRPCListen string `yaml:"grpc_listen"`
}

// VMConfig configures the Firecracker VM orchestrator.
type VMConfig struct {
	Mode           string        `yaml:"mode"`            // "local" (dev, no VMs) or "firecracker" (production)
	FirecrackerBin string        `yaml:"firecracker_bin"`
	Kernel         string        `yaml:"kernel"`
	Rootfs         string        `yaml:"rootfs"`
	PoolSize       int           `yaml:"pool_size"`
	DefaultMemoryMB int          `yaml:"default_memory_mb"`
	DefaultVCPUs   int           `yaml:"default_vcpus"`
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	NetworkAccess  bool          `yaml:"network_access"`
}

// AuthConfig configures authentication and authorization.
type AuthConfig struct {
	Method string     `yaml:"method"` // "oidc", "apikey", "local"
	OIDC   OIDCConfig `yaml:"oidc,omitempty"`
}

// OIDCConfig configures OpenID Connect authentication.
type OIDCConfig struct {
	Issuer       string `yaml:"issuer"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// StorageConfig configures the storage backend.
type StorageConfig struct {
	Driver   string         `yaml:"driver"` // "sqlite", "postgres"
	SQLite   SQLiteConfig   `yaml:"sqlite,omitempty"`
	Postgres PostgresConfig `yaml:"postgres,omitempty"`
}

// SQLiteConfig configures the SQLite storage backend.
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// PostgresConfig configures the PostgreSQL storage backend.
type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

// TelemetryConfig configures observability.
type TelemetryConfig struct {
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	Metrics      bool   `yaml:"metrics"`
	Traces       bool   `yaml:"traces"`
}

// Defaults returns a Config with sensible default values.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Listen:     ":8420",
			GRPCListen: ":8421",
		},
		VM: VMConfig{
			Mode:           "local",
			FirecrackerBin: "/usr/bin/firecracker",
			Kernel:         "/var/lib/forgebox/vmlinux",
			Rootfs:         "/var/lib/forgebox/rootfs.ext4",
			PoolSize:       5,
			DefaultMemoryMB: 512,
			DefaultVCPUs:   2,
			DefaultTimeout: 5 * time.Minute,
			NetworkAccess:  false,
		},
		Providers: make(map[string]map[string]any),
		Channels:  make(map[string]map[string]any),
		Auth: AuthConfig{
			Method: "local",
		},
		Storage: StorageConfig{
			Driver: "sqlite",
			SQLite: SQLiteConfig{
				Path: "/var/lib/forgebox/forgebox.db",
			},
		},
		Telemetry: TelemetryConfig{
			Metrics: true,
			Traces:  true,
		},
	}
}

// Load reads configuration from file, overlays environment variables,
// and returns the merged config.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file — use defaults with env overrides.
			cfg.applyEnvOverrides()
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Overlay environment variables.
	cfg.applyEnvOverrides()

	return cfg, nil
}

// WriteDefault writes the default configuration to a file.
func WriteDefault(path string) error {
	cfg := Defaults()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	header := []byte("# ForgeBox configuration\n# See https://docs.forgebox.dev/configuration for all options\n\n")
	return os.WriteFile(path, append(header, data...), 0o644)
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("FORGEBOX_LISTEN"); v != "" {
		c.Server.Listen = v
	}
	if v := os.Getenv("FORGEBOX_GRPC_LISTEN"); v != "" {
		c.Server.GRPCListen = v
	}
	if v := os.Getenv("FORGEBOX_STORAGE_PATH"); v != "" {
		c.Storage.SQLite.Path = v
	}
	if v := os.Getenv("FORGEBOX_OTLP_ENDPOINT"); v != "" {
		c.Telemetry.OTLPEndpoint = v
	}
	if v := os.Getenv("FORGEBOX_VM_MODE"); v != "" {
		c.VM.Mode = v
	}

	// Provider API keys from environment.
	envProviders := map[string]string{
		"anthropic":  "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"google":     "GOOGLE_API_KEY",
	}
	for name, envKey := range envProviders {
		if v := os.Getenv(envKey); v != "" {
			if c.Providers[name] == nil {
				c.Providers[name] = make(map[string]any)
			}
			c.Providers[name]["api_key"] = v
		}
	}
}
