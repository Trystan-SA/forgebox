// Conversation list / detail / rename / delete endpoints. The transcript
// for a session is returned inline by handleGetSession so the dashboard
// can hydrate a thread in a single round-trip.
package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// sessionWithTranscript is the response shape of GET /sessions/{id}.
type sessionWithTranscript struct {
	*sdk.SessionRecord
	Messages []sdk.Message `json:"messages"`
}

const maxSessionTitleLen = 200

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.ListSessions(r.Context(), sdk.SessionFilter{
		UserID: s.userID(r),
		Limit:  50,
	})
	if err != nil {
		slog.Error("list sessions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.UserID != s.userID(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	msgs, err := s.store.GetTranscript(r.Context(), id)
	if err != nil {
		slog.Error("get transcript", "session_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load transcript")
		return
	}
	writeJSON(w, http.StatusOK, sessionWithTranscript{
		SessionRecord: sess,
		Messages:      msgs,
	})
}

func (s *Server) handlePatchSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.UserID != s.userID(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	var req struct {
		Title *string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if len(title) > maxSessionTitleLen {
			title = title[:maxSessionTitleLen]
		}
		if title == "" {
			// Empty title falls back to the auto-derived title from the
			// first user prompt. If we can't recover one, leave the
			// existing title alone.
			msgs, terr := s.store.GetTranscript(r.Context(), id)
			if terr == nil {
				for _, msg := range msgs {
					if msg.Role == "user" && msg.Content != "" {
						title = truncateTitle(msg.Content)
						break
					}
				}
			}
		}
		if title != "" {
			if err := s.store.UpdateSessionTitle(r.Context(), id, title); err != nil {
				slog.Error("update session title", "session_id", id, "error", err)
				writeError(w, http.StatusInternalServerError, "failed to rename session")
				return
			}
			sess.Title = title
		}
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if sess.UserID != s.userID(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if err := s.store.DeleteSession(r.Context(), id); err != nil {
		slog.Error("delete session", "session_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// truncateTitle returns a 1-line, ≤80-char rendering of the prompt for
// use as the session's auto title. Newlines collapse to spaces; trailing
// whitespace is trimmed; if truncation occurs, an ellipsis is appended.
func truncateTitle(prompt string) string {
	p := strings.TrimSpace(strings.ReplaceAll(prompt, "\n", " "))
	p = strings.Join(strings.Fields(p), " ")
	const limit = 80
	if len(p) <= limit {
		return p
	}
	return strings.TrimSpace(p[:limit-1]) + "…"
}
