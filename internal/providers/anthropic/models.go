// Package anthropic exposes the shared Anthropic model catalog used by
// both the anthropic-api and anthropic-subscription providers. The HTTP-level
// shared code lives in the base/ subpackage.
package anthropic

import "github.com/forgebox/forgebox/pkg/sdk"

// Models returns the Anthropic model catalog. The same list is exposed by
// the API-key and subscription providers; the subscription provider may
// filter entries that aren't available on its plan.
func Models() []sdk.Model {
	return []sdk.Model{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
		{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", MaxInputTokens: 200000, MaxOutputTokens: 8192, SupportsTools: true, SupportsVision: true},
		{ID: "claude-opus-4-7", Name: "Claude Opus 4.7", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
	}
}
