package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"normal password", "correct-horse-battery-staple"},
		{"empty password", ""},
		{"special characters", "p@$$w0rd!#%^&*()"},
		{"unicode", "пароль123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			require.NoError(t, err)
			assert.True(t,
				strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"),
				"expected bcrypt hash prefix, got %q", hash,
			)
			assert.NotEqual(t, tt.password, hash)
		})
	}
}

func TestHashPassword_ProducesDistinctHashes(t *testing.T) {
	hash1, err := HashPassword("secret")
	require.NoError(t, err)
	hash2, err := HashPassword("secret")
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2, "bcrypt uses random salts, hashes must differ")
}

func TestCheckPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		check    string
		want     bool
	}{
		{"correct password", "secret", "secret", true},
		{"wrong password", "secret", "wrong", false},
		{"empty against empty", "", "", true},
		{"empty against set", "secret", "", false},
		{"case sensitive", "Secret", "secret", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			require.NoError(t, err)
			assert.Equal(t, tt.want, CheckPassword(hash, tt.check))
		})
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	assert.False(t, CheckPassword("not-a-valid-bcrypt-hash", "anything"))
}