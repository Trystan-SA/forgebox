package base

import "strings"

// Anthropic API beta header values.
const (
	BetaFineGrainedToolStreaming = "fine-grained-tool-streaming-2025-05-14"
	BetaInterleavedThinking      = "interleaved-thinking-2025-05-14"
	BetaClaudeCode               = "claude-code-20250219"
	BetaOAuth                    = "oauth-2025-04-20"
	BetaContext1M                = "context-1m-2025-08-07"
)

// APIKeyBetas is the default beta set sent with API-key auth.
var APIKeyBetas = []string{
	BetaFineGrainedToolStreaming,
	BetaInterleavedThinking,
}

// OAuthBetas is the beta set sent with OAuth/subscription auth.
var OAuthBetas = []string{
	BetaClaudeCode,
	BetaOAuth,
	BetaFineGrainedToolStreaming,
	BetaInterleavedThinking,
}

// MergeBetaHeader returns a deduplicated, comma-separated header value built
// from an existing anthropic-beta header and additions. Order is not stable.
func MergeBetaHeader(existing string, add []string) string {
	seen := make(map[string]struct{})
	var out []string
	push := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range strings.Split(existing, ",") {
		push(v)
	}
	for _, v := range add {
		push(v)
	}
	return strings.Join(out, ", ")
}
