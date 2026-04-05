package sdk

import (
	"context"
	"time"
)

// StoragePlugin is the interface for persistent storage backends.
type StoragePlugin interface {
	Plugin

	TaskStore
	SessionStore
	AuditStore
	UserStore
}

// TaskStore manages task persistence.
type TaskStore interface {
	CreateTask(ctx context.Context, task *TaskRecord) error
	GetTask(ctx context.Context, id string) (*TaskRecord, error)
	UpdateTask(ctx context.Context, task *TaskRecord) error
	ListTasks(ctx context.Context, filter TaskFilter) ([]*TaskRecord, error)
}

// SessionStore manages session persistence.
type SessionStore interface {
	CreateSession(ctx context.Context, session *SessionRecord) error
	GetSession(ctx context.Context, id string) (*SessionRecord, error)
	UpdateSession(ctx context.Context, session *SessionRecord) error
	ListSessions(ctx context.Context, filter SessionFilter) ([]*SessionRecord, error)
	AppendMessage(ctx context.Context, sessionID string, msg *Message) error
	GetTranscript(ctx context.Context, sessionID string) ([]Message, error)
}

// AuditStore manages audit log persistence.
type AuditStore interface {
	LogAuditEntry(ctx context.Context, entry *AuditEntry) error
	ListAuditEntries(ctx context.Context, filter AuditFilter) ([]*AuditEntry, error)
}

// UserStore manages user and team data.
type UserStore interface {
	GetUser(ctx context.Context, id string) (*UserRecord, error)
	CreateUser(ctx context.Context, user *UserRecord) error
	ListUsers(ctx context.Context) ([]*UserRecord, error)
}

// TaskRecord represents a stored task.
type TaskRecord struct {
	ID          string     `json:"id"`
	Status      TaskStatus `json:"status"`
	Prompt      string     `json:"prompt"`
	Result      string     `json:"result,omitempty"`
	Provider    string     `json:"provider"`
	Model       string     `json:"model"`
	UserID      string     `json:"user_id"`
	SessionID   string     `json:"session_id"`
	Cost        float64    `json:"cost"`
	TokensIn    int        `json:"tokens_in"`
	TokensOut   int        `json:"tokens_out"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// TaskStatus is the lifecycle state of a task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// TaskFilter specifies criteria for listing tasks.
type TaskFilter struct {
	UserID string
	Status TaskStatus
	Limit  int
	Offset int
}

// SessionRecord represents a stored session.
type SessionRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionFilter specifies criteria for listing sessions.
type SessionFilter struct {
	UserID string
	Limit  int
	Offset int
}

// AuditEntry records a permission decision or significant action.
type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	TaskID    string    `json:"task_id"`
	Action    string    `json:"action"`
	Tool      string    `json:"tool,omitempty"`
	Decision  string    `json:"decision"` // "allow", "deny"
	Reason    string    `json:"reason,omitempty"`
}

// AuditFilter specifies criteria for listing audit entries.
type AuditFilter struct {
	UserID string
	TaskID string
	Limit  int
	Offset int
}

// UserRecord represents a stored user.
type UserRecord struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Role     string   `json:"role"` // "admin", "developer", "operator", "viewer"
	TeamIDs  []string `json:"team_ids,omitempty"`
	Disabled bool     `json:"disabled"`
}
