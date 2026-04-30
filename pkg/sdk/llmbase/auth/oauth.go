package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// OAuthAuth injects an OAuth bearer token via "Authorization: Bearer <token>".
// Optional prefix asserts the token shape (e.g. "sk-ant-oat" for Anthropic
// subscription tokens) and is checked by Validate.
type OAuthAuth struct {
	token  string
	prefix string
}

// NewOAuth constructs an OAuthAuth. An empty prefix disables prefix validation.
func NewOAuth(token, prefix string) *OAuthAuth {
	return &OAuthAuth{token: token, prefix: prefix}
}

// Validate checks the token is non-empty and (if prefix was set) starts with it.
func (a *OAuthAuth) Validate() error {
	if a.token == "" {
		return fmt.Errorf("oauth token is empty")
	}
	if a.prefix != "" && !strings.HasPrefix(a.token, a.prefix) {
		return fmt.Errorf("oauth token must start with %q", a.prefix)
	}
	return nil
}

// Apply implements Auth.
func (a *OAuthAuth) Apply(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.token)
}
