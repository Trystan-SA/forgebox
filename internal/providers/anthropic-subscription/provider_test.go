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
		{"api key rejected", map[string]any{"token": "sk-ant-api-abc"}, "anthropic-api provider"},
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
