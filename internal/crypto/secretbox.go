// Package crypto implements authenticated encryption for secrets at rest.
//
// SecretBox uses AES-256-GCM with a key loaded from FORGEBOX_DB_ENCRYPTION_KEY.
// The on-wire format is base64url(version || nonce || ciphertext_with_tag).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// EnvKey is the environment variable holding the 32-byte master key.
// The value may be encoded as 64 hex chars or as raw-or-padded base64.
const EnvKey = "FORGEBOX_DB_ENCRYPTION_KEY"

// version1 is the current envelope format. Older versions remain decryptable
// after rotation; new writes always use the latest version.
const version1 byte = 0x01

// SecretBox seals and opens secrets with AES-256-GCM.
type SecretBox struct {
	aead cipher.AEAD
}

// NewFromEnv loads the master key from FORGEBOX_DB_ENCRYPTION_KEY.
func NewFromEnv() (*SecretBox, error) {
	raw := os.Getenv(EnvKey)
	if raw == "" {
		return nil, fmt.Errorf("%s is not set; provide a 32-byte key as hex or base64", EnvKey)
	}
	key, err := decodeKey(raw)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", EnvKey, err)
	}
	return New(key)
}

// New constructs a SecretBox from a 32-byte key.
func New(key []byte) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return &SecretBox{aead: aead}, nil
}

// Encrypt seals plaintext, returning a base64url-encoded envelope safe for DB storage.
func (s *SecretBox) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	out := make([]byte, 0, 1+len(nonce)+len(plaintext)+s.aead.Overhead())
	out = append(out, version1)
	out = append(out, nonce...)
	out = s.aead.Seal(out, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(out), nil
}

// Decrypt opens an envelope produced by Encrypt.
func (s *SecretBox) Decrypt(sealed string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(sealed)
	if err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if len(raw) < 1+s.aead.NonceSize()+s.aead.Overhead() {
		return nil, fmt.Errorf("envelope too short")
	}
	if raw[0] != version1 {
		return nil, fmt.Errorf("unsupported envelope version 0x%02x", raw[0])
	}
	nonce := raw[1 : 1+s.aead.NonceSize()]
	ct := raw[1+s.aead.NonceSize():]
	pt, err := s.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return pt, nil
}

func decodeKey(raw string) ([]byte, error) {
	if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, fmt.Errorf("key must be 32 bytes encoded as hex or base64")
}
