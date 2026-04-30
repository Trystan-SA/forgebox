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

type Provider struct {
	*base.Provider
}

func New() *Provider { return &Provider{} }

func (p *Provider) Name() string    { return "anthropic-subscription" }
func (p *Provider) Version() string { return "1.0.0" }

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

func (p *Provider) Shutdown(_ context.Context) error { return nil }

func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
