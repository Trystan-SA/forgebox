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

// Provider implements sdk.ProviderPlugin for the pay-per-use Anthropic API.
type Provider struct {
	*base.Provider
}

// New returns an unconfigured anthropic-api provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name returns the provider identifier.
func (p *Provider) Name() string { return "anthropic-api" }

// Version returns the provider plugin version.
func (p *Provider) Version() string { return "1.0.0" }

// Init configures the provider from the supplied config map.
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

// Shutdown is a no-op for the anthropic-api provider.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models returns the list of supported Anthropic models.
func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
