// Package anthropicsubscription implements the Claude Max subscription
// provider via OAuth setup tokens (sk-ant-oat01-*).
package anthropicsubscription

import (
	"context"
	"time"

	"github.com/forgebox/forgebox/internal/providers/anthropic"
	"github.com/forgebox/forgebox/internal/providers/anthropic/base"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Provider implements sdk.ProviderPlugin for Claude Max subscription accounts.
type Provider struct {
	*base.Provider
}

// New returns an unconfigured anthropic-subscription provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name returns the provider identifier.
func (p *Provider) Name() string { return "anthropic-subscription" }

// Version returns the provider plugin version.
func (p *Provider) Version() string { return "1.0.0" }

// Init configures the provider from the supplied config map.
func (p *Provider) Init(ctx context.Context, raw map[string]any) error {
	cfg, err := fromMap(ctx, raw)
	if err != nil {
		return err
	}

	a := auth.NewOAuth(cfg.Token, "sk-ant-oat")
	if err := a.Validate(); err != nil {
		return err
	}
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	p.Provider = base.New(base.Options{
		Auth:        a,
		Betas:       base.OAuthBetas,
		Timeout:     timeout,
		GateRequest: gate,
	})
	return nil
}

// Shutdown is a no-op for the anthropic-subscription provider.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models returns the list of supported Anthropic models.
func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
