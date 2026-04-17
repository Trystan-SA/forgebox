// Package gateway implements the HTTP and gRPC API server.
//
// The gateway is the single entry point for all clients: web dashboard, CLI,
// chat integrations, and programmatic API access. It exposes a REST API for
// the web dashboard and a gRPC service for the CLI and SDK.
package gateway

import (
	"context"
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

	"crypto/rand"
	"encoding/hex"

	"github.com/forgebox/forgebox/internal/auth"
	"github.com/forgebox/forgebox/internal/engine"
	"github.com/forgebox/forgebox/internal/plugins"
	"github.com/forgebox/forgebox/internal/sessions"
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
}

// Server is the main ForgeBox API server.
type Server struct {
	cfg      Config
	mux      *http.ServeMux
	engine   *engine.Engine
	sessions *sessions.Manager
	registry *plugins.Registry
	store    sdk.StoragePlugin
}

// New creates a new gateway server.
func New(cfg Config) *Server {
	s := &Server{
		cfg:      cfg,
		mux:      http.NewServeMux(),
		engine:   cfg.Engine,
		sessions: cfg.Sessions,
		registry: cfg.Registry,
		store:    cfg.Store,
	}
	s.registerRoutes()
	return s
}

// Run starts the HTTP and gRPC servers. Blocks until ctx is cancelled.
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
		defer ln.Close()
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

	// Task endpoints.
	s.mux.HandleFunc("POST /api/v1/tasks", s.handleCreateTask)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}/stream", s.handleStreamTask)
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

	// Discovery endpoints.
	s.mux.HandleFunc("GET /api/v1/providers", s.handleListProviders)
	s.mux.HandleFunc("GET /api/v1/tools", s.handleListTools)
}

// --- Handlers ---

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	// Check VM orchestrator and storage health.
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ready")
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
		UserID:        getUserID(r),
	}

	// Run the task asynchronously.
	go func() {
		ctx := context.Background()
		result, err := s.engine.Run(ctx, task)
		if err != nil {
			slog.Error("task failed", "task_id", task.ID, "error", err)
			return
		}
		slog.Info("task completed", "task_id", task.ID, "tool_uses", result.ToolUses)
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

func (s *Server) handleStreamTask(w http.ResponseWriter, r *http.Request) {
	// SSE streaming endpoint.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// TODO: Subscribe to task events and stream them.
	fmt.Fprintf(w, "data: {\"type\": \"connected\"}\n\n")
	flusher.Flush()

	<-r.Context().Done()
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	// TODO: Cancel the running task via context cancellation.
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]string{
		"task_id": id,
		"status":  "cancelled",
	})
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.ListTasks(r.Context(), sdk.TaskFilter{
		UserID: getUserID(r),
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
		UserID: getUserID(r),
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
		UserID: getUserID(r),
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
		CreatedBy:   getUserID(r),
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

// --- Automation handlers ---

func (s *Server) handleListAutomations(w http.ResponseWriter, r *http.Request) {
	automations, err := s.store.ListAutomations(r.Context(), sdk.AutomationFilter{
		UserID: getUserID(r),
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
		CreatedBy:   getUserID(r),
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

func getUserID(r *http.Request) string {
	// TODO: Extract from auth middleware (JWT, API key, etc.).
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return "authenticated-user"
	}
	return "anonymous"
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
