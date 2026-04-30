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