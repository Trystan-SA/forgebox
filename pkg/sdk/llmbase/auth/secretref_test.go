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