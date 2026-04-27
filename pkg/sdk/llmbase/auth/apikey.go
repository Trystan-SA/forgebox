package auth

import "net/http"

// APIKeyAuth injects a single header carrying an API key.
// Used for vendors with header-based auth (Anthropic x-api-key, OpenAI
// Authorization: Bearer, Google x-goog-api-key, etc.).
type APIKeyAuth struct {
	header string
	value  string
}

// NewAPIKey constructs an APIKeyAuth that sets req.Header[header] = value
// on every Apply call.
func NewAPIKey(header, value string) *APIKeyAuth {
	return &APIKeyAuth{header: header, value: value}
}

// Apply implements Auth.
func (a *APIKeyAuth) Apply(req *http.Request) {
	req.Header.Set(a.header, a.value)
}
