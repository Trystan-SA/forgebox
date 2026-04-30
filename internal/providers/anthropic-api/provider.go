// Package anthropicapi implements the pay-per-use Anthropic API-key provider.
package anthropicapi

import (
	"context"
	"time"

	"github.com/forgebox/forgebox/internal/providers/anthropic"
	"github.com/forgebox/forgebox/internal/providers/anthropic/base"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Provider is the anthropic provider (sdk.ProviderPlugin).
type Provider struct {
	*base.Provider
}

// New returns an unconfigured Provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name implements sdk.Plugin.
func (p *Provider) Name() string { return "anthropic" }

// Version implements sdk.Plugin.
func (p *Provider) Version() string { return "1.0.0" }

// Init validates and loads configuration.
func (p *Provider) Init(ctx context.Context, raw map[string]any) error {
	cfg, err := fromMap(ctx, raw)
	if err != nil {
		return err
	}
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	p.Provider = base.New(base.Options{
		Auth:    auth.NewAPIKey("x-api-key", cfg.APIKey),
		Betas:   base.APIKeyBetas,
		BaseURL: cfg.BaseURL,
		Timeout: timeout,
	})
	return nil
}

// Shutdown implements sdk.Plugin.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models implements sdk.ProviderPlugin.
func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
