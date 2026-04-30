// Package llmbase provides vendor-agnostic primitives for LLM provider plugins:
// HTTP transport, SSE parsing, streaming pumps, and credential strategies (auth/).
package llmbase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// HTTPOptions configures an HTTPRunner.
type HTTPOptions struct {
	Timeout    time.Duration // per-attempt
	MaxRetries int           // additional attempts after the first; 0 disables retries
	RetryDelay time.Duration // base delay between attempts
}

// HTTPRunner performs JSON POST requests with retry on transient failures.
type HTTPRunner struct {
	client     *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewHTTPRunner builds a runner from options. Defaults: 120s timeout,
// 0 retries, 200ms delay.
func NewHTTPRunner(opts HTTPOptions) *HTTPRunner {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	delay := opts.RetryDelay
	if delay == 0 {
		delay = 200 * time.Millisecond
	}
	return &HTTPRunner{
		client:     &http.Client{Timeout: timeout},
		maxRetries: opts.MaxRetries,
		retryDelay: delay,
	}
}

// Do sends a POST to url with the given headers and body. The body is buffered
// so retries can replay it. Caller must close the returned response body.
//
// Retries on 5xx and network errors up to MaxRetries times. 4xx responses
// are returned to the caller for classification.
func (r *HTTPRunner) Do(ctx context.Context, url string, headers http.Header, body io.Reader) (*http.Response, error) {
	buf, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.retryDelay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		for k, vs := range headers {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}

		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do: %w", err)
			continue
		}
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			drainAndClose(resp)
			lastErr = fmt.Errorf("http %d: %w", resp.StatusCode, sdk.ErrTransient)
			continue
		}
		return resp, nil
	}
	if lastErr == nil {
		lastErr = errors.New("http: exhausted retries")
	}
	return nil, lastErr
}

func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// ClassifyHTTPError maps an HTTP status code and response body to a sdk
// sentinel error wrapped with the body for context. Returns nil if the
// status indicates success.
func ClassifyHTTPError(status int, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	bodyStr := strings.TrimSpace(string(body))
	switch {
	case status == 401 || status == 403:
		return fmt.Errorf("http %d: %s: %w", status, bodyStr, sdk.ErrAuth)
	case status == 429:
		return fmt.Errorf("http %d: %s: %w", status, bodyStr, sdk.ErrRateLimit)
	case status == 400 && strings.Contains(bodyStr, "context_length_exceeded"):
		return fmt.Errorf("http %d: %s: %w", status, bodyStr, sdk.ErrInputTooLarge)
	case status >= 500 && status < 600:
		return fmt.Errorf("http %d: %s: %w", status, bodyStr, sdk.ErrTransient)
	default:
		return fmt.Errorf("http %d: %s", status, bodyStr)
	}
}
