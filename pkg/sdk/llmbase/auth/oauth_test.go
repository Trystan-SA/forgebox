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