package anthropicsubscription

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/forgebox/forgebox/internal/providers/anthropic/base"
)

// gate enforces subscription-mode product limits on an outgoing request:
//   - strips the context-1m beta from any caller-supplied anthropic-beta value
//   - strips cache_control fields from message content blocks
//
// Each strip logs a single WARN explaining the constraint.
func gate(rq *base.Request) {
	stripContext1MBeta(rq)
	stripCacheControl(rq)
}

func stripContext1MBeta(rq *base.Request) {
	if rq.Extras == nil {
		return
	}
	cur, _ := rq.Extras["anthropic-beta"].(string)
	if cur == "" || !strings.Contains(cur, base.BetaContext1M) {
		return
	}
	parts := strings.Split(cur, ",")
	out := parts[:0]
	for _, p := range parts {
		if strings.TrimSpace(p) == base.BetaContext1M {
			continue
		}
		out = append(out, p)
	}
	rq.Extras["anthropic-beta"] = strings.Join(out, ",")
	slog.Warn("subscription auth does not support 1M context; falling back to 200K")
}

func stripCacheControl(rq *base.Request) {
	stripped := false
	for i, msg := range rq.Messages {
		var arr []map[string]any
		if err := json.Unmarshal(msg.Content, &arr); err != nil {
			continue
		}
		modified := false
		for j := range arr {
			if _, ok := arr[j]["cache_control"]; ok {
				delete(arr[j], "cache_control")
				modified = true
			}
		}
		if !modified {
			continue
		}
		newContent, err := json.Marshal(arr)
		if err != nil {
			continue
		}
		rq.Messages[i].Content = newContent
		stripped = true
	}
	if stripped {
		slog.Warn("prompt caching not supported on subscription auth; ignoring cache_control")
	}
}
