// Package sdk: sentinel errors used by provider plugins so callers can
// branch on failure category without parsing strings. Wrap with fmt.Errorf("...: %w", err).
package sdk

import "errors"

var (
	// ErrAuth indicates an authentication or authorization failure
	// (HTTP 401 or 403). Token invalid, missing scope, expired.
	ErrAuth = errors.New("provider auth failed")

	// ErrRateLimit indicates the provider rejected the call due to rate
	// or quota limits (HTTP 429). Subscription quota exhaustion also maps here.
	ErrRateLimit = errors.New("provider rate limit")

	// ErrInputTooLarge indicates the prompt exceeded the model's context window.
	ErrInputTooLarge = errors.New("provider input too large")

	// ErrTransient indicates a likely-retryable failure: 5xx, network error,
	// timeout, context deadline.
	ErrTransient = errors.New("provider transient error")

	// ErrCLIBackend indicates a failure from a vendor CLI subprocess
	// (non-zero exit, malformed output).
	ErrCLIBackend = errors.New("provider cli backend error")
)
