package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	assert.Equal(t, ":8420", cfg.Server.Listen)
	assert.Equal(t, ":8421", cfg.Server.GRPCListen)
	assert.Equal(t, "local", cfg.VM.Mode)
	assert.Equal(t, 512, cfg.VM.DefaultMemoryMB)
	assert.Equal(t, 2, cfg.VM.DefaultVCPUs)
	assert.Equal(t, 5*time.Minute, cfg.VM.DefaultTimeout)
	assert.Equal(t, 5, cfg.VM.PoolSize)
	assert.False(t, cfg.VM.NetworkAccess)
	assert.Equal(t, "local", cfg.Auth.Method)
	assert.True(t, cfg.Telemetry.Metrics)
	assert.True(t, cfg.Telemetry.Traces)
	assert.Equal(t, "0 2 * * *", cfg.Brain.DreamSchedule)
	assert.NotNil(t, cfg.Providers)
	assert.NotNil(t, cfg.Channels)
}

func TestLoad_NonExistentFile_UsesDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/forgebox.yaml")
	require.NoError(t, err)
	assert.Equal(t, ":8420", cfg.Server.Listen)
}

func TestLoad_ValidYAML(t *testing.T) {
	yaml := `
server:
  listen: ":9999"
  grpc_listen: ":9998"
vm:
  mode: firecracker
  pool_size: 10
`
	dir := t.TempDir()
	path := filepath.Join(dir, "forgebox.yaml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, ":9999", cfg.Server.Listen)
	assert.Equal(t, ":9998", cfg.Server.GRPCListen)
	assert.Equal(t, "firecracker", cfg.VM.Mode)
	assert.Equal(t, 10, cfg.VM.PoolSize)
	assert.Equal(t, 512, cfg.VM.DefaultMemoryMB, "unset fields should retain defaults")
}

func TestLoad_MalformedYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forgebox.yaml")
	require.NoError(t, os.WriteFile(path, []byte(":\tinvalid: [[["), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		verify func(t *testing.T, cfg *Config)
	}{
		{
			name: "FORGEBOX_LISTEN",
			env:  map[string]string{"FORGEBOX_LISTEN": ":1234"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, ":1234", cfg.Server.Listen)
			},
		},
		{
			name: "FORGEBOX_GRPC_LISTEN",
			env:  map[string]string{"FORGEBOX_GRPC_LISTEN": ":5678"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, ":5678", cfg.Server.GRPCListen)
			},
		},
		{
			name: "FORGEBOX_VM_MODE",
			env:  map[string]string{"FORGEBOX_VM_MODE": "firecracker"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "firecracker", cfg.VM.Mode)
			},
		},
		{
			name: "DATABASE_URL",
			env:  map[string]string{"DATABASE_URL": "postgres://user:pass@host:5432/db"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "postgres://user:pass@host:5432/db", cfg.Storage.DSN)
			},
		},
		{
			name: "FORGEBOX_DATABASE_URL takes precedence over DATABASE_URL",
			env: map[string]string{
				"FORGEBOX_DATABASE_URL": "postgres://primary",
				"DATABASE_URL":          "postgres://fallback",
			},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "postgres://primary", cfg.Storage.DSN)
			},
		},
		{
			name: "ANTHROPIC_API_KEY sets provider",
			env:  map[string]string{"ANTHROPIC_API_KEY": "sk-ant-test"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Providers["anthropic"])
				assert.Equal(t, "sk-ant-test", cfg.Providers["anthropic"]["api_key"])
			},
		},
		{
			name: "OPENAI_API_KEY sets provider",
			env:  map[string]string{"OPENAI_API_KEY": "sk-openai-test"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Providers["openai"])
				assert.Equal(t, "sk-openai-test", cfg.Providers["openai"]["api_key"])
			},
		},
		{
			name: "GOOGLE_API_KEY sets provider",
			env:  map[string]string{"GOOGLE_API_KEY": "goog-test"},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Providers["google"])
				assert.Equal(t, "goog-test", cfg.Providers["google"]["api_key"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg := Defaults()
			cfg.applyEnvOverrides()
			tt.verify(t, cfg)
		})
	}
}

func TestWriteDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forgebox.yaml")

	require.NoError(t, WriteDefault(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "ForgeBox configuration")
	assert.Contains(t, string(data), "listen")

	// Written file must be loadable and preserve defaults.
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, ":8420", cfg.Server.Listen)
}
