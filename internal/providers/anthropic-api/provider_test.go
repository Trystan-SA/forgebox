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
