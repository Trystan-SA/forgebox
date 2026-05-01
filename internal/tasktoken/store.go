// Package tasktoken issues short-lived opaque tokens that map back to a
// (userID, taskID) pair. Used by the engine to authenticate in-VM tool
// callbacks to the gateway's own API.
package tasktoken

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

const Prefix = "fbtask_"

type entry struct {
	userID    string
	taskID    string
	expiresAt time.Time
}

// Store holds live task tokens in memory. Safe for concurrent use.
type Store struct {
	mu sync.RWMutex
	m  map[string]entry
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{m: make(map[string]entry)}
}

// Issue mints a fresh token bound to (userID, taskID) that expires after ttl.
func (s *Store) Issue(userID, taskID string, ttl time.Duration) string {
	var raw [32]byte
	_, _ = rand.Read(raw[:])
	tok := Prefix + base64.RawURLEncoding.EncodeToString(raw[:])
	s.mu.Lock()
	s.m[tok] = entry{
		userID:    userID,
		taskID:    taskID,
		expiresAt: time.Now().Add(ttl),
	}
	s.mu.Unlock()
	return tok
}

// Resolve returns the bound (userID, taskID) if the token is known and unexpired.
func (s *Store) Resolve(tok string) (string, string, bool) {
	s.mu.RLock()
	e, ok := s.m[tok]
	s.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return "", "", false
	}
	return e.userID, e.taskID, true
}

// Revoke removes the token. Idempotent.
func (s *Store) Revoke(tok string) {
	s.mu.Lock()
	delete(s.m, tok)
	s.mu.Unlock()
}
