// Package permissions implements the multi-layered permission system.
//
// Every tool invocation passes through the permission checker before execution.
// The checker evaluates org policies, RBAC roles, and task-scoped restrictions
// to determine whether a tool call is allowed.
package permissions

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Checker evaluates permissions for tool invocations.
type Checker struct {
	authCfg config.AuthConfig
	store   sdk.AuditStore
	// orgPolicies are loaded from configuration.
	orgPolicies []Policy
}

// Policy is an org-level permission rule.
type Policy struct {
	// Role is the role this policy applies to ("*" for all).
	Role string `json:"role" yaml:"role"`
	// Tool is the tool name this policy applies to ("*" for all).
	Tool string `json:"tool" yaml:"tool"`
	// Action is "allow" or "deny".
	Action string `json:"action" yaml:"action"`
	// Reason is a human-readable explanation.
	Reason string `json:"reason,omitempty" yaml:"reason"`
}

// NewChecker creates a permission checker.
func NewChecker(authCfg config.AuthConfig, store sdk.AuditStore) *Checker {
	return &Checker{
		authCfg: authCfg,
		store:   store,
	}
}

// Check evaluates whether a user is allowed to invoke a tool.
// Returns (allowed, reason).
func (c *Checker) Check(userID, toolName string, input json.RawMessage) (bool, string) {
	// Layer 1: Org policies (admin-defined).
	if allowed, reason := c.checkOrgPolicies(userID, toolName); !allowed {
		c.audit(userID, toolName, "deny", reason)
		return false, reason
	}

	// Layer 2: RBAC (role-based access control).
	if allowed, reason := c.checkRBAC(userID, toolName); !allowed {
		c.audit(userID, toolName, "deny", reason)
		return false, reason
	}

	// Layer 3: Tool-level checks (destructive operations need elevated roles).
	// This is evaluated by the engine based on ToolPlugin.IsDestructive().

	c.audit(userID, toolName, "allow", "passed all checks")
	return true, ""
}

func (c *Checker) checkOrgPolicies(userID, toolName string) (bool, string) {
	for _, p := range c.orgPolicies {
		if (p.Tool == "*" || p.Tool == toolName) && p.Action == "deny" {
			return false, p.Reason
		}
	}
	return true, ""
}

func (c *Checker) checkRBAC(userID, toolName string) (bool, string) {
	// TODO: Look up user role from store and check against tool requirements.
	// For now, allow all authenticated users.
	if userID == "" || userID == "anonymous" {
		return false, "authentication required"
	}
	return true, ""
}

// SetOrgPolicies loads organization-level policies.
func (c *Checker) SetOrgPolicies(policies []Policy) {
	c.orgPolicies = policies
}

func (c *Checker) audit(userID, tool, decision, reason string) {
	entry := &sdk.AuditEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		UserID:    userID,
		Tool:      tool,
		Action:    "tool_call",
		Decision:  decision,
		Reason:    reason,
	}
	if err := c.store.LogAuditEntry(context.Background(), entry); err != nil {
		slog.Warn("failed to log audit entry", "error", err)
	}
}
