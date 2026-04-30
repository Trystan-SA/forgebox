package anthropicsubscription

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/internal/providers/anthropic/base"
)

func TestGate_StripsContext1MBeta(t *testing.T) {
	rq := &base.Request{
		Extras: map[string]any{
			"anthropic-beta": base.BetaContext1M + "," + base.BetaInterleavedThinking,
		},
	}
	gate(rq)
	v, _ := rq.Extras["anthropic-beta"].(string)
	require.NotContains(t, v, base.BetaContext1M)
}

func TestGate_StripsCacheControl(t *testing.T) {
	contentJSON, _ := json.Marshal([]map[string]any{
		{"type": "text", "text": "hi", "cache_control": map[string]any{"type": "ephemeral"}},
	})
	rq := &base.Request{
		Messages: []base.Message{{Role: "user", Content: contentJSON}},
	}
	gate(rq)
	require.NotContains(t, string(rq.Messages[0].Content), "cache_control")
	require.True(t, strings.Contains(string(rq.Messages[0].Content), `"text":"hi"`))
}
