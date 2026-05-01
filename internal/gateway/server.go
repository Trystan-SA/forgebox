// Package gateway implements the HTTP and gRPC API server.
//
// The gateway is the single entry point for all clients: web dashboard, CLI,
// chat integrations, and programmatic API access. It exposes a REST API for
// the web dashboard and a gRPC service for the CLI and SDK.
package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/forgebox/forgebox/internal/auth"
	"github.com/forgebox/forgebox/internal/brain"
	fbcrypto "github.com/forgebox/forgebox/internal/crypto"
	"github.com/forgebox/forgebox/internal/engine"
	"github.com/forgebox/forgebox/internal/events"
	"github.com/forgebox/forgebox/internal/plugins"
	"github.com/forgebox/forgebox/internal/sessions"
	"github.com/forgebox/forgebox/internal/tasktoken"
	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Config holds the gateway server dependencies.
type Config struct {
	ListenAddr     string
	GRPCListenAddr string
	Engine         *engine.Engine
	Sessions       *sessions.Manager
	Registry       *plugins.Registry
	Store          sdk.StoragePlugin
	BrainService   *brain.Service
	BrainStore     sdk.BrainStore
	SecretBox      *fbcrypto.SecretBox
	// Events is the in-process event bus that producers publish to and the
	// WebSocket Hub fans out to connected clients. Required.
	Events *events.Bus
	// TaskTokens stores short-lived `fbtask_…` tokens issued by the engine
	// for in-VM tool callbacks. Resolved by userID() to map back to the
	// originating user. Optional; when nil, fbtask_-prefixed bearer tokens
	// resolve as invalid (empty user id → 401-eligible).
	TaskTokens *tasktoken.Store
}

// Server is the main ForgeBox API server.
type Server struct {
	cfg          Config
	mux          *http.ServeMux
	engine       *engine.Engine
	sessions     *sessions.Manager
	registry     *plugins.Registry
	store        sdk.StoragePlugin
	brainService *brain.Service
	brainStore   sdk.BrainStore
	secretBox    *fbcrypto.SecretBox
	hub          *Hub
	taskTokens   *tasktoken.Store
}

// New creates a new gateway server. The event bus must be supplied; the
// server creates its own Hub bound to that bus.
func New(cfg Config) *Server {
	if cfg.Events == nil {
		cfg.Events = events.New(0)
	}
	s := &Server{
		cfg:          cfg,
		mux:          http.NewServeMux(),
		engine:       cfg.Engine,
		sessions:     cfg.Sessions,
		registry:     cfg.Registry,
		store:        cfg.Store,
		brainService: cfg.BrainService,
		brainStore:   cfg.BrainStore,
		secretBox:    cfg.SecretBox,
		hub:          NewHub(cfg.Events, 0),
		taskTokens:   cfg.TaskTokens,
	}
	s.registerRoutes()
	return s
}

// Run starts the HTTP and gRPC servers. Blocks until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:         s.cfg.ListenAddr,
		Handler:      s.middleware(s.mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// WebSocket Hub event-dispatch loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.hub.Run(ctx)
	}()

	// HTTP server.
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("HTTP server listening", "addr", s.cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	// gRPC server (placeholder — implement with google.golang.org/grpc).
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("gRPC server listening", "addr", s.cfg.GRPCListenAddr)
		ln, err := net.Listen("tcp", s.cfg.GRPCListenAddr)
		if err != nil {
			errCh <- fmt.Errorf("grpc listen: %w", err)
			return
		}
		defer func() { _ = ln.Close() }()
		// TODO: Register gRPC services and serve.
		<-ctx.Done()
	}()

	// Wait for shutdown signal.
	select {
	case <-ctx.Done():
		slog.Info("shutting down gateway")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
	}

	wg.Wait()
	return nil
}

func (s *Server) registerRoutes() {
	// Health endpoints.
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)

	// WebSocket endpoint for real-time events (replaces per-task SSE).
	s.mux.HandleFunc("GET /api/v1/ws", s.handleWS)

	// Task endpoints.
	s.mux.HandleFunc("POST /api/v1/tasks", s.handleCreateTask)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("DELETE /api/v1/tasks/{id}", s.handleCancelTask)
	s.mux.HandleFunc("GET /api/v1/tasks", s.handleListTasks)

	// Session endpoints.
	s.mux.HandleFunc("GET /api/v1/sessions", s.handleListSessions)
	s.mux.HandleFunc("GET /api/v1/sessions/{id}", s.handleGetSession)
	s.mux.HandleFunc("POST /api/v1/sessions/{id}/message", s.handleSendMessage)

	// Setup endpoint (first-install bootstrap).
	s.mux.HandleFunc("GET /api/v1/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("POST /api/v1/setup", s.handleSetup)

	// Auth endpoints.
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

	// Automation endpoints.
	s.mux.HandleFunc("GET /api/v1/automations", s.handleListAutomations)
	s.mux.HandleFunc("POST /api/v1/automations", s.handleCreateAutomation)
	s.mux.HandleFunc("GET /api/v1/automations/{id}", s.handleGetAutomation)
	s.mux.HandleFunc("GET /api/v1/automations/{id}/yaml", s.handleGetAutomationYAML)
	s.mux.HandleFunc("PUT /api/v1/automations/{id}", s.handleUpdateAutomation)
	s.mux.HandleFunc("DELETE /api/v1/automations/{id}", s.handleDeleteAutomation)

	// App endpoints.
	s.mux.HandleFunc("GET /api/v1/apps", s.handleListApps)
	s.mux.HandleFunc("POST /api/v1/apps", s.handleCreateApp)
	s.mux.HandleFunc("GET /api/v1/apps/{id}", s.handleGetApp)
	s.mux.HandleFunc("PUT /api/v1/apps/{id}", s.handleUpdateApp)
	s.mux.HandleFunc("DELETE /api/v1/apps/{id}", s.handleDeleteApp)

	// Provider endpoints.
	s.mux.HandleFunc("GET /api/v1/providers", s.handleListProviders)
	s.mux.HandleFunc("POST /api/v1/providers", s.handleCreateProvider)
	s.mux.HandleFunc("DELETE /api/v1/providers/{id}", s.handleDeleteProvider)
	s.mux.HandleFunc("GET /api/v1/tools", s.handleListTools)

	// Agent endpoints.
	s.mux.HandleFunc("GET /api/v1/agents", s.handleListAgents)
	s.mux.HandleFunc("POST /api/v1/agents", s.handleCreateAgent)
	s.mux.HandleFunc("GET /api/v1/agents/{id}", s.handleGetAgent)
	s.mux.HandleFunc("PUT /api/v1/agents/{id}", s.handleUpdateAgent)
	s.mux.HandleFunc("DELETE /api/v1/agents/{id}", s.handleDeleteAgent)

	// Brain endpoints.
	s.mux.HandleFunc("GET /api/v1/agents/{id}/brain", s.handleGetBrain)
	s.mux.HandleFunc("GET /api/v1/agents/{id}/brain/files", s.handleListBrainFiles)
	s.mux.HandleFunc("POST /api/v1/agents/{id}/brain/files", s.handleCreateBrainFile)
	s.mux.HandleFunc("GET /api/v1/agents/{id}/brain/files/{fid}", s.handleGetBrainFile)
	s.mux.HandleFunc("PUT /api/v1/agents/{id}/brain/files/{fid}", s.handleUpdateBrainFile)
	s.mux.HandleFunc("DELETE /api/v1/agents/{id}/brain/files/{fid}", s.handleDeleteBrainFile)
	s.mux.HandleFunc("GET /api/v1/agents/{id}/brain/graph", s.handleGetBrainGraph)
	s.mux.HandleFunc("POST /api/v1/agents/{id}/brain/search", s.handleSearchBrainFiles)
	s.mux.HandleFunc("GET /api/v1/agents/{id}/brain/dreams", s.handleListDreamProposals)
	s.mux.HandleFunc("GET /api/v1/agents/{id}/brain/dreams/{did}", s.handleGetDreamProposal)
	s.mux.HandleFunc("POST /api/v1/agents/{id}/brain/dreams/{did}/approve", s.handleApproveDream)
	s.mux.HandleFunc("POST /api/v1/agents/{id}/brain/dreams/{did}/reject", s.handleRejectDream)
}

// --- Handlers ---

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "ok")
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	// Check VM orchestrator and storage health.
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "ready")
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt        string `json:"prompt"`
		Provider      string `json:"provider,omitempty"`
		Model         string `json:"model,omitempty"`
		Timeout       string `json:"timeout,omitempty"`
		MemoryMB      int    `json:"memory_mb,omitempty"`
		VCPUs         int    `json:"vcpus,omitempty"`
		NetworkAccess bool   `json:"network_access,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	timeout, _ := time.ParseDuration(req.Timeout)

	task := &engine.Task{
		ID:            uuid.New().String(),
		Prompt:        req.Prompt,
		Provider:      req.Provider,
		Model:         req.Model,
		MemoryMB:      req.MemoryMB,
		VCPUs:         req.VCPUs,
		Timeout:       timeout,
		NetworkAccess: req.NetworkAccess,
		UserID:        s.userID(r),
	}

	// Persist the task before kicking off the runner so GET /tasks/{id}
	// can answer the dashboard's poll while the engine is still working.
	// Without this row, the poll loops on 404 until it times out.
	now := time.Now().UTC()
	record := &sdk.TaskRecord{
		ID:        task.ID,
		Status:    sdk.TaskRunning,
		Prompt:    task.Prompt,
		Provider:  task.Provider,
		Model:     task.Model,
		UserID:    task.UserID,
		CreatedAt: now,
		StartedAt: &now,
	}
	if err := s.store.CreateTask(r.Context(), record); err != nil {
		slog.Error("create task record", "task_id", task.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	// Run the task asynchronously and persist the terminal status so the
	// dashboard's poll converges on completed/failed.
	go func() {
		ctx := context.Background()
		result, err := s.engine.Run(ctx, task)
		completed := time.Now().UTC()
		record.CompletedAt = &completed
		if err != nil {
			slog.Error("task failed", "task_id", task.ID, "error", err)
			record.Status = sdk.TaskFailed
			record.Error = friendlyTaskError(err)
		} else {
			slog.Info("task completed", "task_id", task.ID, "tool_uses", result.ToolUses)
			record.Status = sdk.TaskCompleted
			record.Result = result.Output
			record.TokensIn = result.Cost.InputTokens
			record.TokensOut = result.Cost.OutputTokens
			record.Cost = result.Cost.TotalCost
		}
		if uerr := s.store.UpdateTask(ctx, record); uerr != nil {
			slog.Error("update task record", "task_id", task.ID, "error", uerr)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"task_id": task.ID,
		"status":  "running",
	})
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	// TODO: Cancel the running task via context cancellation.
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]string{
		"task_id": id,
		"status":  "canceled",
	})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.ListTasks(r.Context(), sdk.TaskFilter{
		UserID: s.userID(r),
		Limit:  50,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.ListSessions(r.Context(), sdk.SessionFilter{
		UserID: s.userID(r),
		Limit:  50,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	session, err := s.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// TODO: Send message to existing session.
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := s.registry.ListProviders()
	writeJSON(w, http.StatusOK, providers)
}

type createProviderRequest struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

func (s *Server) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	if s.secretBox == nil {
		writeError(w, http.StatusServiceUnavailable, "encryption key not configured")
		return
	}
	var req createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}

	// The display name is derived from the type — operators do not get to
	// customize it (see specs/3.1.2). Reject unknown types early so the user
	// gets a clear error before we touch the registry or DB.
	label, ok := plugins.LabelForType(req.Type)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown provider type %q", req.Type))
		return
	}

	// Conflict if a provider of this type is already configured. We only
	// permit one DB-backed provider per type now that names are derived.
	rows, err := s.store.ListProviders(r.Context())
	if err != nil {
		slog.Error("list providers for dedup", "error", err)
		writeError(w, http.StatusInternalServerError, "load providers")
		return
	}
	for _, row := range rows {
		if row.Type == req.Type {
			writeError(w, http.StatusConflict, fmt.Sprintf("provider type %q is already configured", req.Type))
			return
		}
	}
	// Also conflict with a registry-name collision (e.g. a built-in keyed
	// under the same label as the derived one).
	if _, regErr := s.registry.GetProvider(label); regErr == nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("provider %q already in use", label))
		return
	}

	// Validate by instantiating + Init-ing a throwaway plugin. This catches bad
	// types and bad config (wrong key prefix, missing fields, etc.) before we
	// persist anything.
	probe, err := loadProviderProbe(req.Type)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err = probe.Init(r.Context(), req.Config); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid provider config: %v", err))
		return
	}
	_ = probe.Shutdown(r.Context())

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshal config")
		return
	}
	sealed, err := s.secretBox.Encrypt(configJSON)
	if err != nil {
		slog.Error("encrypt provider config", "error", err)
		writeError(w, http.StatusInternalServerError, "encrypt config")
		return
	}

	rec := &sdk.ProviderRecord{
		Type:            req.Type,
		Name:            label,
		ConfigEncrypted: sealed,
	}
	if err := s.store.CreateProvider(r.Context(), rec); err != nil {
		slog.Error("create provider row", "error", err)
		writeError(w, http.StatusInternalServerError, "save provider")
		return
	}
	if err := s.registry.AddStoredProvider(r.Context(), rec, s.secretBox); err != nil {
		// Roll back the row so the next attempt with the same name works.
		_ = s.store.DeleteProvider(r.Context(), rec.ID)
		slog.Error("register provider", "error", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("register provider: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, sdk.PluginMeta{
		ID:           rec.ID,
		Name:         rec.Name,
		Type:         sdk.PluginTypeProvider,
		ProviderType: rec.Type,
	})
}

func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	rec, err := s.store.GetProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}
	if err := s.store.DeleteProvider(r.Context(), id); err != nil {
		slog.Error("delete provider row", "error", err)
		writeError(w, http.StatusInternalServerError, "delete provider")
		return
	}
	s.registry.UnregisterProviderByID(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"id": rec.ID, "status": "deleted"})
}

// loadProviderProbe constructs an unconfigured plugin matching the registry
// factory so the handler can validate config without exposing registry internals.
func loadProviderProbe(name string) (sdk.ProviderPlugin, error) {
	return plugins.NewProvider(name)
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.registry.ListTools()
	schemas := make([]sdk.ToolSchema, len(tools))
	for i, t := range tools {
		schemas[i] = t.Schema()
	}
	writeJSON(w, http.StatusOK, schemas)
}

// --- App handlers ---

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListApps(r.Context(), sdk.AppFilter{
		UserID: s.userID(r),
		Limit:  100,
	})
	if err != nil {
		slog.Error("failed to list apps", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list apps")
		return
	}
	if apps == nil {
		apps = []*sdk.AppRecord{}
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Sharing     string `json:"sharing"`
		TeamID      string `json:"team_id,omitempty"`
		Tools       string `json:"tools"`
		Config      string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Sharing == "" {
		req.Sharing = "personal"
	}
	if req.Tools == "" {
		req.Tools = "[]"
	}
	if req.Config == "" {
		req.Config = "{}"
	}

	now := time.Now()
	app := &sdk.AppRecord{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   s.userID(r),
		Sharing:     req.Sharing,
		TeamID:      req.TeamID,
		Status:      sdk.AppDraft,
		Tools:       req.Tools,
		Config:      req.Config,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateApp(r.Context(), app); err != nil {
		slog.Error("failed to create app", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create app")
		return
	}

	writeJSON(w, http.StatusCreated, app)
}

func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	app, err := s.store.GetApp(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := s.store.GetApp(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}

	var req struct {
		Name        *string        `json:"name,omitempty"`
		Description *string        `json:"description,omitempty"`
		Sharing     *string        `json:"sharing,omitempty"`
		TeamID      *string        `json:"team_id,omitempty"`
		Status      *sdk.AppStatus `json:"status,omitempty"`
		Tools       *string        `json:"tools,omitempty"`
		Config      *string        `json:"config,omitempty"`
		URL         *string        `json:"url,omitempty"`
		Enabled     *bool          `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Sharing != nil {
		existing.Sharing = *req.Sharing
	}
	if req.TeamID != nil {
		existing.TeamID = *req.TeamID
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.Tools != nil {
		existing.Tools = *req.Tools
	}
	if req.Config != nil {
		existing.Config = *req.Config
	}
	if req.URL != nil {
		existing.URL = *req.URL
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.UpdatedAt = time.Now()

	if err := s.store.UpdateApp(r.Context(), existing); err != nil {
		slog.Error("failed to update app", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update app")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteApp(r.Context(), id); err != nil {
		slog.Error("failed to delete app", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete app")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Agent handlers ---
//
// See specs/1.0.0-agents.md. The handlers persist agent records and enforce
// the validation rules in 1.2.3 (sharing values, provider / model existence,
// tool name allow-list).

var validAgentSharing = map[string]bool{"personal": true, "team": true, "org": true}

// validateAgentRefs checks that the provider, model, and tool names on an
// incoming agent record correspond to objects the gateway actually knows
// about. provider/model are optional at create-time per spec 1.2.3; tools
// must always be a JSON array of registered tool names if non-empty.
func (s *Server) validateAgentRefs(provider, model, toolsJSON string) error {
	if provider != "" {
		p, err := s.registry.GetProvider(provider)
		if err != nil {
			return fmt.Errorf("provider %q is not configured", provider)
		}
		if model != "" {
			found := false
			for _, m := range p.Models() {
				if m.ID == model {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("model %q is not available on provider %q", model, provider)
			}
		}
	}
	if toolsJSON != "" && toolsJSON != "[]" {
		var names []string
		if err := json.Unmarshal([]byte(toolsJSON), &names); err != nil {
			return fmt.Errorf("tools must be a JSON array of strings")
		}
		known := map[string]bool{}
		for _, t := range s.registry.ListTools() {
			known[t.Name()] = true
		}
		for _, n := range names {
			if !known[n] {
				return fmt.Errorf("tool %q is not registered", n)
			}
		}
	}
	return nil
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	uid := s.userID(r)
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	agents, err := s.store.ListAgents(r.Context(), sdk.AgentFilter{
		UserID: uid,
		Limit:  100,
	})
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	if agents == nil {
		agents = []*sdk.AgentRecord{}
	}
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	uid := s.userID(r)
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		SystemPrompt string `json:"system_prompt"`
		Provider     string `json:"provider"`
		Model        string `json:"model"`
		Tools        string `json:"tools"`
		Sharing      string `json:"sharing"`
		TeamID       string `json:"team_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Sharing == "" {
		req.Sharing = "personal"
	}
	if !validAgentSharing[req.Sharing] {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid sharing %q", req.Sharing))
		return
	}
	if req.Sharing == "team" && req.TeamID == "" {
		writeError(w, http.StatusBadRequest, "team_id is required when sharing is team")
		return
	}
	if req.Tools == "" {
		req.Tools = "[]"
	}
	if err := s.validateAgentRefs(req.Provider, req.Model, req.Tools); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	agent := &sdk.AgentRecord{
		ID:           uuid.New().String(),
		Name:         strings.TrimSpace(req.Name),
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Provider:     req.Provider,
		Model:        req.Model,
		Tools:        req.Tools,
		Sharing:      req.Sharing,
		TeamID:       req.TeamID,
		CreatedBy:    uid,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateAgent(r.Context(), agent); err != nil {
		slog.Error("failed to create agent", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	if uid := s.userID(r); uid == "" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	id := r.PathValue("id")
	agent, err := s.store.GetAgent(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	if uid := s.userID(r); uid == "" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	id := r.PathValue("id")
	existing, err := s.store.GetAgent(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	var req struct {
		Name         *string `json:"name,omitempty"`
		Description  *string `json:"description,omitempty"`
		SystemPrompt *string `json:"system_prompt,omitempty"`
		Provider     *string `json:"provider,omitempty"`
		Model        *string `json:"model,omitempty"`
		Tools        *string `json:"tools,omitempty"`
		Sharing      *string `json:"sharing,omitempty"`
		TeamID       *string `json:"team_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			writeError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
		existing.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.SystemPrompt != nil {
		existing.SystemPrompt = *req.SystemPrompt
	}
	if req.Provider != nil {
		existing.Provider = *req.Provider
	}
	if req.Model != nil {
		existing.Model = *req.Model
	}
	if req.Tools != nil {
		existing.Tools = *req.Tools
	}
	if req.Sharing != nil {
		if !validAgentSharing[*req.Sharing] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid sharing %q", *req.Sharing))
			return
		}
		existing.Sharing = *req.Sharing
	}
	if req.TeamID != nil {
		existing.TeamID = *req.TeamID
	}
	if existing.Sharing == "team" && existing.TeamID == "" {
		writeError(w, http.StatusBadRequest, "team_id is required when sharing is team")
		return
	}
	if err := s.validateAgentRefs(existing.Provider, existing.Model, existing.Tools); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.store.UpdateAgent(r.Context(), existing); err != nil {
		slog.Error("failed to update agent", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if uid := s.userID(r); uid == "" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	id := r.PathValue("id")
	if err := s.store.DeleteAgent(r.Context(), id); err != nil {
		slog.Error("failed to delete agent", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete agent")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Automation handlers ---

func (s *Server) handleListAutomations(w http.ResponseWriter, r *http.Request) {
	automations, err := s.store.ListAutomations(r.Context(), sdk.AutomationFilter{
		UserID: s.userID(r),
		Limit:  100,
	})
	if err != nil {
		slog.Error("failed to list automations", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list automations")
		return
	}
	if automations == nil {
		automations = []*sdk.AutomationRecord{}
	}
	writeJSON(w, http.StatusOK, automations)
}

func (s *Server) handleCreateAutomation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Sharing     string `json:"sharing"`
		TeamID      string `json:"team_id,omitempty"`
		Trigger     string `json:"trigger"`
		Nodes       string `json:"nodes"`
		Edges       string `json:"edges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Sharing == "" {
		req.Sharing = "personal"
	}
	if req.Trigger == "" {
		req.Trigger = "{}"
	}
	if req.Nodes == "" {
		req.Nodes = "[]"
	}
	if req.Edges == "" {
		req.Edges = "[]"
	}

	now := time.Now()
	automation := &sdk.AutomationRecord{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   s.userID(r),
		Sharing:     req.Sharing,
		TeamID:      req.TeamID,
		Trigger:     req.Trigger,
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateAutomation(r.Context(), automation); err != nil {
		slog.Error("failed to create automation", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create automation")
		return
	}

	writeJSON(w, http.StatusCreated, automation)
}

func (s *Server) handleGetAutomation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	automation, err := s.store.GetAutomation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "automation not found")
		return
	}
	writeJSON(w, http.StatusOK, automation)
}

func (s *Server) handleGetAutomationYAML(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	automation, err := s.store.GetAutomation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "automation not found")
		return
	}

	out, err := automationToYAML(automation)
	if err != nil {
		slog.Error("failed to serialize automation as yaml", "error", err, "automation_id", id)
		writeError(w, http.StatusInternalServerError, "failed to serialize automation")
		return
	}

	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func (s *Server) handleUpdateAutomation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := s.store.GetAutomation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "automation not found")
		return
	}

	var req struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		Sharing     *string `json:"sharing,omitempty"`
		TeamID      *string `json:"team_id,omitempty"`
		Trigger     *string `json:"trigger,omitempty"`
		Nodes       *string `json:"nodes,omitempty"`
		Edges       *string `json:"edges,omitempty"`
		Enabled     *bool   `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Sharing != nil {
		existing.Sharing = *req.Sharing
	}
	if req.TeamID != nil {
		existing.TeamID = *req.TeamID
	}
	if req.Trigger != nil {
		existing.Trigger = *req.Trigger
	}
	if req.Nodes != nil {
		existing.Nodes = *req.Nodes
	}
	if req.Edges != nil {
		existing.Edges = *req.Edges
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.UpdatedAt = time.Now()

	if err := s.store.UpdateAutomation(r.Context(), existing); err != nil {
		slog.Error("failed to update automation", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update automation")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteAutomation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteAutomation(r.Context(), id); err != nil {
		slog.Error("failed to delete automation", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete automation")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if user.Disabled {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token := generateToken()

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email)
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.CountUsers(r.Context())
	if err != nil {
		slog.Error("failed to count users", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to check setup status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{
		"setup_required": count == 0,
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	count, err := s.store.CountUsers(ctx)
	if err != nil {
		slog.Error("failed to count users", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to check user count")
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "setup already completed — users exist")
		return
	}

	setupPassword := os.Getenv("FORGEBOX_FIRST_PASSWORD")
	if setupPassword == "" {
		writeError(w, http.StatusServiceUnavailable, "FORGEBOX_FIRST_PASSWORD not set — cannot bootstrap")
		return
	}

	var req struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Password      string `json:"password"`
		SetupPassword string `json:"setup_password"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" || req.SetupPassword == "" {
		writeError(w, http.StatusBadRequest, "name, email, password, and setup_password are required")
		return
	}

	if req.SetupPassword != setupPassword {
		writeError(w, http.StatusForbidden, "invalid setup password")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	user := &sdk.UserRecord{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         "admin",
	}

	if err := s.store.CreateUser(ctx, user); err != nil {
		slog.Error("failed to create admin user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	slog.Info("first admin account created via /api/v1/setup", "user_id", user.ID, "email", user.Email)
	writeJSON(w, http.StatusCreated, map[string]string{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

// --- Middleware ---

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// CORS.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Request logging.
		next.ServeHTTP(w, r)
		slog.Debug("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}

// --- Helpers ---

// userID resolves the calling user from the Authorization header. Bearer
// tokens prefixed with `fbtask_` are looked up in TaskTokens (issued by the
// engine for in-VM tool callbacks) and resolve to the originating user.
// Other Bearer tokens fall back to the existing stub until proper auth
// lands.
func (s *Server) userID(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "anonymous"
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if strings.HasPrefix(token, tasktoken.Prefix) {
		if s.taskTokens != nil {
			if userID, _, ok := s.taskTokens.Resolve(token); ok {
				return userID
			}
		}
		return "" // token-shaped but invalid → 401-eligible
	}
	return "authenticated-user"
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// friendlyTaskError maps an engine.Run error chain to a short, user-facing
// message to persist as TaskRecord.Error. The full wrapped error is still
// logged via slog at the call site for operators. Falls back to err.Error()
// when no sentinel matches so unexpected failures stay debuggable in the UI.
func friendlyTaskError(err error) string {
	switch {
	case errors.Is(err, sdk.ErrRateLimit):
		return "Rate limit exceeded. The provider is throttling requests — wait a moment and try again, or switch to a different model."
	case errors.Is(err, sdk.ErrAuth):
		return "Provider authentication failed. The configured credentials are missing, expired, or unauthorized — reconfigure the provider in Settings."
	case errors.Is(err, sdk.ErrInputTooLarge):
		return "The prompt is too large for this model's context window. Shorten it or pick a model with a larger context."
	case errors.Is(err, sdk.ErrTransient):
		return "The provider is temporarily unavailable. Try again in a moment."
	default:
		return err.Error()
	}
}
