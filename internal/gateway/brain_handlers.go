package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/forgebox/forgebox/pkg/sdk"
)

func (s *Server) handleGetBrain(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	if s.brainService == nil {
		writeError(w, http.StatusServiceUnavailable, "brain feature not configured")
		return
	}
	brain, err := s.brainService.GetOrCreateBrain(r.Context(), agentID)
	if err != nil {
		slog.Error("failed to get brain", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get brain")
		return
	}
	writeJSON(w, http.StatusOK, brain)
}

func (s *Server) handleListBrainFiles(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	brain, err := s.brainService.GetOrCreateBrain(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get brain")
		return
	}
	files, err := s.brainStore.ListFiles(r.Context(), brain.ID)
	if err != nil {
		slog.Error("failed to list brain files", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list files")
		return
	}
	if files == nil {
		files = []*sdk.BrainFile{}
	}
	writeJSON(w, http.StatusOK, files)
}

func (s *Server) handleCreateBrainFile(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	brain, err := s.brainService.GetOrCreateBrain(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get brain")
		return
	}

	file, err := s.brainService.CreateFile(r.Context(), brain.ID, req.Title, req.Content, getUserID(r))
	if err != nil {
		slog.Error("failed to create brain file", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create file")
		return
	}
	writeJSON(w, http.StatusCreated, file)
}

func (s *Server) handleGetBrainFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("fid")
	file, err := s.brainStore.GetFile(r.Context(), fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	writeJSON(w, http.StatusOK, file)
}

func (s *Server) handleUpdateBrainFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("fid")
	var req struct {
		Title   *string `json:"title,omitempty"`
		Content *string `json:"content,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := s.brainStore.GetFile(r.Context(), fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	title := existing.Title
	content := existing.Content
	if req.Title != nil {
		title = *req.Title
	}
	if req.Content != nil {
		content = *req.Content
	}

	file, err := s.brainService.UpdateFile(r.Context(), fileID, title, content)
	if err != nil {
		slog.Error("failed to update brain file", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update file")
		return
	}
	writeJSON(w, http.StatusOK, file)
}

func (s *Server) handleDeleteBrainFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("fid")
	if err := s.brainStore.DeleteFile(r.Context(), fileID); err != nil {
		slog.Error("failed to delete brain file", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleGetBrainGraph(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	brain, err := s.brainService.GetOrCreateBrain(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get brain")
		return
	}
	graph, err := s.brainStore.GetGraph(r.Context(), brain.ID)
	if err != nil {
		writeJSON(w, http.StatusOK, &sdk.BrainGraph{
			BrainID:  brain.ID,
			Clusters: []sdk.GraphCluster{},
			Nodes:    []sdk.GraphNode{},
			Links:    []sdk.BrainLink{},
		})
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) handleSearchBrainFiles(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	brain, err := s.brainService.GetOrCreateBrain(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get brain")
		return
	}

	results, err := s.brainService.Search(r.Context(), brain.ID, req.Query, req.Limit)
	if err != nil {
		slog.Error("brain search failed", "error", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	if results == nil {
		results = []*sdk.BrainFileWithMeta{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleListDreamProposals(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	brain, err := s.brainService.GetOrCreateBrain(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get brain")
		return
	}
	proposals, err := s.brainStore.ListDreamProposals(r.Context(), brain.ID)
	if err != nil {
		slog.Error("failed to list dream proposals", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list dreams")
		return
	}
	if proposals == nil {
		proposals = []*sdk.DreamProposal{}
	}
	writeJSON(w, http.StatusOK, proposals)
}

func (s *Server) handleGetDreamProposal(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("did")
	proposal, err := s.brainStore.GetDreamProposal(r.Context(), did)
	if err != nil {
		writeError(w, http.StatusNotFound, "dream proposal not found")
		return
	}
	writeJSON(w, http.StatusOK, proposal)
}

func (s *Server) handleApproveDream(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("did")
	proposal, err := s.brainStore.GetDreamProposal(r.Context(), did)
	if err != nil {
		writeError(w, http.StatusNotFound, "dream proposal not found")
		return
	}
	if proposal.Status != sdk.DreamPending {
		writeError(w, http.StatusConflict, "proposal already resolved")
		return
	}

	var changes []sdk.DreamChange
	if err := json.Unmarshal([]byte(proposal.Changes), &changes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse dream changes")
		return
	}

	for _, ch := range changes {
		switch ch.Action {
		case "create":
			if _, err := s.brainService.CreateFile(r.Context(), proposal.BrainID, ch.NewTitle, ch.NewContent, "dream"); err != nil {
				slog.Error("dream apply: create failed", "error", err)
			}
		case "edit":
			if _, err := s.brainService.UpdateFile(r.Context(), ch.FileID, ch.NewTitle, ch.NewContent); err != nil {
				slog.Error("dream apply: edit failed", "error", err)
			}
		case "delete":
			if err := s.brainStore.DeleteFile(r.Context(), ch.FileID); err != nil {
				slog.Error("dream apply: delete failed", "error", err)
			}
		}
	}

	if err := s.brainStore.UpdateDreamProposalStatus(r.Context(), did, sdk.DreamApproved, getUserID(r)); err != nil {
		slog.Error("failed to update dream status", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to approve dream")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleRejectDream(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("did")
	if err := s.brainStore.UpdateDreamProposalStatus(r.Context(), did, sdk.DreamRejected, getUserID(r)); err != nil {
		slog.Error("failed to reject dream", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to reject dream")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
