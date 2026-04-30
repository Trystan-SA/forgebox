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

type Provider struct {
	*base.Provider
}

func New() *Provider { return &Provider{} }

func (p *Provider) Name() string    { return "anthropic-api" }
func (p *Provider) Version() string { return "1.0.0" }

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

func (p *Provider) Shutdown(_ context.Context) error { return nil }

func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
