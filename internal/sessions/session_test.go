package sessions

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*sdk.SessionRecord
	messages map[string][]*sdk.Message
	getCalls int
}

func newMockStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*sdk.SessionRecord),
		messages: make(map[string][]*sdk.Message),
	}
}

func (m *mockSessionStore) CreateSession(_ context.Context, s *sdk.SessionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionStore) GetSession(_ context.Context, id string) (*sdk.SessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session %q not found", id)
}

func (m *mockSessionStore) UpdateSession(_ context.Context, s *sdk.SessionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionStore) ListSessions(_ context.Context, filter sdk.SessionFilter) ([]*sdk.SessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*sdk.SessionRecord
	for _, s := range m.sessions {
		if filter.UserID != "" && s.UserID != filter.UserID {
			continue
		}
		out = append(out, s)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (m *mockSessionStore) AppendMessage(_ context.Context, sessionID string, msg *sdk.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

func (m *mockSessionStore) GetTranscript(_ context.Context, sessionID string) ([]sdk.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []sdk.Message
	for _, msg := range m.messages[sessionID] {
		out = append(out, *msg)
	}
	return out, nil
}

func TestCreate_SetsRequiredFields(t *testing.T) {
	mgr := NewManager(newMockStore())
	s, err := mgr.Create(context.Background(), "user-1", "anthropic", "claude-sonnet-4-6")
	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, "user-1", s.UserID)
	assert.Equal(t, "anthropic", s.Provider)
	assert.Equal(t, "claude-sonnet-4-6", s.Model)
	assert.NotZero(t, s.CreatedAt)
	assert.NotZero(t, s.UpdatedAt)
}

func TestCreate_GeneratesUniqueIDs(t *testing.T) {
	mgr := NewManager(newMockStore())
	s1, err := mgr.Create(context.Background(), "user-1", "anthropic", "claude-sonnet-4-6")
	require.NoError(t, err)
	s2, err := mgr.Create(context.Background(), "user-1", "anthropic", "claude-sonnet-4-6")
	require.NoError(t, err)
	assert.NotEqual(t, s1.ID, s2.ID)
}

func TestGet_ReturnsCachedSessionWithoutHittingStore(t *testing.T) {
	store := newMockStore()
	mgr := NewManager(store)
	s, err := mgr.Create(context.Background(), "user-1", "openai", "gpt-4")
	require.NoError(t, err)

	got, err := mgr.Get(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.ID)
	assert.Equal(t, 0, store.getCalls, "cache hit must not call store.GetSession")
}

func TestGet_FallsBackToStoreOnCacheMiss(t *testing.T) {
	store := newMockStore()
	// Bypass the manager and write directly to the store.
	session := &sdk.SessionRecord{ID: "session-42", UserID: "user-1"}
	store.sessions["session-42"] = session

	mgr := NewManager(store)
	got, err := mgr.Get(context.Background(), "session-42")
	require.NoError(t, err)
	assert.Equal(t, "session-42", got.ID)
	assert.Equal(t, 1, store.getCalls)
}

func TestGet_ReturnsErrorForMissingSession(t *testing.T) {
	mgr := NewManager(newMockStore())
	_, err := mgr.Get(context.Background(), "does-not-exist")
	require.Error(t, err)
}

func TestAddMessage_StoresInBackend(t *testing.T) {
	store := newMockStore()
	mgr := NewManager(store)
	s, err := mgr.Create(context.Background(), "user-1", "openai", "gpt-4")
	require.NoError(t, err)

	err = mgr.AddMessage(context.Background(), s.ID, &sdk.Message{Role: "user", Content: "hello"})
	require.NoError(t, err)
	assert.Len(t, store.messages[s.ID], 1)
}

func TestGetTranscript_ReturnsAllMessages(t *testing.T) {
	mgr := NewManager(newMockStore())
	s, err := mgr.Create(context.Background(), "user-1", "openai", "gpt-4")
	require.NoError(t, err)

	require.NoError(t, mgr.AddMessage(context.Background(), s.ID, &sdk.Message{Role: "user", Content: "hello"}))
	require.NoError(t, mgr.AddMessage(context.Background(), s.ID, &sdk.Message{Role: "assistant", Content: "world"}))

	msgs, err := mgr.GetTranscript(context.Background(), s.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "assistant", msgs[1].Role)
}

func TestList_FiltersByUserID(t *testing.T) {
	mgr := NewManager(newMockStore())
	_, err := mgr.Create(context.Background(), "user-1", "openai", "gpt-4")
	require.NoError(t, err)
	_, err = mgr.Create(context.Background(), "user-2", "openai", "gpt-4")
	require.NoError(t, err)

	sessions, err := mgr.List(context.Background(), "user-1", 10)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "user-1", sessions[0].UserID)
}
