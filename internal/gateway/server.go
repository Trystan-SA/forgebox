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
	"strings"
	"sync"
	"time"

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
