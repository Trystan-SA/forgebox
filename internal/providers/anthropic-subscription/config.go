package anthropicsubscription

import (
	"context"
	"fmt"
	"strings"

	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Config holds the configuration for the anthropic-subscription provider.
type Config struct {
	Token     string `yaml:"token"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// OAuth/subscription auth is bound to api.anthropic.com; base_url is rejected
// to fail loudly rather than silently produce 401s against a custom endpoint.
func fromMap(ctx context.Context, raw map[string]any) (*Config, error) {
	cfg := &Config{}
	if v, ok := raw["token"].(string); ok {
		cfg.Token = v
	}
	if v, ok := raw["base_url"].(string); ok && v != "" {
		return nil, fmt.Errorf("base_url is not supported for anthropic-subscription (OAuth is bound to api.anthropic.com)")
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
			return nil, fmt.Errorf("this looks like an anthropic-api key; use the anthropic-api provider instead")
		}
		return nil, fmt.Errorf("token must start with sk-ant-oat")
	}
	return cfg, nil
}
