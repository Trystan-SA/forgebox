package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// --- TaskStore ---

// CreateTask persists a new task record.
func (s *Store) CreateTask(ctx context.Context, task *sdk.TaskRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (id, status, prompt, provider, model, user_id, session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		task.ID, task.Status, task.Prompt, task.Provider, task.Model,
		task.UserID, task.SessionID, task.CreatedAt,
	)
	return err
}

// GetTask retrieves a task by ID.
func (s *Store) GetTask(ctx context.Context, id string) (*sdk.TaskRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, status, prompt, result, provider, model, user_id, cost, tokens_in, tokens_out, error, created_at
		 FROM tasks WHERE id = $1`, id)

	var t sdk.TaskRecord
	var result, errStr sql.NullString
	if err := row.Scan(&t.ID, &t.Status, &t.Prompt, &result, &t.Provider, &t.Model,
		&t.UserID, &t.Cost, &t.TokensIn, &t.TokensOut, &errStr, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.Result = result.String
	t.Error = errStr.String
	return &t, nil
}

// UpdateTask updates status, result, and usage for a task.
func (s *Store) UpdateTask(ctx context.Context, task *sdk.TaskRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status=$1, result=$2, cost=$3, tokens_in=$4, tokens_out=$5, error=$6, completed_at=$7
		 WHERE id=$8`,
		task.Status, task.Result, task.Cost, task.TokensIn, task.TokensOut, task.Error,
		task.CompletedAt, task.ID,
	)
	return err
}

// ListTasks returns tasks matching the given filter.
func (s *Store) ListTasks(ctx context.Context, filter sdk.TaskFilter) ([]*sdk.TaskRecord, error) {
	query := `SELECT id, status, prompt, provider, model, user_id, cost, created_at FROM tasks WHERE 1=1`
	args := []any{}
	i := 1
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", i)
		args = append(args, filter.UserID)
		i++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", i)
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
	defer func() { _ = rows.Close() }()

	var tasks []*sdk.TaskRecord
	for rows.Next() {
		var t sdk.TaskRecord
		if err := rows.Scan(&t.ID, &t.Status, &t.Prompt, &t.Provider, &t.Model, &t.UserID, &t.Cost, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, rows.Err()
}

// --- SessionStore ---

// CreateSession persists a new session record.
func (s *Store) CreateSession(ctx context.Context, session *sdk.SessionRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, provider, model, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		session.ID, session.UserID, session.Provider, session.Model,
		session.CreatedAt, session.UpdatedAt,
	)
	return err
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(ctx context.Context, id string) (*sdk.SessionRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, provider, model, created_at, updated_at FROM sessions WHERE id = $1`, id)
	var sess sdk.SessionRecord
	if err := row.Scan(&sess.ID, &sess.UserID, &sess.Provider, &sess.Model, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
		return nil, err
	}
	return &sess, nil
}

// UpdateSession updates the updated_at timestamp for a session.
func (s *Store) UpdateSession(ctx context.Context, session *sdk.SessionRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET updated_at = $1 WHERE id = $2`,
		session.UpdatedAt, session.ID,
	)
	return err
}

// ListSessions returns sessions matching the given filter.
func (s *Store) ListSessions(ctx context.Context, filter sdk.SessionFilter) ([]*sdk.SessionRecord, error) {
	query := `SELECT id, user_id, provider, model, created_at, updated_at FROM sessions WHERE 1=1`
	args := []any{}
	i := 1
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", i)
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
	defer func() { _ = rows.Close() }()

	var sessions []*sdk.SessionRecord
	for rows.Next() {
		var sess sdk.SessionRecord
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.Provider, &sess.Model, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, &sess)
	}
	return sessions, rows.Err()
}

// AppendMessage adds a message to a session transcript.
func (s *Store) AppendMessage(ctx context.Context, sessionID string, msg *sdk.Message) error {
	toolCalls, _ := json.Marshal(msg.ToolCalls)
	toolResults, _ := json.Marshal(msg.ToolResults)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, session_id, role, content, tool_calls, tool_results, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New().String(), sessionID, msg.Role, msg.Content,
		string(toolCalls), string(toolResults), time.Now().UTC(),
	)
	return err
}

// GetTranscript retrieves all messages for a session in chronological order.
func (s *Store) GetTranscript(ctx context.Context, sessionID string) ([]sdk.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT role, content, tool_calls, tool_results FROM messages WHERE session_id = $1 ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []sdk.Message
	for rows.Next() {
		var msg sdk.Message
		var content sql.NullString
		var toolCalls, toolResults sql.NullString
		if err := rows.Scan(&msg.Role, &content, &toolCalls, &toolResults); err != nil {
			return nil, err
		}
		msg.Content = content.String
		if toolCalls.Valid {
			_ = json.Unmarshal([]byte(toolCalls.String), &msg.ToolCalls)
		}
		if toolResults.Valid {
			_ = json.Unmarshal([]byte(toolResults.String), &msg.ToolResults)
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// --- AuditStore ---

// LogAuditEntry records an audit log entry.
func (s *Store) LogAuditEntry(ctx context.Context, entry *sdk.AuditEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, timestamp, user_id, task_id, action, tool, decision, reason) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		entry.ID, entry.Timestamp, entry.UserID, entry.TaskID,
		entry.Action, entry.Tool, entry.Decision, entry.Reason,
	)
	return err
}

// ListAuditEntries returns audit entries matching the given filter.
func (s *Store) ListAuditEntries(ctx context.Context, filter sdk.AuditFilter) ([]*sdk.AuditEntry, error) {
	query := `SELECT id, timestamp, user_id, task_id, action, tool, decision, reason FROM audit_log WHERE 1=1`
	args := []any{}
	i := 1
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", i)
		args = append(args, filter.UserID)
		i++
	}
	if filter.TaskID != "" {
		query += fmt.Sprintf(" AND task_id = $%d", i)
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
	defer func() { _ = rows.Close() }()

	var entries []*sdk.AuditEntry
	for rows.Next() {
		var e sdk.AuditEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.UserID, &e.TaskID, &e.Action, &e.Tool, &e.Decision, &e.Reason); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

// --- UserStore ---

// GetUser retrieves a user by ID.
func (s *Store) GetUser(ctx context.Context, id string) (*sdk.UserRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, email, password_hash, role, disabled FROM users WHERE id = $1`, id)
	var u sdk.UserRecord
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.Disabled); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByEmail retrieves a user by email address.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*sdk.UserRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, email, password_hash, role, disabled FROM users WHERE email = $1`, email)
	var u sdk.UserRecord
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.Disabled); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser persists a new user record.
func (s *Store) CreateUser(ctx context.Context, user *sdk.UserRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, name, email, password_hash, role, disabled) VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Name, user.Email, user.PasswordHash, user.Role, user.Disabled,
	)
	return err
}

// ListUsers returns all users ordered by name.
func (s *Store) ListUsers(ctx context.Context) ([]*sdk.UserRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, email, password_hash, role, disabled FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var users []*sdk.UserRecord
	for rows.Next() {
		var u sdk.UserRecord
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.Disabled); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// CountUsers returns the total number of user records.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// --- AutomationStore ---

// CreateAutomation persists a new automation record.
func (s *Store) CreateAutomation(ctx context.Context, a *sdk.AutomationRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO automations (id, name, description, created_by, sharing, team_id, trigger_config, nodes, edges, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		a.ID, a.Name, a.Description, a.CreatedBy, a.Sharing, a.TeamID,
		a.Trigger, a.Nodes, a.Edges, a.Enabled,
		a.CreatedAt, a.UpdatedAt,
	)
	return err
}

// GetAutomation retrieves an automation by ID.
func (s *Store) GetAutomation(ctx context.Context, id string) (*sdk.AutomationRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_by, sharing, team_id, trigger_config, nodes, edges, enabled, created_at, updated_at
		 FROM automations WHERE id = $1`, id)
	return scanAutomation(row)
}

// UpdateAutomation updates a stored automation record.
func (s *Store) UpdateAutomation(ctx context.Context, a *sdk.AutomationRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE automations SET name=$1, description=$2, sharing=$3, team_id=$4, trigger_config=$5, nodes=$6, edges=$7, enabled=$8, updated_at=$9
		 WHERE id=$10`,
		a.Name, a.Description, a.Sharing, a.TeamID,
		a.Trigger, a.Nodes, a.Edges, a.Enabled,
		a.UpdatedAt, a.ID,
	)
	return err
}

// DeleteAutomation removes an automation record.
func (s *Store) DeleteAutomation(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM automations WHERE id = $1`, id)
	return err
}

// ListAutomations returns automations matching the given filter.
func (s *Store) ListAutomations(ctx context.Context, filter sdk.AutomationFilter) ([]*sdk.AutomationRecord, error) {
	query := `SELECT id, name, description, created_by, sharing, team_id, trigger_config, nodes, edges, enabled, created_at, updated_at FROM automations WHERE 1=1`
	args := []any{}
	i := 1
	if filter.UserID != "" {
		query += fmt.Sprintf(` AND (created_by = $%d OR sharing = 'org' OR (sharing = 'team' AND team_id IN (SELECT team_ids FROM users WHERE id = $%d)))`, i, i)
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
	defer func() { _ = rows.Close() }()

	var automations []*sdk.AutomationRecord
	for rows.Next() {
		a, err := scanAutomationRow(rows)
		if err != nil {
			return nil, err
		}
		automations = append(automations, a)
	}
	return automations, rows.Err()
}

func scanAutomation(row *sql.Row) (*sdk.AutomationRecord, error) {
	var a sdk.AutomationRecord
	var teamID sql.NullString
	if err := row.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedBy, &a.Sharing, &teamID,
		&a.Trigger, &a.Nodes, &a.Edges, &a.Enabled, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.TeamID = teamID.String
	return &a, nil
}

func scanAutomationRow(rows *sql.Rows) (*sdk.AutomationRecord, error) {
	var a sdk.AutomationRecord
	var teamID sql.NullString
	if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedBy, &a.Sharing, &teamID,
		&a.Trigger, &a.Nodes, &a.Edges, &a.Enabled, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.TeamID = teamID.String
	return &a, nil
}

// --- AppStore ---

// CreateApp persists a new app record.
func (s *Store) CreateApp(ctx context.Context, app *sdk.AppRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO apps (id, name, description, created_by, sharing, team_id, status, tools, config, url, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		app.ID, app.Name, app.Description, app.CreatedBy, app.Sharing, app.TeamID,
		app.Status, app.Tools, app.Config, app.URL, app.Enabled,
		app.CreatedAt, app.UpdatedAt,
	)
	return err
}

// GetApp retrieves an app by ID.
func (s *Store) GetApp(ctx context.Context, id string) (*sdk.AppRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_by, sharing, team_id, status, tools, config, url, enabled, created_at, updated_at
		 FROM apps WHERE id = $1`, id)
	return scanApp(row)
}

// UpdateApp replaces the mutable fields of an existing app record.
func (s *Store) UpdateApp(ctx context.Context, app *sdk.AppRecord) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE apps SET name=$1, description=$2, sharing=$3, team_id=$4, status=$5, tools=$6, config=$7, url=$8, enabled=$9, updated_at=$10
		 WHERE id=$11`,
		app.Name, app.Description, app.Sharing, app.TeamID,
		app.Status, app.Tools, app.Config, app.URL, app.Enabled,
		app.UpdatedAt, app.ID,
	)
	return err
}

// DeleteApp removes an app record by ID.
func (s *Store) DeleteApp(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM apps WHERE id = $1`, id)
	return err
}

// ListApps returns apps matching the given filter.
func (s *Store) ListApps(ctx context.Context, filter sdk.AppFilter) ([]*sdk.AppRecord, error) {
	query := `SELECT id, name, description, created_by, sharing, team_id, status, tools, config, url, enabled, created_at, updated_at FROM apps WHERE 1=1`
	args := []any{}
	i := 1
	if filter.UserID != "" {
		query += fmt.Sprintf(` AND (created_by = $%d OR sharing = 'org' OR (sharing = 'team' AND team_id IN (SELECT team_ids FROM users WHERE id = $%d)))`, i, i)
		args = append(args, filter.UserID)
		i++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, filter.Status)
	}
	query += " ORDER BY updated_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var apps []*sdk.AppRecord
	for rows.Next() {
		a, err := scanAppRow(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func scanApp(row *sql.Row) (*sdk.AppRecord, error) {
	var a sdk.AppRecord
	var teamID sql.NullString
	if err := row.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedBy, &a.Sharing, &teamID,
		&a.Status, &a.Tools, &a.Config, &a.URL, &a.Enabled, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.TeamID = teamID.String
	return &a, nil
}

func scanAppRow(rows *sql.Rows) (*sdk.AppRecord, error) {
	var a sdk.AppRecord
	var teamID sql.NullString
	if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedBy, &a.Sharing, &teamID,
		&a.Status, &a.Tools, &a.Config, &a.URL, &a.Enabled, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.TeamID = teamID.String
	return &a, nil
}

// --- ProviderStore ---

// CreateProvider persists a new encrypted provider record.
func (s *Store) CreateProvider(ctx context.Context, p *sdk.ProviderRecord) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO providers (id, type, name, config_encrypted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Type, p.Name, p.ConfigEncrypted, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// GetProvider retrieves a provider record by ID.
func (s *Store) GetProvider(ctx context.Context, id string) (*sdk.ProviderRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, type, name, config_encrypted, created_at, updated_at FROM providers WHERE id = $1`, id)
	var p sdk.ProviderRecord
	if err := row.Scan(&p.ID, &p.Type, &p.Name, &p.ConfigEncrypted, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateProvider replaces the mutable fields of an existing provider record.
func (s *Store) UpdateProvider(ctx context.Context, p *sdk.ProviderRecord) error {
	p.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE providers SET type=$1, name=$2, config_encrypted=$3, updated_at=$4 WHERE id=$5`,
		p.Type, p.Name, p.ConfigEncrypted, p.UpdatedAt, p.ID,
	)
	return err
}

// DeleteProvider removes a provider record by ID.
func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = $1`, id)
	return err
}

// ListProviders returns all provider records ordered by creation time.
func (s *Store) ListProviders(ctx context.Context) ([]*sdk.ProviderRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, name, config_encrypted, created_at, updated_at FROM providers ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*sdk.ProviderRecord
	for rows.Next() {
		var p sdk.ProviderRecord
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.ConfigEncrypted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// --- AgentStore ---

// CreateAgent persists a new agent record. ID is generated when blank.
func (s *Store) CreateAgent(ctx context.Context, agent *sdk.AgentRecord) error {
	if agent.ID == "" {
		agent.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	if agent.UpdatedAt.IsZero() {
		agent.UpdatedAt = now
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agents (id, name, description, role, system_prompt, provider, model, tools, sharing, team_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		agent.ID, agent.Name, agent.Description, agent.Role, agent.SystemPrompt,
		agent.Provider, agent.Model, agent.Tools, agent.Sharing, agent.TeamID,
		agent.CreatedBy, agent.CreatedAt, agent.UpdatedAt,
	)
	return err
}

// GetAgent retrieves an agent by ID.
func (s *Store) GetAgent(ctx context.Context, id string) (*sdk.AgentRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, role, system_prompt, provider, model, tools, sharing, team_id, created_by, created_at, updated_at
		 FROM agents WHERE id = $1`, id)
	return scanAgent(row)
}

// UpdateAgent replaces the mutable fields of an existing agent record.
func (s *Store) UpdateAgent(ctx context.Context, agent *sdk.AgentRecord) error {
	agent.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET name=$1, description=$2, role=$3, system_prompt=$4, provider=$5, model=$6, tools=$7, sharing=$8, team_id=$9, updated_at=$10
		 WHERE id=$11`,
		agent.Name, agent.Description, agent.Role, agent.SystemPrompt,
		agent.Provider, agent.Model, agent.Tools, agent.Sharing, agent.TeamID,
		agent.UpdatedAt, agent.ID,
	)
	return err
}

// DeleteAgent removes an agent record by ID.
func (s *Store) DeleteAgent(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE id = $1`, id)
	return err
}

// ListAgents returns agents visible to filter.UserID. Visibility:
//   - personal: only the creator
//   - team: any user whose team membership includes the agent's team_id
//   - org: everyone
//
// When UserID is empty the filter is skipped and every agent is returned
// (used by background jobs and admin tooling).
func (s *Store) ListAgents(ctx context.Context, filter sdk.AgentFilter) ([]*sdk.AgentRecord, error) {
	query := `SELECT id, name, description, role, system_prompt, provider, model, tools, sharing, team_id, created_by, created_at, updated_at FROM agents WHERE 1=1`
	args := []any{}
	if filter.UserID != "" {
		query += ` AND (created_by = $1 OR sharing = 'org' OR (sharing = 'team' AND team_id IN (SELECT team_ids FROM users WHERE id = $1)))`
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
	defer func() { _ = rows.Close() }()

	var agents []*sdk.AgentRecord
	for rows.Next() {
		a, err := scanAgentRow(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func scanAgent(row *sql.Row) (*sdk.AgentRecord, error) {
	var a sdk.AgentRecord
	var teamID sql.NullString
	if err := row.Scan(&a.ID, &a.Name, &a.Description, &a.Role, &a.SystemPrompt,
		&a.Provider, &a.Model, &a.Tools, &a.Sharing, &teamID,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.TeamID = teamID.String
	return &a, nil
}

func scanAgentRow(rows *sql.Rows) (*sdk.AgentRecord, error) {
	var a sdk.AgentRecord
	var teamID sql.NullString
	if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Role, &a.SystemPrompt,
		&a.Provider, &a.Model, &a.Tools, &a.Sharing, &teamID,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.TeamID = teamID.String
	return &a, nil
}

