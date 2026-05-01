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
	AutomationStore
	AppStore
	ProviderStore
	AgentStore
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
	GetUserByEmail(ctx context.Context, email string) (*UserRecord, error)
	CreateUser(ctx context.Context, user *UserRecord) error
	ListUsers(ctx context.Context) ([]*UserRecord, error)
	CountUsers(ctx context.Context) (int, error)
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

// Task status values.
const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "canceled"
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
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"-"`    // never exposed in JSON
	Role         string   `json:"role"` // "admin", "developer", "operator", "viewer"
	TeamIDs      []string `json:"team_ids,omitempty"`
	Disabled     bool     `json:"disabled"`
}

// AppStore manages app persistence.
type AppStore interface {
	CreateApp(ctx context.Context, app *AppRecord) error
	GetApp(ctx context.Context, id string) (*AppRecord, error)
	UpdateApp(ctx context.Context, app *AppRecord) error
	DeleteApp(ctx context.Context, id string) error
	ListApps(ctx context.Context, filter AppFilter) ([]*AppRecord, error)
}

// AutomationStore manages automation persistence.
type AutomationStore interface {
	CreateAutomation(ctx context.Context, automation *AutomationRecord) error
	GetAutomation(ctx context.Context, id string) (*AutomationRecord, error)
	UpdateAutomation(ctx context.Context, automation *AutomationRecord) error
	DeleteAutomation(ctx context.Context, id string) error
	ListAutomations(ctx context.Context, filter AutomationFilter) ([]*AutomationRecord, error)
}

// AutomationRecord represents a stored automation workflow.
type AutomationRecord struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	Sharing     string    `json:"sharing"` // "personal", "team", "org"
	TeamID      string    `json:"team_id,omitempty"`
	Trigger     string    `json:"trigger"` // JSON blob
	Nodes       string    `json:"nodes"`   // JSON blob (Svelte Flow nodes)
	Edges       string    `json:"edges"`   // JSON blob (Svelte Flow edges)
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AutomationFilter specifies criteria for listing automations.
type AutomationFilter struct {
	UserID string
	TeamID string
	Limit  int
	Offset int
}

// AppStatus is the lifecycle state of an app.
type AppStatus string

// App status values.
const (
	AppDraft     AppStatus = "draft"
	AppDeploying AppStatus = "deploying"
	AppRunning   AppStatus = "running"
	AppStopped   AppStatus = "stopped"
	AppError     AppStatus = "error"
)

// AppRecord represents a stored internal tool app.
type AppRecord struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	Sharing     string    `json:"sharing"` // "personal", "team", "org"
	TeamID      string    `json:"team_id,omitempty"`
	Status      AppStatus `json:"status"`
	Tools       string    `json:"tools"`  // JSON array of granted tools
	Config      string    `json:"config"` // JSON blob for VM/model config
	URL         string    `json:"url"`    // iframe URL when running
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AppFilter specifies criteria for listing apps.
type AppFilter struct {
	UserID string
	TeamID string
	Status AppStatus
	Limit  int
	Offset int
}

// ProviderStore manages user-configured LLM provider credentials.
//
// Providers are global to the install (no per-user scoping). The Config field
// is opaque to the store — it is the AEAD-sealed envelope produced by
// internal/crypto.SecretBox. The store never sees plaintext secrets.
type ProviderStore interface {
	CreateProvider(ctx context.Context, p *ProviderRecord) error
	GetProvider(ctx context.Context, id string) (*ProviderRecord, error)
	UpdateProvider(ctx context.Context, p *ProviderRecord) error
	DeleteProvider(ctx context.Context, id string) error
	ListProviders(ctx context.Context) ([]*ProviderRecord, error)
}

// ProviderRecord is one configured provider instance.
type ProviderRecord struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"` // e.g. "anthropic", "anthropic-subscription"
	Name            string    `json:"name"` // user-supplied display name; unique; used as registry key
	ConfigEncrypted string    `json:"-"`    // AEAD-sealed JSON of the provider config map
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AgentStore manages agent persistence. See specs/1.0.0-agents.md.
type AgentStore interface {
	CreateAgent(ctx context.Context, agent *AgentRecord) error
	GetAgent(ctx context.Context, id string) (*AgentRecord, error)
	UpdateAgent(ctx context.Context, agent *AgentRecord) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context, filter AgentFilter) ([]*AgentRecord, error)
}

// AgentRecord is one stored agent. Tools is a JSON-encoded array of tool
// names, mirroring AppRecord.Tools so the column type stays uniform.
type AgentRecord struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Role         string    `json:"role"` // "worker" or "orchestrator"
	SystemPrompt string    `json:"system_prompt"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Tools        string    `json:"tools"`   // JSON array of tool names
	Sharing      string    `json:"sharing"` // "personal", "team", "org"
	TeamID       string    `json:"team_id,omitempty"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AgentFilter specifies criteria for listing agents.
type AgentFilter struct {
	UserID string
	TeamID string
	Limit  int
	Offset int
}
