// Package main is the entry point for the ForgeBox server and CLI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/forgebox/forgebox/internal/brain"
	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/internal/crypto"
	"github.com/forgebox/forgebox/internal/engine"
	"github.com/forgebox/forgebox/internal/events"
	"github.com/forgebox/forgebox/internal/gateway"
	"github.com/forgebox/forgebox/internal/permissions"
	"github.com/forgebox/forgebox/internal/plugins"
	"github.com/forgebox/forgebox/internal/sessions"
	"github.com/forgebox/forgebox/internal/storage/postgres"
	"github.com/forgebox/forgebox/internal/telemetry"
	"github.com/forgebox/forgebox/internal/vm"
	"github.com/forgebox/forgebox/pkg/sdk"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		if err := cmdServe(); err != nil {
			os.Exit(1) // cmdServe already logged the error
		}
	case "run":
		if err := cmdRun(); err != nil {
			os.Exit(1) // cmdRun already logged the error
		}
	case "init":
		cmdInit()
	case "status":
		cmdStatus()
	case "version":
		fmt.Printf("forgebox %s (commit: %s, built: %s)\n", version, commit, buildDate)
	case "help", "-h", "--help":
		printUsage()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdServe() error {
	cfg, err := config.Load(configPath())
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Storage.DSN == "" {
		slog.Error("storage DSN missing — set FORGEBOX_DATABASE_URL or storage.dsn in config")
		os.Exit(1)
	}

	// All os.Exit-on-fail initializations must complete before any defer so that
	// deferred cleanups are not skipped by os.Exit.
	store, err := postgres.New(cfg.Storage.DSN)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}

	registry := plugins.NewRegistry()
	if err = registry.LoadBuiltins(cfg); err != nil {
		slog.Error("failed to load plugins", "error", err)
		os.Exit(1)
	}

	secretBox, err := crypto.NewFromEnv()
	if err != nil {
		slog.Error("encryption key required for DB-backed providers", "error", err, "env", crypto.EnvKey)
		os.Exit(1)
	}

	orch, err := vm.NewOrchestrator(cfg.VM)
	if err != nil {
		slog.Error("failed to init VM orchestrator", "error", err)
		os.Exit(1)
	}

	if err = telemetry.Init(cfg.Telemetry); err != nil {
		slog.Warn("telemetry init failed, continuing without", "error", err)
	}
	defer telemetry.Shutdown()
	defer func() { _ = store.Close() }()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	defer orch.Shutdown(ctx)

	// Check if this is a fresh install with no users.
	userCount, err := store.CountUsers(ctx)
	if err != nil {
		slog.Warn("failed to count users", "error", err)
	} else if userCount == 0 {
		if os.Getenv("FORGEBOX_FIRST_PASSWORD") != "" {
			slog.Info("no users found — POST /api/v1/setup to create the first admin account")
		} else {
			slog.Warn("no users found and FORGEBOX_FIRST_PASSWORD is not set — set it to enable first-time setup")
		}
	}

	if err = registry.LoadFromStore(ctx, store, secretBox); err != nil {
		slog.Warn("failed to load DB-backed providers", "error", err)
	}

	// Brain feature shares the same Postgres connection as the core store.
	embeddingProvider := cfg.Brain.EmbeddingProvider
	embeddingModel := cfg.Brain.EmbeddingModel
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	if embeddingProvider == "" {
		embeddingProvider = "openai"
	}

	var apiKey string
	if provCfg, ok := cfg.Providers[embeddingProvider]; ok {
		if key, ok := provCfg["api_key"].(string); ok {
			apiKey = key
		}
	}

	var embedder brain.Embedder
	if apiKey != "" {
		embedder = brain.NewOpenAIEmbedder(apiKey, embeddingModel)
		slog.Info("brain embedder configured", "provider", embeddingProvider, "model", embeddingModel)
	} else {
		slog.Warn("brain: no embedding API key found, using mock embedder")
		embedder = brain.NewMockEmbedder(1536)
	}

	var brainStore sdk.BrainStore = store
	brainSvc := brain.NewService(store, embedder)
	slog.Info("brain feature enabled")

	go runArchiveCleanup(ctx, brainSvc)

	sessionMgr := sessions.NewManager(store)
	permChecker := permissions.NewChecker(cfg.Auth, store)

	eng := engine.New(engine.Config{
		Registry:     registry,
		Orchestrator: orch,
		Permissions:  permChecker,
		Sessions:     sessionMgr,
	})

	bus := events.New(0)

	srv := gateway.New(gateway.Config{
		ListenAddr:     cfg.Server.Listen,
		GRPCListenAddr: cfg.Server.GRPCListen,
		Engine:         eng,
		Sessions:       sessionMgr,
		Registry:       registry,
		Store:          store,
		BrainService:   brainSvc,
		BrainStore:     brainStore,
		SecretBox:      secretBox,
		Events:         bus,
	})

	slog.Info("starting ForgeBox",
		"version", version,
		"http", cfg.Server.Listen,
		"grpc", cfg.Server.GRPCListen,
	)

	if err := srv.Run(ctx); err != nil {
		slog.Error("server error", "error", err)
		return err
	}
	return nil
}

// runArchiveCleanup runs hourly and purges brain files whose deleted_at is
// older than brain.ArchiveRetention.
func runArchiveCleanup(ctx context.Context, svc *brain.Service) {
	const interval = time.Hour
	tick := time.NewTicker(interval)
	defer tick.Stop()

	run := func() {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		n, err := svc.PurgeArchived(runCtx, brain.ArchiveRetention)
		if err != nil {
			slog.Error("brain archive cleanup failed", "error", err)
			return
		}
		if n > 0 {
			slog.Info("brain archive cleanup", "purged", n)
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			run()
		}
	}
}

func cmdRun() error {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: forgebox run <prompt> [--provider NAME] [--model NAME]")
		os.Exit(1)
	}
	prompt := os.Args[2]

	cfg, err := config.Load(configPath())
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Storage.DSN == "" {
		slog.Error("storage DSN missing — set FORGEBOX_DATABASE_URL or storage.dsn in config")
		os.Exit(1)
	}

	store, err := postgres.New(cfg.Storage.DSN)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}

	registry := plugins.NewRegistry()
	if err = registry.LoadBuiltins(cfg); err != nil {
		slog.Error("failed to load plugins", "error", err)
		os.Exit(1)
	}

	orch, err := vm.NewOrchestrator(cfg.VM)
	if err != nil {
		slog.Error("failed to init VM orchestrator", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	defer func() { _ = store.Close() }()
	defer orch.Shutdown(ctx)

	eng := engine.New(engine.Config{
		Registry:     registry,
		Orchestrator: orch,
		Permissions:  permissions.NewChecker(cfg.Auth, store),
		Sessions:     sessions.NewManager(store),
	})

	result, err := eng.Run(ctx, &engine.Task{
		Prompt:   prompt,
		Provider: firstProvider(cfg),
	})
	if err != nil {
		slog.Error("task failed", "error", err)
		return err
	}

	fmt.Println(result.Output)
	return nil
}

func cmdInit() {
	if err := config.WriteDefault("forgebox.yaml"); err != nil {
		slog.Error("failed to write config", "error", err)
		os.Exit(1)
	}
	fmt.Println("Created forgebox.yaml — edit it to configure providers and channels.")
}

func cmdStatus() {
	fmt.Println("TODO: query running gateway for status")
}

func configPath() string {
	if p := os.Getenv("FORGEBOX_CONFIG"); p != "" {
		return p
	}
	return "forgebox.yaml"
}

func firstProvider(cfg *config.Config) string {
	for name := range cfg.Providers {
		return name
	}
	return ""
}

func printUsage() {
	fmt.Println(`ForgeBox — Secure AI automation for every team

Usage:
  forgebox <command> [flags]

Commands:
  serve       Start the gateway server
  run         Run a one-shot task
  init        Create a default configuration file
  status      Show gateway status
  version     Print version information
  help        Show this help

Run 'forgebox <command> --help' for more information on a command.`)
}
