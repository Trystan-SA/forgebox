package tasktoken

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_IssueResolveRevoke(t *testing.T) {
	tests := []struct {
		name string
		op   func(t *testing.T, s *Store)
	}{
		{
			name: "issue and resolve a fresh token",
			op: func(t *testing.T, s *Store) {
				t.Helper()
				tok := s.Issue("user-1", "task-1", time.Minute)
				assert.True(t, strings.HasPrefix(tok, "fbtask_"))
				userID, taskID, ok := s.Resolve(tok)
				require.True(t, ok)
				assert.Equal(t, "user-1", userID)
				assert.Equal(t, "task-1", taskID)
			},
		},
		{
			name: "expired token does not resolve",
			op: func(t *testing.T, s *Store) {
				t.Helper()
				tok := s.Issue("user-1", "task-1", time.Nanosecond)
				time.Sleep(2 * time.Millisecond)
				_, _, ok := s.Resolve(tok)
				assert.False(t, ok)
			},
		},
		{
			name: "revoked token does not resolve",
			op: func(t *testing.T, s *Store) {
				t.Helper()
				tok := s.Issue("user-1", "task-1", time.Minute)
				s.Revoke(tok)
				_, _, ok := s.Resolve(tok)
				assert.False(t, ok)
			},
		},
		{
			name: "unknown token does not resolve",
			op: func(t *testing.T, s *Store) {
				t.Helper()
				_, _, ok := s.Resolve("fbtask_does-not-exist")
				assert.False(t, ok)
			},
		},
		{
			name: "revoke is idempotent",
			op: func(t *testing.T, s *Store) {
				t.Helper()
				s.Revoke("fbtask_does-not-exist")
				// must not panic
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.op(t, NewStore())
		})
	}
}
