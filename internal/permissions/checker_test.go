package permissions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuditStore struct {
	entries []*sdk.AuditEntry
}

func (m *mockAuditStore) LogAuditEntry(_ context.Context, entry *sdk.AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditStore) ListAuditEntries(_ context.Context, _ sdk.AuditFilter) ([]*sdk.AuditEntry, error) {
	return m.entries, nil
}

func newTestChecker(policies []Policy) (*Checker, *mockAuditStore) {
	store := &mockAuditStore{}
	checker := NewChecker(config.AuthConfig{Method: "local"}, store)
	checker.SetOrgPolicies(policies)
	return checker, store
}

func TestCheck_AllowsAuthenticatedUser(t *testing.T) {
	checker, _ := newTestChecker(nil)
	allowed, reason := checker.Check("user-123", "bash", json.RawMessage(`{}`))
	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestCheck_DeniesEmptyUserID(t *testing.T) {
	checker, _ := newTestChecker(nil)
	allowed, reason := checker.Check("", "bash", json.RawMessage(`{}`))
	assert.False(t, allowed)
	assert.Equal(t, "authentication required", reason)
}

func TestCheck_DeniesAnonymousUserID(t *testing.T) {
	checker, _ := newTestChecker(nil)
	allowed, reason := checker.Check("anonymous", "bash", json.RawMessage(`{}`))
	assert.False(t, allowed)
	assert.NotEmpty(t, reason)
}

func TestCheck_OrgPolicyDeniesSpecificTool(t *testing.T) {
	policies := []Policy{
		{Tool: "bash", Action: "deny", Reason: "bash not allowed"},
	}
	checker, _ := newTestChecker(policies)
	allowed, reason := checker.Check("user-123", "bash", json.RawMessage(`{}`))
	assert.False(t, allowed)
	assert.Equal(t, "bash not allowed", reason)
}

func TestCheck_OrgPolicyWildcardDeniesAllTools(t *testing.T) {
	policies := []Policy{
		{Tool: "*", Action: "deny", Reason: "all tools blocked"},
	}
	checker, _ := newTestChecker(policies)
	allowed, reason := checker.Check("user-123", "file_read", json.RawMessage(`{}`))
	assert.False(t, allowed)
	assert.Equal(t, "all tools blocked", reason)
}

func TestCheck_OrgPolicyDoesNotDenyOtherTool(t *testing.T) {
	policies := []Policy{
		{Tool: "bash", Action: "deny", Reason: "bash not allowed"},
	}
	checker, _ := newTestChecker(policies)
	allowed, _ := checker.Check("user-123", "file_read", json.RawMessage(`{}`))
	assert.True(t, allowed)
}

func TestCheck_AuditsAllowDecision(t *testing.T) {
	checker, store := newTestChecker(nil)
	checker.Check("user-123", "bash", json.RawMessage(`{}`))
	require.Len(t, store.entries, 1)
	assert.Equal(t, "allow", store.entries[0].Decision)
	assert.Equal(t, "user-123", store.entries[0].UserID)
	assert.Equal(t, "bash", store.entries[0].Tool)
	assert.NotEmpty(t, store.entries[0].ID)
}

func TestCheck_AuditsDenyDecision(t *testing.T) {
	policies := []Policy{{Tool: "bash", Action: "deny", Reason: "blocked"}}
	checker, store := newTestChecker(policies)
	checker.Check("user-123", "bash", json.RawMessage(`{}`))
	require.Len(t, store.entries, 1)
	assert.Equal(t, "deny", store.entries[0].Decision)
}

func TestSetOrgPolicies_ReplacesExisting(t *testing.T) {
	policies := []Policy{{Tool: "bash", Action: "deny", Reason: "initial"}}
	checker, _ := newTestChecker(policies)

	// Replacing with empty policies allows the tool again.
	checker.SetOrgPolicies(nil)
	allowed, _ := checker.Check("user-123", "bash", json.RawMessage(`{}`))
	assert.True(t, allowed)
}
