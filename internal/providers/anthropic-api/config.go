package anthropicapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Config is the YAML-decoded configuration for the anthropic provider.
type Config struct {
	APIKey    string `yaml:"api_key"`
	BaseURL   string `yaml:"base_url"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// fromMap decodes a generic config map (the shape passed by the registry).
// API key is resolved via secret-ref before being returned.
func fromMap(ctx context.Context, raw map[string]any) (*Config, error) {
	cfg := &Config{}
	if v, ok := raw["api_key"].(string); ok {
		cfg.APIKey = v
	}
	if v, ok := raw["base_url"].(string); ok {
		cfg.BaseURL = v
	}
	if v, ok := raw["timeout_ms"].(int); ok {
		cfg.TimeoutMS = v
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}
	resolved, err := auth.ResolveSecret(ctx, cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("resolve api_key: %w", err)
	}
	cfg.APIKey = resolved

	if !strings.HasPrefix(cfg.APIKey, "sk-ant-api-") {
		if strings.HasPrefix(cfg.APIKey, "sk-ant-oat") {
			return nil, fmt.Errorf("api_key looks like a subscription token; use the anthropic-subscription provider")
		}
		return nil, fmt.Errorf("api_key must start with sk-ant-api-")
	}
	return cfg, nil
}
