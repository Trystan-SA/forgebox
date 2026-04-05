// Package sqlite implements ForgeBox storage using SQLite.
//
// SQLite is the default storage backend — zero configuration required.
// For production deployments with multiple gateway instances, use PostgreSQL.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Store implements sdk.StoragePlugin using SQLite.
type Store struct {
	db *sql.DB
}

// New opens or creates a SQLite database at the given path.
func New(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error { return s.db.Close() }

// Plugin interface stubs.
func (s *Store) Name() string    { return "sqlite" }
func (s *Store) Version() string { return "1.0.0" }
func (s *Store) Init(_ context.Context, _ map[string]any) error { return nil }
func (s *Store) Shutdown(_ context.Context) error               { return s.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL DEFAULT 'pending',
			prompt TEXT NOT NULL,
			result TEXT,
			provider TEXT,
			model TEXT,
			user_id TEXT,
			session_id TEXT,
			cost REAL DEFAULT 0,
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			error TEXT,
			created_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT
		);
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			provider TEXT,
			model TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT,
			tool_calls TEXT,
			tool_results TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);
		CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL,
			user_id TEXT,
			task_id TEXT,
			action TEXT NOT NULL,
			tool TEXT,
			decision TEXT NOT NULL,
			reason TEXT
		);
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			role TEXT NOT NULL DEFAULT 'viewer',
			team_ids TEXT,
			disabled INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_tasks_user ON tasks(user_id);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
		CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id);
		CREATE INDEX IF NOT EXISTS idx_audit_task ON audit_log(task_id);
	`)
	return err
}

// --- TaskStore ---

func (s *Store) CreateTask(ctx context.Context, task *sdk.TaskRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (id, status, prompt, provider, model, user_id, session_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Status, task.Prompt, task.Provider, task.Model,
		task.UserID, task.SessionID, task.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetTask(ctx context.Context, id string) (*sdk.TaskRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, status, prompt, result, provider, model, user_id, cost, tokens_in, tokens_out, error, created_at
		 FROM tasks WHERE id = ?`, id)

	var t sdk.TaskRecord
	var createdAt string
	var result, errStr sql.NullString
	err := row.Scan(&t.ID, &t.Status, &t.Prompt, &result, &t.Provider, &t.Model,
		&t.UserID, &t.Cost, &t.TokensIn, &t.TokensOut, &errStr, &createdAt)
	if err != nil {
		return nil, err
	}
	t.Result = result.String
	t.Error = errStr.String
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &t, nil
}

func (s *Store) UpdateTask(ctx context.Context, task *sdk.TaskRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status=?, result=?, cost=?, tokens_in=?, tokens_out=?, error=?, completed_at=?
		 WHERE id=?`,
		task.Status, task.Result, task.Cost, task.TokensIn, task.TokensOut, task.Error,
		timePtr(task.CompletedAt), task.ID,
	)
	return err
}

func (s *Store) ListTasks(ctx context.Context, filter sdk.TaskFilter) ([]*sdk.TaskRecord, error) {
	query := `SELECT id, status, prompt, provider, model, user_id, cost, created_at FROM tasks WHERE 1=1`
	args := []any{}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*sdk.TaskRecord
	for rows.Next() {
		var t sdk.TaskRecord
		var createdAt string
		if err := rows.Scan(&t.ID, &t.Status, &t.Prompt, &t.Provider, &t.Model, &t.UserID, &t.Cost, &createdAt); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

// --- SessionStore ---

func (s *Store) CreateSession(ctx context.Context, session *sdk.SessionRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, provider, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.Provider, session.Model,
		session.CreatedAt.Format(time.RFC3339), session.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetSession(ctx context.Context, id string) (*sdk.SessionRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, provider, model, created_at, updated_at FROM sessions WHERE id = ?`, id)
	var sess sdk.SessionRecord
	var createdAt, updatedAt string
	err := row.Scan(&sess.ID, &sess.UserID, &sess.Provider, &sess.Model, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sess.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &sess, nil
}

func (s *Store) UpdateSession(ctx context.Context, session *sdk.SessionRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET updated_at = ? WHERE id = ?`,
		session.UpdatedAt.Format(time.RFC3339), session.ID,
	)
	return err
}

func (s *Store) ListSessions(ctx context.Context, filter sdk.SessionFilter) ([]*sdk.SessionRecord, error) {
	query := `SELECT id, user_id, provider, model, created_at, updated_at FROM sessions WHERE 1=1`
	args := []any{}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	query += " ORDER BY updated_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*sdk.SessionRecord
	for rows.Next() {
		var sess sdk.SessionRecord
		var createdAt, updatedAt string
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.Provider, &sess.Model, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		sess.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		sessions = append(sessions, &sess)
	}
	return sessions, rows.Err()
}

func (s *Store) AppendMessage(ctx context.Context, sessionID string, msg *sdk.Message) error {
	toolCalls, _ := json.Marshal(msg.ToolCalls)
	toolResults, _ := json.Marshal(msg.ToolResults)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, session_id, role, content, tool_calls, tool_results, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), sessionID, msg.Role, msg.Content,
		string(toolCalls), string(toolResults), time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetTranscript(ctx context.Context, sessionID string) ([]sdk.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT role, content, tool_calls, tool_results FROM messages WHERE session_id = ? ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []sdk.Message
	for rows.Next() {
		var msg sdk.Message
		var toolCalls, toolResults string
		if err := rows.Scan(&msg.Role, &msg.Content, &toolCalls, &toolResults); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(toolCalls), &msg.ToolCalls)
		json.Unmarshal([]byte(toolResults), &msg.ToolResults)
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// --- AuditStore ---

func (s *Store) LogAuditEntry(ctx context.Context, entry *sdk.AuditEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, timestamp, user_id, task_id, action, tool, decision, reason) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Timestamp.Format(time.RFC3339), entry.UserID, entry.TaskID,
		entry.Action, entry.Tool, entry.Decision, entry.Reason,
	)
	return err
}

func (s *Store) ListAuditEntries(ctx context.Context, filter sdk.AuditFilter) ([]*sdk.AuditEntry, error) {
	query := `SELECT id, timestamp, user_id, task_id, action, tool, decision, reason FROM audit_log WHERE 1=1`
	args := []any{}
	if filter.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}
	if filter.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, filter.TaskID)
	}
	query += " ORDER BY timestamp DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*sdk.AuditEntry
	for rows.Next() {
		var e sdk.AuditEntry
		var ts string
		if err := rows.Scan(&e.ID, &ts, &e.UserID, &e.TaskID, &e.Action, &e.Tool, &e.Decision, &e.Reason); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

// --- UserStore ---

func (s *Store) GetUser(ctx context.Context, id string) (*sdk.UserRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, email, role, disabled FROM users WHERE id = ?`, id)
	var u sdk.UserRecord
	var disabled int
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &disabled); err != nil {
		return nil, err
	}
	u.Disabled = disabled != 0
	return &u, nil
}

func (s *Store) CreateUser(ctx context.Context, user *sdk.UserRecord) error {
	disabled := 0
	if user.Disabled {
		disabled = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, name, email, role, disabled) VALUES (?, ?, ?, ?, ?)`,
		user.ID, user.Name, user.Email, user.Role, disabled,
	)
	return err
}

func (s *Store) ListUsers(ctx context.Context) ([]*sdk.UserRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, email, role, disabled FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*sdk.UserRecord
	for rows.Next() {
		var u sdk.UserRecord
		var disabled int
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &disabled); err != nil {
			return nil, err
		}
		u.Disabled = disabled != 0
		users = append(users, &u)
	}
	return users, rows.Err()
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
