package anthropicsubscription

import (
	"bytes"
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
	rq.Extras["anthropic-beta"] = base.RemoveBetas(cur, base.BetaContext1M)
	slog.Warn("subscription auth does not support 1M context; falling back to 200K")
}

var cacheControlMarker = []byte("cache_control")

func stripCacheControl(rq *base.Request) {
	stripped := false
	for i, msg := range rq.Messages {
		// Most messages won't carry cache_control; skip the unmarshal+marshal
		// round-trip unless the marker is present in the raw bytes.
		if !bytes.Contains(msg.Content, cacheControlMarker) {
			continue
		}
		var arr []map[string]any
		if err := json.Unmarshal(msg.Content, &arr); err != nil {
			slog.Warn("subscription gate: unparseable message content; leaving as-is", "error", err)
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
			slog.Warn("subscription gate: failed to re-marshal stripped content", "error", err)
			continue
		}
		rq.Messages[i].Content = newContent
		stripped = true
	}
	if stripped {
		slog.Warn("prompt caching not supported on subscription auth; ignoring cache_control")
	}
}
