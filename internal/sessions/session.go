// Package sessions manages conversation sessions and transcripts.
package sessions

import (
	"context"
	"sync"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Manager handles session lifecycle and in-memory caching.
type Manager struct {
	store    sdk.SessionStore
	mu       sync.RWMutex
	sessions map[string]*sdk.SessionRecord
}

// NewManager creates a session manager backed by the given store.
func NewManager(store sdk.SessionStore) *Manager {
	return &Manager{
		store:    store,
		sessions: make(map[string]*sdk.SessionRecord),
	}
}

// Create starts a new session with a server-generated ID.
func (m *Manager) Create(ctx context.Context, userID, provider, model string) (*sdk.SessionRecord, error) {
	return m.CreateWithID(ctx, uuid.New().String(), userID, provider, model, "", "dashboard")
}

// CreateWithID starts a new session using the caller-supplied ID. Used by
// the dashboard's first-turn flow so the client can stamp the session id
// before the round-trip and avoid a duplicate-session race.
func (m *Manager) CreateWithID(ctx context.Context, id, userID, provider, model, title, source string) (*sdk.SessionRecord, error) {
	if source == "" {
		source = "dashboard"
	}
	now := time.Now().UTC()
	session := &sdk.SessionRecord{
		ID:            id,
		UserID:        userID,
		Provider:      provider,
		Model:         model,
		Title:         title,
		Source:        source,
		CreatedAt:     now,
		UpdatedAt:     now,
		LastMessageAt: now,
	}

	if err := m.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	return session, nil
}

// Get returns a session by ID.
func (m *Manager) Get(ctx context.Context, id string) (*sdk.SessionRecord, error) {
	m.mu.RLock()
	if s, ok := m.sessions[id]; ok {
		m.mu.RUnlock()
		return s, nil
	}
	m.mu.RUnlock()

	return m.store.GetSession(ctx, id)
}

// AddMessage appends a message to a session's transcript and bumps
// last_message_at on the cached record.
func (m *Manager) AddMessage(ctx context.Context, sessionID string, msg *sdk.Message) error {
	if err := m.store.AppendMessage(ctx, sessionID, msg); err != nil {
		return err
	}

	now := time.Now().UTC()
	m.mu.Lock()
	if s, ok := m.sessions[sessionID]; ok {
		s.UpdatedAt = now
		s.LastMessageAt = now
	}
	m.mu.Unlock()

	return nil
}

// Touch rewrites provider/model/last_message_at on the session row to
// reflect the latest turn. Called after each task completes.
func (m *Manager) Touch(ctx context.Context, sessionID, provider, model string) error {
	sess, err := m.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	sess.Provider = provider
	sess.Model = model
	sess.UpdatedAt = now
	sess.LastMessageAt = now
	if err := m.store.UpdateSession(ctx, sess); err != nil {
		return err
	}
	m.mu.Lock()
	m.sessions[sessionID] = sess
	m.mu.Unlock()
	return nil
}

// UpdateTitle rewrites the title for a session.
func (m *Manager) UpdateTitle(ctx context.Context, sessionID, title string) error {
	if err := m.store.UpdateSessionTitle(ctx, sessionID, title); err != nil {
		return err
	}
	m.mu.Lock()
	if s, ok := m.sessions[sessionID]; ok {
		s.Title = title
		s.UpdatedAt = time.Now().UTC()
	}
	m.mu.Unlock()
	return nil
}

// Delete removes a session and evicts the cache entry.
func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	if err := m.store.DeleteSession(ctx, sessionID); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	return nil
}

// GetTranscript retrieves the full message history for a session.
func (m *Manager) GetTranscript(ctx context.Context, sessionID string) ([]sdk.Message, error) {
	return m.store.GetTranscript(ctx, sessionID)
}

// List returns sessions for a user.
func (m *Manager) List(ctx context.Context, userID string, limit int) ([]*sdk.SessionRecord, error) {
	return m.store.ListSessions(ctx, sdk.SessionFilter{
		UserID: userID,
		Limit:  limit,
	})
}
