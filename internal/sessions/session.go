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

// Create starts a new session.
func (m *Manager) Create(ctx context.Context, userID, provider, model string) (*sdk.SessionRecord, error) {
	session := &sdk.SessionRecord{
		ID:        uuid.New().String(),
		UserID:    userID,
		Provider:  provider,
		Model:     model,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

// AddMessage appends a message to a session's transcript.
func (m *Manager) AddMessage(ctx context.Context, sessionID string, msg *sdk.Message) error {
	if err := m.store.AppendMessage(ctx, sessionID, msg); err != nil {
		return err
	}

	m.mu.Lock()
	if s, ok := m.sessions[sessionID]; ok {
		s.UpdatedAt = time.Now()
	}
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
