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

// Provider is the anthropic-subscription provider (sdk.ProviderPlugin).
type Provider struct {
	*base.Provider
}

// New returns an unconfigured Provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name implements sdk.Plugin.
func (p *Provider) Name() string { return "anthropic-subscription" }

// Version implements sdk.Plugin.
func (p *Provider) Version() string { return "1.0.0" }

// Init validates and loads configuration.
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
		BaseURL:     cfg.BaseURL,
		Timeout:     timeout,
		GateRequest: gate,
	})
	return nil
}

// Shutdown implements sdk.Plugin.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models returns the catalogue. We currently expose the same list as the
// API provider; if/when 1M-context variants ship, this method filters them.
func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
