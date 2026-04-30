package crypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 32 bytes hex

func mustNew(t *testing.T) *SecretBox {
	t.Helper()
	t.Setenv(EnvKey, testKey)
	sb, err := NewFromEnv()
	require.NoError(t, err)
	return sb
}

func TestNewFromEnv_RequiresKey(t *testing.T) {
	t.Setenv(EnvKey, "")
	_, err := NewFromEnv()
	require.Error(t, err)
	require.Contains(t, err.Error(), EnvKey)
}

func TestNewFromEnv_RejectsShortKey(t *testing.T) {
	t.Setenv(EnvKey, "deadbeef")
	_, err := NewFromEnv()
	require.Error(t, err)
}

func TestNewFromEnv_AcceptsHexAndBase64(t *testing.T) {
	cases := map[string]string{
		"hex":    testKey,
		"base64": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32)),
	}
	for name, k := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(EnvKey, k)
			_, err := NewFromEnv()
			require.NoError(t, err)
		})
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	sb := mustNew(t)
	cases := []string{"", "hello", strings.Repeat("x", 4096), "sk-ant-oat01-" + strings.Repeat("y", 80)}
	for _, pt := range cases {
		sealed, err := sb.Encrypt([]byte(pt))
		require.NoError(t, err)
		if pt != "" {
			require.NotContains(t, sealed, pt, "ciphertext must not contain plaintext")
		}
		got, err := sb.Decrypt(sealed)
		require.NoError(t, err)
		require.Equal(t, pt, string(got))
	}
}

func TestEncrypt_FreshNonceEachCall(t *testing.T) {
	sb := mustNew(t)
	a, err := sb.Encrypt([]byte("same"))
	require.NoError(t, err)
	b, err := sb.Encrypt([]byte("same"))
	require.NoError(t, err)
	require.NotEqual(t, a, b, "nonce must change per call")
}

func TestDecrypt_RejectsTamperedCiphertext(t *testing.T) {
	sb := mustNew(t)
	sealed, err := sb.Encrypt([]byte("hello"))
	require.NoError(t, err)
	// Flip one byte in the middle
	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	require.NoError(t, err)
	raw[len(raw)/2] ^= 0xFF
	tampered := base64.RawURLEncoding.EncodeToString(raw)
	_, err = sb.Decrypt(tampered)
	require.Error(t, err)
}

func TestDecrypt_RejectsUnknownVersion(t *testing.T) {
	sb := mustNew(t)
	bogus := append([]byte{0xFF}, bytes.Repeat([]byte{0}, 40)...)
	_, err := sb.Decrypt(base64.RawURLEncoding.EncodeToString(bogus))
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}
