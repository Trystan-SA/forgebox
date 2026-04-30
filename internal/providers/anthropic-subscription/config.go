package anthropicsubscription

import (
	"context"
	"fmt"
	"strings"

	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Config is the YAML-decoded configuration for the anthropic-subscription provider.
type Config struct {
	Token     string `yaml:"token"`
	BaseURL   string `yaml:"base_url"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// fromMap decodes a generic config map (the shape passed by the registry).
// Token is resolved via secret-ref before being returned.
func fromMap(ctx context.Context, raw map[string]any) (*Config, error) {
	cfg := &Config{}
	if v, ok := raw["token"].(string); ok {
		cfg.Token = v
	}
	if v, ok := raw["base_url"].(string); ok {
		cfg.BaseURL = v
	}
	if v, ok := raw["timeout_ms"].(int); ok {
		cfg.TimeoutMS = v
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}
	resolved, err := auth.ResolveSecret(ctx, cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("resolve token: %w", err)
	}
	cfg.Token = resolved

	if !strings.HasPrefix(cfg.Token, "sk-ant-oat") {
		if strings.HasPrefix(cfg.Token, "sk-ant-api-") {
			return nil, fmt.Errorf("token looks like an API key; use the anthropic provider")
		}
		return nil, fmt.Errorf("token must start with sk-ant-oat")
	}
	return cfg, nil
}
