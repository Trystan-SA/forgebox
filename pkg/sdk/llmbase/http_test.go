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
		{400, "some other 400", nil},
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
