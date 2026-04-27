# Anthropic Providers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing `internal/providers/anthropic` package with three first-class providers (`anthropic`, `anthropic-subscription`, `claude-cli`) sharing a Go-idiomatic composition base, plus a new `pkg/sdk/llmbase` package of vendor-agnostic primitives (HTTP, SSE, auth, secret-ref).

**Architecture:** `pkg/sdk/llmbase` holds reusable building blocks. `internal/providers/anthropic/base` wraps them with the Anthropic wire protocol. Two sibling provider packages (`anthropic-api`, `anthropic-subscription`) embed that base and supply auth + betas + gating. `claude-cli` is structurally separate — subprocess driver only — but emits the same `sdk.StreamEvent` shape.

**Tech Stack:** Go (gofumpt), `log/slog`, `net/http`, `bufio`, `os/exec`, `testify/require`, table-driven tests, `httptest.Server` for integration.

**Spec reference:** `docs/specs/2026-04-27-anthropic-providers-design.md`

**Conventions (from `CLAUDE.md`):**
- Errors wrap with `fmt.Errorf("context: %w", err)` — never `errors.New` for wrapping.
- All I/O functions take `ctx context.Context` as first parameter.
- `slog` for logging. Pass through struct fields, never globals.
- Named struct initialization. No package-name stutter.
- Tests: table-driven, `testify/require`, `t.Helper()` in helpers, white-box (same package).
- Spec updates ship in the same commit as code changes.

---

## File map

**Created:**
```
pkg/sdk/errors.go
pkg/sdk/llmbase/auth/auth.go
pkg/sdk/llmbase/auth/apikey.go
pkg/sdk/llmbase/auth/oauth.go
pkg/sdk/llmbase/auth/secretref.go
pkg/sdk/llmbase/auth/secretref_test.go
pkg/sdk/llmbase/auth/apikey_test.go
pkg/sdk/llmbase/auth/oauth_test.go
pkg/sdk/llmbase/http.go
pkg/sdk/llmbase/http_test.go
pkg/sdk/llmbase/sse.go
pkg/sdk/llmbase/sse_test.go
pkg/sdk/llmbase/streaming.go

internal/providers/anthropic/base/base.go
internal/providers/anthropic/base/wire.go
internal/providers/anthropic/base/betas.go
internal/providers/anthropic/base/stream.go
internal/providers/anthropic/base/base_test.go
internal/providers/anthropic/base/betas_test.go
internal/providers/anthropic/base/stream_test.go
internal/providers/anthropic/models.go

internal/providers/anthropic-api/provider.go
internal/providers/anthropic-api/config.go
internal/providers/anthropic-api/provider_test.go

internal/providers/anthropic-subscription/provider.go
internal/providers/anthropic-subscription/config.go
internal/providers/anthropic-subscription/gate.go
internal/providers/anthropic-subscription/provider_test.go
internal/providers/anthropic-subscription/gate_test.go

internal/providers/claude-cli/provider.go
internal/providers/claude-cli/config.go
internal/providers/claude-cli/parser.go
internal/providers/claude-cli/models.go
internal/providers/claude-cli/parser_test.go
internal/providers/claude-cli/provider_test.go
internal/providers/claude-cli/testdata/fake-claude.sh

specs/3.0.0-providers.md
```

**Deleted:**
```
internal/providers/anthropic/anthropic.go
```

**Modified:**
```
internal/plugins/registry.go     # replace one mapping with three
```

---

## Task 1: SDK sentinel errors

**Files:**
- Create: `pkg/sdk/errors.go`

- [ ] **Step 1: Create `pkg/sdk/errors.go`**

```go
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
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./pkg/sdk/...`
Expected: clean exit, no output.

- [ ] **Step 3: Commit**

```bash
git add pkg/sdk/errors.go
git commit -m "feat(sdk): add provider sentinel errors"
```

---

## Task 2: llmbase secret-ref resolver

Resolves credential strings: literal, `env://NAME`, `file:///path`, `exec://cmd`.

**Files:**
- Create: `pkg/sdk/llmbase/auth/secretref.go`
- Create: `pkg/sdk/llmbase/auth/secretref_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sdk/llmbase/auth/secretref_test.go`:

```go
package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveSecret(t *testing.T) {
	tmp := t.TempDir()
	tokenFile := filepath.Join(tmp, "tok")
	require.NoError(t, os.WriteFile(tokenFile, []byte("file-secret\n"), 0o600))
	t.Setenv("TEST_SECRET_VAR", "env-secret")

	cases := []struct {
		name    string
		ref     string
		want    string
		wantErr bool
	}{
		{"literal", "sk-ant-api-abc", "sk-ant-api-abc", false},
		{"env", "env://TEST_SECRET_VAR", "env-secret", false},
		{"env missing", "env://NOPE_NOT_SET", "", true},
		{"file trims", "file://" + tokenFile, "file-secret", false},
		{"file missing", "file:///does/not/exist", "", true},
		{"exec", `exec://printf hello`, "hello", false},
		{"exec failure", `exec://false`, "", true},
		{"empty literal", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveSecret(context.Background(), tc.ref)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sdk/llmbase/auth/ -run TestResolveSecret`
Expected: FAIL — `undefined: ResolveSecret`.

- [ ] **Step 3: Implement `ResolveSecret`**

Create `pkg/sdk/llmbase/auth/secretref.go`:

```go
// Package auth holds vendor-agnostic credential strategies and secret-reference
// resolution for LLM provider plugins.
package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ResolveSecret resolves a credential reference to its concrete value.
//
// Accepted forms:
//   - "sk-..." (literal): returned unchanged.
//   - "env://NAME": value of os.Getenv("NAME"). Errors if unset.
//   - "file:///abs/path": contents of the file, whitespace-trimmed.
//   - "exec://command args...": stdout of the command, whitespace-trimmed.
//
// An empty input returns an error.
func ResolveSecret(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty secret reference")
	}
	switch {
	case strings.HasPrefix(ref, "env://"):
		name := strings.TrimPrefix(ref, "env://")
		v := os.Getenv(name)
		if v == "" {
			return "", fmt.Errorf("env var %q is empty or unset", name)
		}
		return v, nil
	case strings.HasPrefix(ref, "file://"):
		path := strings.TrimPrefix(ref, "file://")
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read secret file %q: %w", path, err)
		}
		return strings.TrimSpace(string(b)), nil
	case strings.HasPrefix(ref, "exec://"):
		cmdLine := strings.TrimPrefix(ref, "exec://")
		parts := strings.Fields(cmdLine)
		if len(parts) == 0 {
			return "", fmt.Errorf("empty exec secret command")
		}
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("exec secret %q: %w", cmdLine, err)
		}
		return strings.TrimSpace(string(out)), nil
	default:
		return ref, nil
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sdk/llmbase/auth/ -run TestResolveSecret -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add pkg/sdk/llmbase/auth/secretref.go pkg/sdk/llmbase/auth/secretref_test.go
git commit -m "feat(llmbase): secret-ref resolver (env://, file://, exec://)"
```

---

## Task 3: llmbase auth interface + APIKeyAuth

**Files:**
- Create: `pkg/sdk/llmbase/auth/auth.go`
- Create: `pkg/sdk/llmbase/auth/apikey.go`
- Create: `pkg/sdk/llmbase/auth/apikey_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sdk/llmbase/auth/apikey_test.go`:

```go
package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuth_Apply(t *testing.T) {
	cases := []struct {
		name   string
		header string
		value  string
		want   string
	}{
		{"x-api-key", "x-api-key", "sk-ant-api-abc", "sk-ant-api-abc"},
		{"authorization bearer", "Authorization", "Bearer xyz", "Bearer xyz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAPIKey(tc.header, tc.value)
			req := httptest.NewRequest("POST", "/", nil)
			a.Apply(req)
			require.Equal(t, tc.want, req.Header.Get(tc.header))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sdk/llmbase/auth/ -run TestAPIKeyAuth`
Expected: FAIL — `undefined: NewAPIKey`.

- [ ] **Step 3: Implement interface and APIKeyAuth**

Create `pkg/sdk/llmbase/auth/auth.go`:

```go
package auth

import "net/http"

// Auth is a credential-application strategy. Implementations mutate the
// outgoing request to add whatever headers (or other state) the vendor
// requires.
type Auth interface {
	// Apply adds credentials to the request. Called per request, before send.
	Apply(req *http.Request)
}
```

Create `pkg/sdk/llmbase/auth/apikey.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sdk/llmbase/auth/ -run TestAPIKeyAuth -v`
Expected: PASS for both subtests.

- [ ] **Step 5: Commit**

```bash
git add pkg/sdk/llmbase/auth/auth.go pkg/sdk/llmbase/auth/apikey.go pkg/sdk/llmbase/auth/apikey_test.go
git commit -m "feat(llmbase): Auth interface and APIKeyAuth strategy"
```

---

## Task 4: llmbase OAuthAuth

Bearer-token strategy with prefix validation.

**Files:**
- Create: `pkg/sdk/llmbase/auth/oauth.go`
- Create: `pkg/sdk/llmbase/auth/oauth_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sdk/llmbase/auth/oauth_test.go`:

```go
package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuthAuth_Validate(t *testing.T) {
	cases := []struct {
		name    string
		token   string
		prefix  string
		wantErr bool
	}{
		{"valid prefix", "sk-ant-oat01-abc", "sk-ant-oat", false},
		{"empty prefix accepted", "anything", "", false},
		{"missing prefix", "sk-ant-api-abc", "sk-ant-oat", true},
		{"empty token", "", "sk-ant-oat", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := NewOAuth(tc.token, tc.prefix)
			err := a.Validate()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOAuthAuth_Apply(t *testing.T) {
	a := NewOAuth("sk-ant-oat01-xyz", "sk-ant-oat")
	req := httptest.NewRequest("POST", "/", nil)
	a.Apply(req)
	require.Equal(t, "Bearer sk-ant-oat01-xyz", req.Header.Get("Authorization"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sdk/llmbase/auth/ -run TestOAuth`
Expected: FAIL — `undefined: NewOAuth`.

- [ ] **Step 3: Implement OAuthAuth**

Create `pkg/sdk/llmbase/auth/oauth.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sdk/llmbase/auth/ -v`
Expected: all auth tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/sdk/llmbase/auth/oauth.go pkg/sdk/llmbase/auth/oauth_test.go
git commit -m "feat(llmbase): OAuthAuth strategy with prefix validation"
```

---

## Task 5: llmbase HTTPRunner

Generic POST with retry on 5xx and network errors. Maps responses to `sdk` sentinel errors.

**Files:**
- Create: `pkg/sdk/llmbase/http.go`
- Create: `pkg/sdk/llmbase/http_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sdk/llmbase/http_test.go`:

```go
package llmbase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
)

func TestHTTPRunner_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		require.Equal(t, `{"hello":"world"}`, string(body))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	r := NewHTTPRunner(HTTPOptions{Timeout: 2 * time.Second, MaxRetries: 0})
	resp, err := r.Do(context.Background(), srv.URL, http.Header{}, strings.NewReader(`{"hello":"world"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHTTPRunner_RetryOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := NewHTTPRunner(HTTPOptions{Timeout: 2 * time.Second, MaxRetries: 3, RetryDelay: 1 * time.Millisecond})
	resp, err := r.Do(context.Background(), srv.URL, http.Header{}, strings.NewReader("{}"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.EqualValues(t, 3, calls.Load())
}

func TestClassifyHTTPError(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{401, "unauthorized", sdk.ErrAuth},
		{403, "forbidden", sdk.ErrAuth},
		{429, "rate limited", sdk.ErrRateLimit},
		{400, "context_length_exceeded for model", sdk.ErrInputTooLarge},
		{400, "some other 400", nil}, // not classified, plain error
		{500, "boom", sdk.ErrTransient},
		{502, "bad gateway", sdk.ErrTransient},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			err := ClassifyHTTPError(tc.status, []byte(tc.body))
			if tc.want == nil {
				require.Error(t, err)
				return
			}
			require.True(t, errors.Is(err, tc.want), "got %v, want wrapping %v", err, tc.want)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sdk/llmbase/ -run TestHTTPRunner`
Expected: FAIL — `undefined: NewHTTPRunner`.

- [ ] **Step 3: Implement HTTPRunner**

Create `pkg/sdk/llmbase/http.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/sdk/llmbase/ -v`
Expected: PASS for all subtests.

- [ ] **Step 5: Commit**

```bash
git add pkg/sdk/llmbase/http.go pkg/sdk/llmbase/http_test.go
git commit -m "feat(llmbase): HTTPRunner with retry and error classification"
```

---

## Task 6: llmbase SSE parser

Generic Server-Sent Events line stream parser. Emits raw event-name + data pairs over a channel.

**Files:**
- Create: `pkg/sdk/llmbase/sse.go`
- Create: `pkg/sdk/llmbase/sse_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/sdk/llmbase/sse_test.go`:

```go
package llmbase

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSEParser_Stream(t *testing.T) {
	stream := "event: message_start\n" +
		"data: {\"id\":\"abc\"}\n" +
		"\n" +
		"event: content_block_delta\n" +
		"data: {\"delta\":{\"text\":\"hello\"}}\n" +
		"\n" +
		"event: content_block_delta\n" +
		"data: {\"delta\":{\"text\":\" world\"}}\n" +
		"\n" +
		"event: message_stop\n" +
		"data: {}\n" +
		"\n"

	p := NewSSEParser(strings.NewReader(stream))
	ch := p.Events(context.Background())

	var got []SSEEvent
	for ev := range ch {
		got = append(got, ev)
	}
	require.Len(t, got, 4)
	require.Equal(t, "message_start", got[0].Event)
	require.JSONEq(t, `{"id":"abc"}`, string(got[0].Data))
	require.Equal(t, "content_block_delta", got[1].Event)
	require.Equal(t, "message_stop", got[3].Event)
}

func TestSSEParser_MultilineData(t *testing.T) {
	stream := "event: x\ndata: line1\ndata: line2\n\n"
	p := NewSSEParser(strings.NewReader(stream))
	ch := p.Events(context.Background())
	ev := <-ch
	require.Equal(t, "x", ev.Event)
	require.Equal(t, "line1\nline2", string(ev.Data))
}

func TestSSEParser_IgnoresComments(t *testing.T) {
	stream := ": this is a comment\nevent: ping\ndata: ok\n\n"
	p := NewSSEParser(strings.NewReader(stream))
	ev := <-p.Events(context.Background())
	require.Equal(t, "ping", ev.Event)
	require.Equal(t, "ok", string(ev.Data))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sdk/llmbase/ -run TestSSE`
Expected: FAIL — `undefined: NewSSEParser`.

- [ ] **Step 3: Implement SSE parser**

Create `pkg/sdk/llmbase/sse.go`:

```go
package llmbase

import (
	"bufio"
	"context"
	"io"
	"strings"
)

// SSEEvent is one parsed Server-Sent Event with its name and raw data payload.
// Multi-line data fields are joined with "\n".
type SSEEvent struct {
	Event string
	Data  []byte
}

// SSEParser reads an io.Reader producing SSE-formatted bytes and surfaces
// events on a channel. The reader is consumed in a goroutine; close it from
// outside to stop, or cancel the context passed to Events().
type SSEParser struct {
	r io.Reader
}

// NewSSEParser wraps an io.Reader.
func NewSSEParser(r io.Reader) *SSEParser {
	return &SSEParser{r: r}
}

// Events returns a buffered channel emitting parsed SSE events. The channel
// closes when the underlying reader hits EOF, the context is canceled, or a
// fatal scanner error occurs.
func (p *SSEParser) Events(ctx context.Context) <-chan SSEEvent {
	out := make(chan SSEEvent, 32)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(p.r)
		// Allow large lines: provider tokens can be 1MB+ in a single delta.
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

		var event string
		var dataParts []string

		flush := func() {
			if event == "" && len(dataParts) == 0 {
				return
			}
			ev := SSEEvent{Event: event, Data: []byte(strings.Join(dataParts, "\n"))}
			select {
			case <-ctx.Done():
			case out <- ev:
			}
			event = ""
			dataParts = nil
		}

		for sc.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := sc.Text()
			switch {
			case line == "":
				flush()
			case strings.HasPrefix(line, ":"):
				// comment, ignore
			case strings.HasPrefix(line, "event:"):
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataParts = append(dataParts, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		flush()
	}()
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/sdk/llmbase/ -run TestSSE -v`
Expected: PASS for all three subtests.

- [ ] **Step 5: Commit**

```bash
git add pkg/sdk/llmbase/sse.go pkg/sdk/llmbase/sse_test.go
git commit -m "feat(llmbase): SSE parser with multi-line data and comment handling"
```

---

## Task 7: llmbase StreamPump

Thin glue between an SSE byte stream and a vendor-specific event mapper. Anthropic's base will call this; future providers reuse it.

**Files:**
- Create: `pkg/sdk/llmbase/streaming.go`

- [ ] **Step 1: Implement StreamPump**

Create `pkg/sdk/llmbase/streaming.go`:

```go
package llmbase

import (
	"context"
	"io"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// EventMapper converts a single raw SSE event into zero or more sdk.StreamEvents.
// Returning a nil slice means "skip this SSE event". An error terminates the stream
// (an EventError is emitted by the pump).
type EventMapper func(ev SSEEvent) ([]sdk.StreamEvent, error)

// StreamPump reads SSE events from r, maps each through mapper, and forwards
// results to a buffered sdk.StreamEvent channel. The channel closes when the
// reader ends, ctx is canceled, or mapper returns an error. The pump owns
// closing r via the returned cleanup function the caller passes a function for.
//
// On any error, an EventError event is sent before the channel closes.
func StreamPump(ctx context.Context, r io.ReadCloser, mapper EventMapper) <-chan sdk.StreamEvent {
	out := make(chan sdk.StreamEvent, 64)
	go func() {
		defer close(out)
		defer r.Close()

		parser := NewSSEParser(r)
		for ev := range parser.Events(ctx) {
			events, err := mapper(ev)
			if err != nil {
				select {
				case <-ctx.Done():
				case out <- sdk.StreamEvent{Type: sdk.EventError, Error: err}:
				}
				return
			}
			for _, sev := range events {
				select {
				case <-ctx.Done():
					return
				case out <- sev:
				}
			}
		}
	}()
	return out
}
```

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./pkg/sdk/llmbase/...`
Expected: clean exit.

- [ ] **Step 3: Commit**

```bash
git add pkg/sdk/llmbase/streaming.go
git commit -m "feat(llmbase): StreamPump glues SSE parser to sdk.StreamEvent channel"
```

---

## Task 8: Anthropic wire-protocol types

Request/response structs matching `/v1/messages`.

**Files:**
- Create: `internal/providers/anthropic/base/wire.go`

- [ ] **Step 1: Implement wire types**

Create `internal/providers/anthropic/base/wire.go`:

```go
// Package base holds the shared Anthropic /v1/messages wire protocol used by
// the anthropic-api and anthropic-subscription providers.
package base

import "encoding/json"

// Request is the JSON payload sent to /v1/messages.
type Request struct {
	Model     string         `json:"model"`
	Messages  []Message      `json:"messages"`
	System    string         `json:"system,omitempty"`
	MaxTokens int            `json:"max_tokens"`
	Tools     []Tool         `json:"tools,omitempty"`
	Stream    bool           `json:"stream,omitempty"`
	// Extras allows callers to set ad-hoc fields (e.g. cache_control on a
	// message); marshaled as additional top-level keys via PayloadJSON.
	Extras map[string]any `json:"-"`
}

// Message is one item in the messages array.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Tool is a tool definition.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Response is the non-streaming /v1/messages response.
type Response struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Role    string           `json:"role"`
	Content []ContentBlock   `json:"content"`
	Model   string           `json:"model"`
	StopReason string        `json:"stop_reason"`
	Usage   ResponseUsage    `json:"usage"`
}

// ContentBlock represents one item in the response content array.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ResponseUsage carries token counts.
type ResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
```

- [ ] **Step 2: Build**

Run: `go build ./internal/providers/anthropic/...`
Expected: clean build (the `base` package only).

- [ ] **Step 3: Commit**

```bash
git add internal/providers/anthropic/base/wire.go
git commit -m "feat(anthropic): wire-protocol types for /v1/messages"
```

---

## Task 9: Anthropic base — beta header constants and merge

**Files:**
- Create: `internal/providers/anthropic/base/betas.go`
- Create: `internal/providers/anthropic/base/betas_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/providers/anthropic/base/betas_test.go`:

```go
package base

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeBetaHeader(t *testing.T) {
	cases := []struct {
		name     string
		existing string
		add      []string
		want     []string
	}{
		{"empty existing", "", []string{"a", "b"}, []string{"a", "b"}},
		{"existing only", "x", nil, []string{"x"}},
		{"dedup", "a, b", []string{"b", "c"}, []string{"a", "b", "c"}},
		{"trim spaces", " a , b ", []string{"c"}, []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MergeBetaHeader(tc.existing, tc.add)
			gotParts := strings.Split(got, ",")
			for i := range gotParts {
				gotParts[i] = strings.TrimSpace(gotParts[i])
			}
			sort.Strings(gotParts)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			require.Equal(t, want, gotParts)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/anthropic/base/ -run TestMergeBetaHeader`
Expected: FAIL — `undefined: MergeBetaHeader`.

- [ ] **Step 3: Implement**

Create `internal/providers/anthropic/base/betas.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/providers/anthropic/base/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/providers/anthropic/base/betas.go internal/providers/anthropic/base/betas_test.go
git commit -m "feat(anthropic/base): beta header constants and merge"
```

---

## Task 10: Anthropic base — Provider struct, request build, response parse

The shared HTTP-mode provider that `anthropic-api` and `anthropic-subscription` both wrap. Includes `Init`-style construction, `Complete` (non-stream), and request/response conversion.

**Files:**
- Create: `internal/providers/anthropic/base/base.go`
- Create: `internal/providers/anthropic/base/base_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/providers/anthropic/base/base_test.go`:

```go
package base

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

func TestBuildRequest(t *testing.T) {
	b := New(Options{
		Auth:    auth.NewAPIKey("x-api-key", "k"),
		Betas:   APIKeyBetas,
		BaseURL: "http://example.invalid",
	})

	req := &sdk.CompletionRequest{
		Model:        "claude-sonnet-4-6",
		SystemPrompt: "you are helpful",
		Messages: []sdk.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello"},
			{Role: "system", Content: "ignored"},
		},
		MaxTokens: 1000,
		Tools: []sdk.ToolDef{{
			Name: "echo", Description: "echo", InputSchema: map[string]any{"type": "object"},
		}},
	}
	out := b.BuildRequest(req)
	require.Equal(t, "claude-sonnet-4-6", out.Model)
	require.Equal(t, "you are helpful", out.System)
	require.Equal(t, 1000, out.MaxTokens)
	require.Len(t, out.Messages, 2, "system messages must be excluded")
	require.Equal(t, "user", out.Messages[0].Role)
	require.Len(t, out.Tools, 1)
	require.Equal(t, "echo", out.Tools[0].Name)
}

func TestBuildRequest_Defaults(t *testing.T) {
	b := New(Options{Auth: auth.NewAPIKey("x-api-key", "k"), BaseURL: "http://x"})
	out := b.BuildRequest(&sdk.CompletionRequest{Messages: []sdk.Message{{Role: "user", Content: "hi"}}})
	require.Equal(t, "claude-sonnet-4-6", out.Model, "default model")
	require.Equal(t, 4096, out.MaxTokens, "default max tokens")
}

func TestComplete_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "k", r.Header.Get("x-api-key"))
		require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		require.Contains(t, r.Header.Get("anthropic-beta"), BetaFineGrainedToolStreaming)
		body, _ := io.ReadAll(r.Body)
		var rq Request
		require.NoError(t, json.Unmarshal(body, &rq))
		require.Equal(t, "claude-sonnet-4-6", rq.Model)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"x","content":[{"type":"text","text":"hello"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":2}}`))
	}))
	defer srv.Close()

	b := New(Options{
		Auth:    auth.NewAPIKey("x-api-key", "k"),
		Betas:   APIKeyBetas,
		BaseURL: srv.URL,
		Timeout: 2 * time.Second,
	})

	resp, err := b.Complete(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, "hello", resp.Content)
	require.Equal(t, "end_turn", resp.StopReason)
	require.Equal(t, 1, resp.Usage.InputTokens)
	require.Equal(t, 2, resp.Usage.OutputTokens)
	require.Equal(t, 3, resp.Usage.TotalTokens)
}

func TestComplete_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer srv.Close()

	b := New(Options{Auth: auth.NewAPIKey("x-api-key", "bad"), BaseURL: srv.URL})
	_, err := b.Complete(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, sdk.ErrAuth)
}

func TestComplete_GateRequest_StripsCacheControl(t *testing.T) {
	var seenBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{}}`))
	}))
	defer srv.Close()

	b := New(Options{
		Auth:    auth.NewAPIKey("x-api-key", "k"),
		BaseURL: srv.URL,
		GateRequest: func(rq *Request) {
			rq.System = "" // proves the hook ran
		},
	})
	_, err := b.Complete(context.Background(), &sdk.CompletionRequest{
		Messages:     []sdk.Message{{Role: "user", Content: "hi"}},
		SystemPrompt: "should be stripped by gate",
	})
	require.NoError(t, err)
	require.NotContains(t, string(seenBody), "should be stripped by gate")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/providers/anthropic/base/ -run TestComplete`
Expected: FAIL — `undefined: New`, `undefined: Options`, etc.

- [ ] **Step 3: Implement base.go**

Create `internal/providers/anthropic/base/base.go`:

```go
package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	apiVersion     = "2023-06-01"
	defaultModel   = "claude-sonnet-4-6"
	defaultMaxTok  = 4096
)

// Options configures a Provider.
type Options struct {
	Auth    auth.Auth     // required
	Betas   []string      // default-empty; merged into anthropic-beta header
	BaseURL string        // optional; defaults to https://api.anthropic.com/v1
	Timeout time.Duration // optional; defaults to 120s
	// GateRequest is an optional hook called after BuildRequest. The
	// subscription provider uses it to strip disallowed fields. May be nil.
	GateRequest func(*Request)
}

// Provider implements the Anthropic /v1/messages call shape. Embed it in a
// concrete provider (anthropic-api, anthropic-subscription) which adds Plugin
// metadata and Models().
type Provider struct {
	auth    auth.Auth
	betas   []string
	baseURL string
	gate    func(*Request)
	runner  *llmbase.HTTPRunner
}

// New constructs a base Provider.
func New(opts Options) *Provider {
	url := opts.BaseURL
	if url == "" {
		url = defaultBaseURL
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &Provider{
		auth:    opts.Auth,
		betas:   opts.Betas,
		baseURL: url,
		gate:    opts.GateRequest,
		runner:  llmbase.NewHTTPRunner(llmbase.HTTPOptions{Timeout: timeout}),
	}
}

// BuildRequest converts an sdk.CompletionRequest to the Anthropic wire format.
// System messages in the messages array are dropped; the SystemPrompt field is
// hoisted to the top-level "system" key.
func (p *Provider) BuildRequest(req *sdk.CompletionRequest) *Request {
	model := req.Model
	if model == "" {
		model = defaultModel
	}
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = defaultMaxTok
	}

	msgs := make([]Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		// Anthropic accepts a string OR an array of content blocks. We send
		// strings for plain text; tool results require array form (handled
		// in a future iteration when the orchestrator emits them).
		raw, _ := json.Marshal(m.Content)
		msgs = append(msgs, Message{Role: m.Role, Content: raw})
	}

	tools := make([]Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	return &Request{
		Model:     model,
		Messages:  msgs,
		System:    req.SystemPrompt,
		MaxTokens: maxTok,
		Tools:     tools,
	}
}

// Complete sends a non-streaming /v1/messages call.
func (p *Provider) Complete(ctx context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	apiReq := p.BuildRequest(req)
	if p.gate != nil {
		p.gate(apiReq)
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.runner.Do(ctx, p.baseURL+"/messages", p.headers(""), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic complete: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if cerr := llmbase.ClassifyHTTPError(resp.StatusCode, respBody); cerr != nil {
		return nil, fmt.Errorf("anthropic complete: %w", cerr)
	}

	var ar Response
	if err := json.Unmarshal(respBody, &ar); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return p.convertResponse(&ar), nil
}

func (p *Provider) headers(extraBeta string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("anthropic-version", apiVersion)
	if beta := MergeBetaHeader(extraBeta, p.betas); beta != "" {
		h.Set("anthropic-beta", beta)
	}
	// Wrap http.Request just to invoke Auth; we set on a stub then copy.
	stub := &http.Request{Header: h}
	p.auth.Apply(stub)
	return stub.Header
}

func (p *Provider) convertResponse(ar *Response) *sdk.CompletionResponse {
	var content string
	var toolCalls []sdk.ToolCall
	for _, c := range ar.Content {
		switch c.Type {
		case "text":
			content += c.Text
		case "tool_use":
			toolCalls = append(toolCalls, sdk.ToolCall{ID: c.ID, Name: c.Name, Input: c.Input})
		}
	}
	return &sdk.CompletionResponse{
		Content:    content,
		ToolCalls:  toolCalls,
		StopReason: ar.StopReason,
		Usage: sdk.Usage{
			InputTokens:  ar.Usage.InputTokens,
			OutputTokens: ar.Usage.OutputTokens,
			TotalTokens:  ar.Usage.InputTokens + ar.Usage.OutputTokens,
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/providers/anthropic/base/ -v`
Expected: PASS for all `TestBuildRequest*`, `TestComplete*` cases.

- [ ] **Step 5: Commit**

```bash
git add internal/providers/anthropic/base/base.go internal/providers/anthropic/base/base_test.go
git commit -m "feat(anthropic/base): Provider with BuildRequest, Complete, gate hook"
```

---

## Task 11: Anthropic base — streaming with SSE event mapping

**Files:**
- Create: `internal/providers/anthropic/base/stream.go`
- Create: `internal/providers/anthropic/base/stream_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/providers/anthropic/base/stream_test.go`:

```go
package base

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

const sampleSSE = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"x\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hel\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"lo\"}}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

func TestStream_TextDeltas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleSSE))
	}))
	defer srv.Close()

	b := New(Options{Auth: auth.NewAPIKey("x-api-key", "k"), BaseURL: srv.URL})
	resp, err := b.Stream(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)

	var got []sdk.StreamEvent
	for ev := range resp.Events {
		got = append(got, ev)
	}
	// Expect: 2 text deltas, 1 done.
	var deltas []string
	var sawDone bool
	for _, ev := range got {
		switch ev.Type {
		case sdk.EventTextDelta:
			deltas = append(deltas, ev.Delta)
		case sdk.EventDone:
			sawDone = true
		case sdk.EventError:
			t.Fatalf("unexpected error event: %v", ev.Error)
		}
	}
	require.Equal(t, []string{"hel", "lo"}, deltas)
	require.True(t, sawDone, "must end with EventDone")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/anthropic/base/ -run TestStream`
Expected: FAIL — `undefined: (*Provider).Stream`.

- [ ] **Step 3: Implement Stream**

Create `internal/providers/anthropic/base/stream.go`:

```go
package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase"
)

// streamEnvelope is the common structure of every SSE event from
// /v1/messages with stream=true. Field presence depends on Type.
type streamEnvelope struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Stream sends a streaming /v1/messages call.
func (p *Provider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	apiReq := p.BuildRequest(req)
	apiReq.Stream = true
	if p.gate != nil {
		p.gate(apiReq)
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal stream request: %w", err)
	}

	headers := p.headers("")
	headers.Set("Accept", "text/event-stream")

	resp, err := p.runner.Do(ctx, p.baseURL+"/messages", headers, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("anthropic stream: %w", llmbase.ClassifyHTTPError(resp.StatusCode, respBody))
	}

	mapper := newAnthropicEventMapper()
	events := llmbase.StreamPump(ctx, resp.Body, mapper)
	return &sdk.StreamResponse{Events: events}, nil
}

// newAnthropicEventMapper returns an EventMapper that maps Anthropic SSE
// events to sdk.StreamEvent values. It accumulates tool_use input deltas
// across content_block_delta events and emits a single EventToolCall on
// content_block_stop.
func newAnthropicEventMapper() llmbase.EventMapper {
	type pendingTool struct {
		id, name string
		input    []byte
	}
	tools := map[int]*pendingTool{}

	return func(ev llmbase.SSEEvent) ([]sdk.StreamEvent, error) {
		var env streamEnvelope
		if len(ev.Data) == 0 {
			return nil, nil
		}
		if err := json.Unmarshal(ev.Data, &env); err != nil {
			return nil, fmt.Errorf("unmarshal stream event %q: %w", ev.Event, err)
		}
		switch env.Type {
		case "content_block_start":
			if env.ContentBlock.Type == "tool_use" {
				tools[env.Index] = &pendingTool{id: env.ContentBlock.ID, name: env.ContentBlock.Name}
			}
			return nil, nil
		case "content_block_delta":
			switch env.Delta.Type {
			case "text_delta":
				return []sdk.StreamEvent{{Type: sdk.EventTextDelta, Delta: env.Delta.Text}}, nil
			case "input_json_delta":
				if t, ok := tools[env.Index]; ok {
					t.input = append(t.input, env.Delta.PartialJSON...)
				}
				return nil, nil
			}
			return nil, nil
		case "content_block_stop":
			t, ok := tools[env.Index]
			if !ok {
				return nil, nil
			}
			delete(tools, env.Index)
			input := t.input
			if len(input) == 0 {
				input = []byte("{}")
			}
			return []sdk.StreamEvent{{
				Type:     sdk.EventToolCall,
				ToolCall: &sdk.ToolCall{ID: t.id, Name: t.name, Input: input},
			}}, nil
		case "message_delta":
			if env.Usage.OutputTokens == 0 && env.Usage.InputTokens == 0 {
				return nil, nil
			}
			return []sdk.StreamEvent{{
				Type:  sdk.EventDone,
				Usage: &sdk.Usage{InputTokens: env.Usage.InputTokens, OutputTokens: env.Usage.OutputTokens, TotalTokens: env.Usage.InputTokens + env.Usage.OutputTokens},
			}}, nil
		case "message_stop":
			return []sdk.StreamEvent{{Type: sdk.EventDone}}, nil
		}
		return nil, nil
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/providers/anthropic/base/ -v`
Expected: PASS for all tests including `TestStream_TextDeltas`.

- [ ] **Step 5: Commit**

```bash
git add internal/providers/anthropic/base/stream.go internal/providers/anthropic/base/stream_test.go
git commit -m "feat(anthropic/base): SSE streaming with text and tool_use event mapping"
```

---

## Task 12: Anthropic shared model catalogue

**Files:**
- Create: `internal/providers/anthropic/models.go`

- [ ] **Step 1: Implement models**

Create `internal/providers/anthropic/models.go`:

```go
// Package anthropic re-exports the shared Anthropic model catalogue used by
// both the anthropic-api and anthropic-subscription providers. The HTTP-level
// shared code lives in the base/ subpackage.
package anthropic

import "github.com/forgebox/forgebox/pkg/sdk"

// Models returns the Anthropic model catalogue. The same list is exposed by
// the API-key and subscription providers; the subscription provider may
// filter entries that aren't available on its plan.
func Models() []sdk.Model {
	return []sdk.Model{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
		{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", MaxInputTokens: 200000, MaxOutputTokens: 8192, SupportsTools: true, SupportsVision: true},
		{ID: "claude-opus-4-7", Name: "Claude Opus 4.7", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
	}
}
```

- [ ] **Step 2: Build**

Run: `go build ./internal/providers/anthropic/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/providers/anthropic/models.go
git commit -m "feat(anthropic): shared model catalogue"
```

---

## Task 13: anthropic-api provider

The pay-per-use provider. Wraps `base.Provider` with API-key auth and `sdk.ProviderPlugin` boilerplate.

**Files:**
- Create: `internal/providers/anthropic-api/config.go`
- Create: `internal/providers/anthropic-api/provider.go`
- Create: `internal/providers/anthropic-api/provider_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/providers/anthropic-api/provider_test.go`:

```go
package anthropicapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit_Validates(t *testing.T) {
	cases := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{"missing key", map[string]any{}, "api_key"},
		{"empty key", map[string]any{"api_key": ""}, "api_key"},
		{"oauth token rejected", map[string]any{"api_key": "sk-ant-oat01-abc"}, "subscription"},
		{"valid key", map[string]any{"api_key": "sk-ant-api-abc"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			err := p.Init(context.Background(), tc.cfg)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestNameAndModels(t *testing.T) {
	p := New()
	require.NoError(t, p.Init(context.Background(), map[string]any{"api_key": "sk-ant-api-abc"}))
	require.Equal(t, "anthropic", p.Name())
	require.NotEmpty(t, p.Models())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/anthropic-api/ -run TestInit_Validates`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement config and provider**

Create `internal/providers/anthropic-api/config.go`:

```go
package anthropicapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Config is the YAML-decoded configuration for the anthropic provider.
type Config struct {
	APIKey    string `yaml:"api_key"`
	BaseURL   string `yaml:"base_url"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// fromMap decodes a generic config map (the shape passed by the registry).
// API key is resolved via secret-ref before being returned.
func fromMap(ctx context.Context, raw map[string]any) (*Config, error) {
	cfg := &Config{}
	if v, ok := raw["api_key"].(string); ok {
		cfg.APIKey = v
	}
	if v, ok := raw["base_url"].(string); ok {
		cfg.BaseURL = v
	}
	if v, ok := raw["timeout_ms"].(int); ok {
		cfg.TimeoutMS = v
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}
	resolved, err := auth.ResolveSecret(ctx, cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("resolve api_key: %w", err)
	}
	cfg.APIKey = resolved

	if !strings.HasPrefix(cfg.APIKey, "sk-ant-api-") {
		if strings.HasPrefix(cfg.APIKey, "sk-ant-oat") {
			return nil, fmt.Errorf("api_key looks like a subscription token; use the anthropic-subscription provider")
		}
		return nil, fmt.Errorf("api_key must start with sk-ant-api-")
	}
	return cfg, nil
}
```

Create `internal/providers/anthropic-api/provider.go`:

```go
// Package anthropicapi implements the pay-per-use Anthropic API-key provider.
package anthropicapi

import (
	"context"
	"time"

	"github.com/forgebox/forgebox/internal/providers/anthropic"
	"github.com/forgebox/forgebox/internal/providers/anthropic/base"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Provider is the anthropic provider (sdk.ProviderPlugin).
type Provider struct {
	*base.Provider
}

// New returns an unconfigured Provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name implements sdk.Plugin.
func (p *Provider) Name() string { return "anthropic" }

// Version implements sdk.Plugin.
func (p *Provider) Version() string { return "1.0.0" }

// Init validates and loads configuration.
func (p *Provider) Init(ctx context.Context, raw map[string]any) error {
	cfg, err := fromMap(ctx, raw)
	if err != nil {
		return err
	}
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	p.Provider = base.New(base.Options{
		Auth:    auth.NewAPIKey("x-api-key", cfg.APIKey),
		Betas:   base.APIKeyBetas,
		BaseURL: cfg.BaseURL,
		Timeout: timeout,
	})
	return nil
}

// Shutdown implements sdk.Plugin.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models implements sdk.ProviderPlugin.
func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/providers/anthropic-api/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/providers/anthropic-api/
git commit -m "feat(providers): anthropic API-key provider"
```

---

## Task 14: anthropic-subscription provider with OAuth gating

**Files:**
- Create: `internal/providers/anthropic-subscription/config.go`
- Create: `internal/providers/anthropic-subscription/gate.go`
- Create: `internal/providers/anthropic-subscription/gate_test.go`
- Create: `internal/providers/anthropic-subscription/provider.go`
- Create: `internal/providers/anthropic-subscription/provider_test.go`

- [ ] **Step 1: Write the failing gate test**

Create `internal/providers/anthropic-subscription/gate_test.go`:

```go
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
	// Build a message whose JSON content carries cache_control breakpoints.
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/anthropic-subscription/ -run TestGate`
Expected: FAIL — `undefined: gate`.

- [ ] **Step 3: Implement gate**

Create `internal/providers/anthropic-subscription/gate.go`:

```go
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
		// Content is JSON: either a string or an array of content blocks.
		// Only the array form can carry cache_control, so we attempt to
		// decode as []any and rewrite if successful.
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
```

- [ ] **Step 4: Run gate tests to verify they pass**

Run: `go test ./internal/providers/anthropic-subscription/ -run TestGate -v`
Expected: PASS.

- [ ] **Step 5: Implement config**

Create `internal/providers/anthropic-subscription/config.go`:

```go
package anthropicsubscription

import (
	"context"
	"fmt"
	"strings"

	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Config is the YAML-decoded configuration for the anthropic-subscription provider.
type Config struct {
	Token     string `yaml:"token"`
	BaseURL   string `yaml:"base_url"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

func fromMap(ctx context.Context, raw map[string]any) (*Config, error) {
	cfg := &Config{}
	if v, ok := raw["token"].(string); ok {
		cfg.Token = v
	}
	if v, ok := raw["base_url"].(string); ok {
		cfg.BaseURL = v
	}
	if v, ok := raw["timeout_ms"].(int); ok {
		cfg.TimeoutMS = v
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}
	resolved, err := auth.ResolveSecret(ctx, cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("resolve token: %w", err)
	}
	cfg.Token = resolved

	if !strings.HasPrefix(cfg.Token, "sk-ant-oat") {
		if strings.HasPrefix(cfg.Token, "sk-ant-api-") {
			return nil, fmt.Errorf("token looks like an API key; use the anthropic provider")
		}
		return nil, fmt.Errorf("token must start with sk-ant-oat")
	}
	return cfg, nil
}
```

- [ ] **Step 6: Implement provider**

Create `internal/providers/anthropic-subscription/provider.go`:

```go
// Package anthropicsubscription implements the Claude Max subscription
// provider via OAuth setup tokens (sk-ant-oat01-*).
package anthropicsubscription

import (
	"context"
	"time"

	"github.com/forgebox/forgebox/internal/providers/anthropic"
	"github.com/forgebox/forgebox/internal/providers/anthropic/base"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/forgebox/forgebox/pkg/sdk/llmbase/auth"
)

// Provider is the anthropic-subscription provider (sdk.ProviderPlugin).
type Provider struct {
	*base.Provider
}

// New returns an unconfigured Provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name implements sdk.Plugin.
func (p *Provider) Name() string { return "anthropic-subscription" }

// Version implements sdk.Plugin.
func (p *Provider) Version() string { return "1.0.0" }

// Init validates and loads configuration.
func (p *Provider) Init(ctx context.Context, raw map[string]any) error {
	cfg, err := fromMap(ctx, raw)
	if err != nil {
		return err
	}

	a := auth.NewOAuth(cfg.Token, "sk-ant-oat")
	if err := a.Validate(); err != nil {
		return err
	}
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	p.Provider = base.New(base.Options{
		Auth:        a,
		Betas:       base.OAuthBetas,
		BaseURL:     cfg.BaseURL,
		Timeout:     timeout,
		GateRequest: gate,
	})
	return nil
}

// Shutdown implements sdk.Plugin.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models returns the catalogue. We currently expose the same list as the
// API provider; if/when 1M-context variants ship, this method filters them.
func (p *Provider) Models() []sdk.Model { return anthropic.Models() }
```

- [ ] **Step 7: Write provider Init test**

Create `internal/providers/anthropic-subscription/provider_test.go`:

```go
package anthropicsubscription

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit_Validates(t *testing.T) {
	cases := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{"missing token", map[string]any{}, "token"},
		{"api key rejected", map[string]any{"token": "sk-ant-api-abc"}, "anthropic provider"},
		{"valid token", map[string]any{"token": "sk-ant-oat01-" + repeat('x', 80)}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			err := p.Init(context.Background(), tc.cfg)
			if tc.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, "anthropic-subscription", p.Name())
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func repeat(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}
```

- [ ] **Step 8: Run all subscription tests**

Run: `go test ./internal/providers/anthropic-subscription/ -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/providers/anthropic-subscription/
git commit -m "feat(providers): anthropic-subscription with OAuth gating"
```

---

## Task 15: claude-cli NDJSON parser

`claude -p --output-format stream-json` emits a sequence of newline-delimited JSON envelopes. We map them to `sdk.StreamEvent`.

**Files:**
- Create: `internal/providers/claude-cli/parser.go`
- Create: `internal/providers/claude-cli/parser_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/providers/claude-cli/parser_test.go`:

```go
package claudecli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
)

func TestParseNDJSON(t *testing.T) {
	stream := `{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[{"type":"text","text":"hel"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"lo"}]}}
{"type":"result","subtype":"success","usage":{"input_tokens":3,"output_tokens":2}}
`
	events := ParseNDJSON(strings.NewReader(stream))

	var deltas []string
	var done *sdk.StreamEvent
	for ev := range events {
		switch ev.Type {
		case sdk.EventTextDelta:
			deltas = append(deltas, ev.Delta)
		case sdk.EventDone:
			e := ev
			done = &e
		case sdk.EventError:
			t.Fatalf("unexpected error: %v", ev.Error)
		}
	}
	require.Equal(t, []string{"hel", "lo"}, deltas)
	require.NotNil(t, done)
	require.NotNil(t, done.Usage)
	require.Equal(t, 3, done.Usage.InputTokens)
	require.Equal(t, 2, done.Usage.OutputTokens)
}

func TestParseNDJSON_MalformedLineEmitsError(t *testing.T) {
	stream := `{"type":"assistant","message":{"content":[{"type":"text","text":"ok"}]}}
not-json
`
	var sawError, sawDelta bool
	for ev := range ParseNDJSON(strings.NewReader(stream)) {
		switch ev.Type {
		case sdk.EventTextDelta:
			sawDelta = true
		case sdk.EventError:
			sawError = true
		}
	}
	require.True(t, sawDelta)
	require.True(t, sawError)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/claude-cli/ -run TestParseNDJSON`
Expected: FAIL — `undefined: ParseNDJSON`.

- [ ] **Step 3: Implement parser**

Create `internal/providers/claude-cli/parser.go`:

```go
package claudecli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// cliEnvelope is a permissive view over claude --output-format stream-json
// envelopes. Field presence depends on Type/Subtype.
type cliEnvelope struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	} `json:"message"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ParseNDJSON consumes claude-cli's stream-json output line-by-line and emits
// sdk.StreamEvents. The channel closes when r reaches EOF; malformed lines
// emit an EventError but parsing continues with the next line.
func ParseNDJSON(r io.Reader) <-chan sdk.StreamEvent {
	out := make(chan sdk.StreamEvent, 32)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		var sawDone bool
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			var env cliEnvelope
			if err := json.Unmarshal(line, &env); err != nil {
				out <- sdk.StreamEvent{Type: sdk.EventError, Error: fmt.Errorf("claude-cli: malformed line: %w", err)}
				continue
			}
			switch env.Type {
			case "assistant":
				for _, c := range env.Message.Content {
					switch c.Type {
					case "text":
						if c.Text != "" {
							out <- sdk.StreamEvent{Type: sdk.EventTextDelta, Delta: c.Text}
						}
					case "tool_use":
						input := []byte(c.Input)
						if len(input) == 0 {
							input = []byte("{}")
						}
						out <- sdk.StreamEvent{
							Type:     sdk.EventToolCall,
							ToolCall: &sdk.ToolCall{ID: c.ID, Name: c.Name, Input: input},
						}
					}
				}
			case "result":
				usage := &sdk.Usage{
					InputTokens:  env.Usage.InputTokens,
					OutputTokens: env.Usage.OutputTokens,
					TotalTokens:  env.Usage.InputTokens + env.Usage.OutputTokens,
				}
				out <- sdk.StreamEvent{Type: sdk.EventDone, Usage: usage}
				sawDone = true
			}
		}
		if err := sc.Err(); err != nil {
			out <- sdk.StreamEvent{Type: sdk.EventError, Error: fmt.Errorf("claude-cli: scan: %w", err)}
		}
		if !sawDone {
			out <- sdk.StreamEvent{Type: sdk.EventDone}
		}
	}()
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/providers/claude-cli/ -run TestParseNDJSON -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/providers/claude-cli/parser.go internal/providers/claude-cli/parser_test.go
git commit -m "feat(claude-cli): NDJSON envelope parser"
```

---

## Task 16: claude-cli provider, models, fake binary

**Files:**
- Create: `internal/providers/claude-cli/config.go`
- Create: `internal/providers/claude-cli/models.go`
- Create: `internal/providers/claude-cli/provider.go`
- Create: `internal/providers/claude-cli/provider_test.go`
- Create: `internal/providers/claude-cli/testdata/fake-claude.sh`

- [ ] **Step 1: Create the fake binary**

Create `internal/providers/claude-cli/testdata/fake-claude.sh`:

```bash
#!/usr/bin/env bash
# Fake `claude` binary for tests. Recognizes --version; for inference invocations
# emits a canned NDJSON stream regardless of input.
set -e

if [[ "$1" == "--version" ]]; then
  echo "claude 1.0.0-fake"
  exit 0
fi

# Drain stdin so the parent's pipe write doesn't block.
cat > /dev/null

cat <<'EOF'
{"type":"system","subtype":"init"}
{"type":"assistant","message":{"content":[{"type":"text","text":"fake-output"}]}}
{"type":"result","subtype":"success","usage":{"input_tokens":1,"output_tokens":2}}
EOF
```

Then make it executable:

```bash
chmod +x internal/providers/claude-cli/testdata/fake-claude.sh
```

- [ ] **Step 2: Implement models and config**

Create `internal/providers/claude-cli/models.go`:

```go
package claudecli

import "github.com/forgebox/forgebox/pkg/sdk"

// Models returns the claude-cli/* model namespace. The CLI accepts bare model
// IDs; mapModelID strips the prefix before invoking it.
func Models() []sdk.Model {
	return []sdk.Model{
		{ID: "claude-cli/claude-sonnet-4-6", Name: "Claude Sonnet 4.6 (CLI)", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
		{ID: "claude-cli/claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5 (CLI)", MaxInputTokens: 200000, MaxOutputTokens: 8192, SupportsTools: true, SupportsVision: true},
		{ID: "claude-cli/claude-opus-4-7", Name: "Claude Opus 4.7 (CLI)", MaxInputTokens: 200000, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true},
	}
}

// mapModelID strips the "claude-cli/" prefix from a model ID. If the prefix
// is absent, the input is returned unchanged.
func mapModelID(id string) string {
	const prefix = "claude-cli/"
	if len(id) > len(prefix) && id[:len(prefix)] == prefix {
		return id[len(prefix):]
	}
	return id
}
```

Create `internal/providers/claude-cli/config.go`:

```go
package claudecli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Config is the YAML-decoded configuration for the claude-cli provider.
type Config struct {
	Bin       string `yaml:"bin"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

func fromMap(_ context.Context, raw map[string]any) (*Config, error) {
	cfg := &Config{TimeoutMS: 600000}
	if v, ok := raw["bin"].(string); ok {
		cfg.Bin = v
	}
	if v, ok := raw["timeout_ms"].(int); ok {
		cfg.TimeoutMS = v
	}

	if cfg.Bin == "" {
		path, err := exec.LookPath("claude")
		if err != nil {
			return nil, fmt.Errorf("claude binary not found in PATH; set bin: in config")
		}
		cfg.Bin = path
	}
	st, err := os.Stat(cfg.Bin)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", cfg.Bin, err)
	}
	if st.IsDir() || st.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("%q is not an executable file", cfg.Bin)
	}
	return cfg, nil
}
```

- [ ] **Step 3: Implement the provider**

Create `internal/providers/claude-cli/provider.go`:

```go
// Package claudecli implements an LLM provider that delegates inference to a
// local `claude` CLI binary. Authentication and the agentic loop are owned
// by the binary; ForgeBox feeds it a prompt and parses its NDJSON output.
package claudecli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// Provider is the claude-cli provider (sdk.ProviderPlugin).
type Provider struct {
	bin     string
	timeout time.Duration
}

// New returns an unconfigured Provider; call Init before use.
func New() *Provider { return &Provider{} }

// Name implements sdk.Plugin.
func (p *Provider) Name() string { return "claude-cli" }

// Version implements sdk.Plugin.
func (p *Provider) Version() string { return "1.0.0" }

// Init validates configuration and probes the binary.
func (p *Provider) Init(ctx context.Context, raw map[string]any) error {
	cfg, err := fromMap(ctx, raw)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, cfg.Bin, "--version")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("probe %q --version: %w (%s)", cfg.Bin, err, strings.TrimSpace(string(out)))
	}
	p.bin = cfg.Bin
	p.timeout = time.Duration(cfg.TimeoutMS) * time.Millisecond
	return nil
}

// Shutdown implements sdk.Plugin.
func (p *Provider) Shutdown(_ context.Context) error { return nil }

// Models implements sdk.ProviderPlugin.
func (p *Provider) Models() []sdk.Model { return Models() }

// Stream invokes the CLI binary and streams its NDJSON output as sdk.StreamEvents.
func (p *Provider) Stream(ctx context.Context, req *sdk.CompletionRequest) (*sdk.StreamResponse, error) {
	cctx, cancel := context.WithTimeout(ctx, p.timeout)

	args := []string{"-p", "--output-format", "stream-json", "--verbose", "--permission-mode", "bypassPermissions"}
	if model := mapModelID(req.Model); model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(cctx, p.bin, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start claude: %w", err)
	}

	go func() {
		defer stdin.Close()
		_, _ = io.WriteString(stdin, buildPrompt(req))
	}()

	parsed := ParseNDJSON(stdout)

	out := make(chan sdk.StreamEvent, 64)
	go func() {
		defer close(out)
		defer cancel()
		for ev := range parsed {
			select {
			case <-ctx.Done():
				return
			case out <- ev:
			}
		}
		if err := cmd.Wait(); err != nil {
			out <- sdk.StreamEvent{
				Type:  sdk.EventError,
				Error: fmt.Errorf("%w: %s: %s", sdk.ErrCLIBackend, err, strings.TrimSpace(stderr.String())),
			}
		}
	}()

	return &sdk.StreamResponse{Events: out}, nil
}

// Complete invokes Stream and accumulates events into a single response.
func (p *Provider) Complete(ctx context.Context, req *sdk.CompletionRequest) (*sdk.CompletionResponse, error) {
	sr, err := p.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	var content strings.Builder
	var toolCalls []sdk.ToolCall
	var usage sdk.Usage
	var stopReason string
	for ev := range sr.Events {
		switch ev.Type {
		case sdk.EventTextDelta:
			content.WriteString(ev.Delta)
		case sdk.EventToolCall:
			if ev.ToolCall != nil {
				toolCalls = append(toolCalls, *ev.ToolCall)
			}
		case sdk.EventDone:
			if ev.Usage != nil {
				usage = *ev.Usage
			}
			stopReason = "end_turn"
		case sdk.EventError:
			return nil, ev.Error
		}
	}
	return &sdk.CompletionResponse{
		Content:    content.String(),
		ToolCalls:  toolCalls,
		StopReason: stopReason,
		Usage:      usage,
	}, nil
}

// buildPrompt assembles a single prompt string from a CompletionRequest. The
// CLI takes the prompt on stdin in -p mode; system prompt and conversation
// history are interleaved as plain text.
func buildPrompt(req *sdk.CompletionRequest) string {
	var b strings.Builder
	if req.SystemPrompt != "" {
		b.WriteString("System: ")
		b.WriteString(req.SystemPrompt)
		b.WriteString("\n\n")
	}
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			b.WriteString("User: ")
		case "assistant":
			b.WriteString("Assistant: ")
		case "system":
			b.WriteString("System: ")
		default:
			b.WriteString(m.Role + ": ")
		}
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}
```

- [ ] **Step 4: Write the provider integration test**

Create `internal/providers/claude-cli/provider_test.go`:

```go
package claudecli

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/forgebox/forgebox/pkg/sdk"
)

func fakeBinaryPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-claude.sh requires bash; skipping on windows")
	}
	abs, err := filepath.Abs("testdata/fake-claude.sh")
	require.NoError(t, err)
	return abs
}

func TestInit_LooksUpBinary(t *testing.T) {
	bin := fakeBinaryPath(t)
	p := New()
	err := p.Init(context.Background(), map[string]any{"bin": bin})
	require.NoError(t, err)
	require.Equal(t, "claude-cli", p.Name())
}

func TestInit_RejectsMissingBinary(t *testing.T) {
	p := New()
	err := p.Init(context.Background(), map[string]any{"bin": "/no/such/binary"})
	require.Error(t, err)
}

func TestComplete_StreamsFakeOutput(t *testing.T) {
	bin := fakeBinaryPath(t)
	p := New()
	require.NoError(t, p.Init(context.Background(), map[string]any{"bin": bin}))

	resp, err := p.Complete(context.Background(), &sdk.CompletionRequest{
		Messages: []sdk.Message{{Role: "user", Content: "hi"}},
	})
	require.NoError(t, err)
	require.True(t, strings.Contains(resp.Content, "fake-output"))
	require.Equal(t, 1, resp.Usage.InputTokens)
	require.Equal(t, 2, resp.Usage.OutputTokens)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/providers/claude-cli/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/providers/claude-cli/
git commit -m "feat(providers): claude-cli subprocess provider"
```

---

## Task 17: Delete old provider, update registry

Switch the registry over to the new package layout, delete the broken file, update imports.

**Files:**
- Delete: `internal/providers/anthropic/anthropic.go`
- Modify: `internal/plugins/registry.go`

- [ ] **Step 1: Delete the old file**

```bash
git rm internal/providers/anthropic/anthropic.go
```

- [ ] **Step 2: Update the registry**

Edit `internal/plugins/registry.go`. Replace the imports and `loadBuiltinProvider` function to register the three new providers.

Change the imports block from:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/internal/providers/anthropic"
	"github.com/forgebox/forgebox/internal/providers/ollama"
	"github.com/forgebox/forgebox/internal/providers/openai"
	"github.com/forgebox/forgebox/pkg/sdk"
)
```

To:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/forgebox/forgebox/internal/config"
	anthropicapi "github.com/forgebox/forgebox/internal/providers/anthropic-api"
	anthropicsubscription "github.com/forgebox/forgebox/internal/providers/anthropic-subscription"
	claudecli "github.com/forgebox/forgebox/internal/providers/claude-cli"
	"github.com/forgebox/forgebox/internal/providers/ollama"
	"github.com/forgebox/forgebox/internal/providers/openai"
	"github.com/forgebox/forgebox/pkg/sdk"
)
```

Change `loadBuiltinProvider` from:

```go
func loadBuiltinProvider(name string, cfg map[string]any) (sdk.ProviderPlugin, error) {
	switch name {
	case "anthropic":
		return anthropic.New(), nil
	case "openai":
		return openai.New(), nil
	case "ollama":
		return ollama.New(), nil
	default:
		return nil, fmt.Errorf("unknown built-in provider %q", name)
	}
}
```

To:

```go
func loadBuiltinProvider(name string, _ map[string]any) (sdk.ProviderPlugin, error) {
	switch name {
	case "anthropic":
		return anthropicapi.New(), nil
	case "anthropic-subscription":
		return anthropicsubscription.New(), nil
	case "claude-cli":
		return claudecli.New(), nil
	case "openai":
		return openai.New(), nil
	case "ollama":
		return ollama.New(), nil
	default:
		return nil, fmt.Errorf("unknown built-in provider %q", name)
	}
}
```

- [ ] **Step 3: Verify the whole module builds**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: PASS for all packages, including the providers we just created and any unrelated existing tests.

- [ ] **Step 5: Commit**

```bash
git add internal/plugins/registry.go internal/providers/anthropic/anthropic.go
git commit -m "feat(registry): register anthropic, anthropic-subscription, claude-cli; remove old provider"
```

---

## Task 18: Spec chapter

Per `CLAUDE.md`, ship the matching spec chapter with the code.

**Files:**
- Create: `specs/3.0.0-providers.md`

- [ ] **Step 1: Create the spec**

Create `specs/3.0.0-providers.md`:

````markdown
# 3.0.0 — Providers

Provider plugins implement `sdk.ProviderPlugin` and broker LLM inference for
ForgeBox. Each provider is registered in `internal/plugins/registry.go` and
configured under the top-level `providers:` key in `forgebox.yaml`.

## 3.1.0 — Anthropic providers

ForgeBox ships three first-class Anthropic-related providers covering the
three distinct ways to access Anthropic's models. Each is registered under a
separate name; they can be configured simultaneously.

### 3.1.1 — `anthropic` (API key)

Pay-per-use direct API access.

**Config:**

```yaml
providers:
  anthropic:
    api_key: sk-ant-api-...    # literal, env://, file://, or exec://
    base_url: ""               # optional; default https://api.anthropic.com/v1
    timeout_ms: 120000         # optional
```

**Accepted token format:** `sk-ant-api-*`. Subscription tokens
(`sk-ant-oat-*`) are rejected at `Init` with a directive to use the
`anthropic-subscription` provider.

**Beta headers sent:** `fine-grained-tool-streaming-2025-05-14`,
`interleaved-thinking-2025-05-14`.

**Errors:** `sdk.ErrAuth` (401/403), `sdk.ErrRateLimit` (429),
`sdk.ErrInputTooLarge` (400 with `context_length_exceeded`),
`sdk.ErrTransient` (5xx, network, timeout).

### 3.1.2 — `anthropic-subscription` (Claude Max OAuth)

Claude Max subscription via OAuth setup token (`sk-ant-oat01-*`) generated by
`claude setup-token`.

**Config:**

```yaml
providers:
  anthropic-subscription:
    token: sk-ant-oat01-...    # literal, env://, file://, or exec://
    base_url: ""
    timeout_ms: 120000
```

**Accepted token format:** `sk-ant-oat*`. API keys (`sk-ant-api-*`) are
rejected at `Init`.

**Beta headers sent:** `claude-code-20250219`, `oauth-2025-04-20`,
`fine-grained-tool-streaming-2025-05-14`, `interleaved-thinking-2025-05-14`.

**Gating rules** (each violation strips the disallowed input and logs WARN
once per request):

- 1M context window beta is stripped; subscription auth is limited to 200K.
- `cache_control` blocks are stripped from message content; prompt caching
  is API-key only.
- `Usage` returns token counts only; `EstimatedCostUSD` is left zero.

**Errors:** Same set as `anthropic`. 429 also covers subscription quota
exhaustion; the response body is preserved on the wrapped error.

### 3.1.3 — `claude-cli` (local CLI delegation)

Delegates inference to a locally installed `claude` binary. ForgeBox does
not authenticate; the binary owns its own credentials in `~/.claude` of the
gateway process user.

**Config:**

```yaml
providers:
  claude-cli:
    bin: /usr/local/bin/claude  # optional; defaults to PATH lookup of "claude"
    timeout_ms: 600000          # default 10 minutes
```

**Model namespace:** `claude-cli/<bare model id>`. The provider strips the
`claude-cli/` prefix when invoking the binary's `--model` flag.

**Invocation:** `claude -p --output-format stream-json --verbose --permission-mode bypassPermissions`.
The prompt is written to stdin; NDJSON envelopes are read from stdout.

**Errors:** `sdk.ErrCLIBackend` wraps non-zero exit codes (with stderr
attached) and malformed output. Context cancellation kills the subprocess.

## 3.2.0 — Provider registration

Built-in providers are loaded by `internal/plugins/registry.go` based on the
configured `providers:` map. Only providers with a configuration entry are
loaded. A misconfigured provider logs a WARN and is skipped (does not crash
gateway startup), but the gateway exits non-zero if every configured provider
fails to init.

## 3.3.0 — Secret references

Credential fields (`api_key`, `token`) on any provider accept literal values
or one of three reference forms resolved at `Init`:

| Form | Resolution |
|------|------------|
| `env://NAME` | `os.Getenv("NAME")` (errors if empty) |
| `file:///abs/path` | `os.ReadFile`, whitespace-trimmed |
| `exec://command args` | stdout of `command`, whitespace-trimmed |

Implemented once in `pkg/sdk/llmbase/auth/secretref.go`.
````

- [ ] **Step 2: Commit**

```bash
git add specs/3.0.0-providers.md
git commit -m "docs(specs): chapter 3.0.0 providers, sections 3.1-3.3"
```

---

## Task 19: Final verification

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS for all packages.

- [ ] **Step 2: Run the linter**

Run: `make lint`
Expected: clean. (If `golangci-lint` is unavailable, fall back to `go vet ./...`.)

- [ ] **Step 3: Verify gofumpt**

Run: `gofumpt -l pkg/sdk/llmbase internal/providers`
Expected: no output (no files would be reformatted).

If files are listed, run `gofumpt -w` on them and amend the relevant earlier commit OR add a tidy commit:

```bash
gofumpt -w pkg/sdk/llmbase internal/providers
git add -u
git commit -m "style: gofumpt"
```

- [ ] **Step 4: Build the binaries**

Run: `make build`
Expected: clean build of `forgebox` and `fb-agent`.

- [ ] **Step 5: Inspect git log**

Run: `git log --oneline main..HEAD`
Expected: a coherent series of commits, each compiling and passing tests on its own.

---

## Self-review checklist (already addressed)

- **Spec coverage:** every section of `docs/specs/2026-04-27-anthropic-providers-design.md` maps to at least one task above (sentinel errors → T1; llmbase → T2-T7; anthropic base → T8-T12; api provider → T13; subscription provider + gating → T14; claude-cli → T15-T16; registry/migration → T17; spec chapter → T18).
- **Placeholder scan:** no TBD/TODO/"add appropriate handling"/"similar to" markers. All code blocks are complete.
- **Type consistency:** `base.Provider`, `base.Options`, `base.Request`, `base.Response`, `base.MergeBetaHeader`, `base.APIKeyBetas`, `base.OAuthBetas`, `base.BetaContext1M`, `auth.Auth`, `auth.NewAPIKey`, `auth.NewOAuth`, `auth.ResolveSecret`, `llmbase.HTTPRunner`, `llmbase.NewHTTPRunner`, `llmbase.HTTPOptions`, `llmbase.SSEParser`, `llmbase.NewSSEParser`, `llmbase.SSEEvent`, `llmbase.StreamPump`, `llmbase.EventMapper`, `llmbase.ClassifyHTTPError` are referenced consistently across tasks.
