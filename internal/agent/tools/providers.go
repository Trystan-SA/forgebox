// Package tools — providers.go provides read-only helpers the LLM uses to
// pick valid provider/model values when creating or updating an agent. See
// specs/5.0.0-management-tools.md §5.1.0.
package tools

import (
	"context"
	"encoding/json"
	"net/http"
)

// ListProvidersTool returns a thin slice of each registered provider —
// name, provider_type, builtin — without the per-provider model catalog.
// Use ListModelsForProviderTool to fetch the model catalog for one provider.
type ListProvidersTool struct{}

// Name returns the tool identifier.
func (t *ListProvidersTool) Name() string { return "list_providers" }

// Execute calls GET /api/v1/providers and returns a slim per-provider summary
// (name, provider_type, builtin) without the model catalog.
func (t *ListProvidersTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodGet, "/api/v1/providers", nil)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	var providers []map[string]any
	if err := json.Unmarshal(body, &providers); err != nil {
		// Don't shadow a successful API call with a parse error — return raw.
		return &Result{Content: string(body)}, nil
	}
	out := make([]map[string]any, 0, len(providers))
	for _, p := range providers {
		out = append(out, map[string]any{
			"name":          p["name"],
			"provider_type": p["provider_type"],
			"builtin":       p["builtin"],
		})
	}
	b, _ := json.Marshal(out)
	return &Result{Content: string(b)}, nil
}

// ListModelsForProviderTool returns the model catalog for one provider, in
// most-powerful-first order (spec 3.3.3). The provider is identified by its
// registry key (the same value `list_providers` returns under "name").
type ListModelsForProviderTool struct{}

// Name returns the tool identifier.
func (t *ListModelsForProviderTool) Name() string { return "list_models_for_provider" }

// Execute returns the model catalog for the named provider, in
// most-powerful-first order.
func (t *ListModelsForProviderTool) Execute(ctx context.Context, input json.RawMessage) (*Result, error) {
	var args struct {
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(input, &args); err != nil || args.Provider == "" {
		return &Result{Content: "missing required field: provider", IsError: true}, nil
	}
	c, err := newAgentAPIClient()
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	body, status, err := c.do(ctx, http.MethodGet, "/api/v1/providers", nil)
	if err != nil {
		return &Result{Content: err.Error(), IsError: true}, nil
	}
	if status >= 400 {
		return &Result{Content: string(body), IsError: true}, nil
	}
	var providers []map[string]any
	if err := json.Unmarshal(body, &providers); err != nil {
		return &Result{Content: "could not parse providers response", IsError: true}, nil
	}
	for _, p := range providers {
		if name, _ := p["name"].(string); name == args.Provider {
			models := p["models"]
			b, _ := json.Marshal(models)
			return &Result{Content: string(b)}, nil
		}
	}
	return &Result{Content: "provider not found: " + args.Provider, IsError: true}, nil
}
