// Package main is the entry point for the ForgeBox server and CLI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/forgebox/forgebox/internal/brain"
	"github.com/forgebox/forgebox/internal/config"
	"github.com/forgebox/forgebox/internal/engine"
	"github.com/forgebox/forgebox/internal/gateway"
	"github.com/forgebox/forgebox/internal/permissions"
	"github.com/forgebox/forgebox/internal/plugins"
	"github.com/forgebox/forgebox/internal/sessions"
	"github.com/forgebox/forgebox/internal/storage/postgres"
	"github.com/forgebox/forgebox/internal/storage/sqlite"
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
		cmdServe()
	case "run":
		cmdRun()
	case "init":
		cmdInit()
	case "status":
		cmdStatus()
	case "version":
		fmt.Printf("forgebox %s (commit: %s, built: %s)\n", version, commit, buildDate)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdServe() {
	cfg, err := config.Load(configPath())
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if err := telemetry.Init(cfg.Telemetry); err != nil {
		slog.Warn("telemetry init failed, continuing without", "error", err)
	}
	defer telemetry.Shutdown()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	store, err := sqlite.New(cfg.Storage.SQLite.Path)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

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

	registry := plugins.NewRegistry()
	if err := registry.LoadBuiltins(cfg); err != nil {
		slog.Error("failed to load plugins", "error", err)
		os.Exit(1)
	}

	// Initialize brain storage (PostgreSQL with pgvector).
	var brainSvc *brain.Service
	var brainStore sdk.BrainStore
	if cfg.Brain.PostgresDSN != "" {
		brainDB, err := postgres.New(cfg.Brain.PostgresDSN)
		if err != nil {
			slog.Warn("brain feature unavailable — postgres unreachable", "error", err)
		} else {
			defer brainDB.Close()

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

			brainStore = brainDB
			brainSvc = brain.NewService(brainDB, embedder)
			slog.Info("brain feature enabled")
		}
	}

	orch, err := vm.NewOrchestrator(cfg.VM)
	if err != nil {
		slog.Error("failed to init VM orchestrator", "error", err)
		os.Exit(1)
	}
	defer orch.Shutdown(ctx)

	sessionMgr := sessions.NewManager(store)
	permChecker := permissions.NewChecker(cfg.Auth, store)

	eng := engine.New(engine.Config{
		Registry:    registry,
		Orchestrator: orch,
		Permissions: permChecker,
		Sessions:    sessionMgr,
	})

	srv := gateway.New(gateway.Config{
		ListenAddr:     cfg.Server.Listen,
		GRPCListenAddr: cfg.Server.GRPCListen,
		Engine:         eng,
		Sessions:       sessionMgr,
		Registry:       registry,
		Store:          store,
		BrainService:   brainSvc,
		BrainStore:     brainStore,
	})

	slog.Info("starting ForgeBox",
		"version", version,
		"http", cfg.Server.Listen,
		"grpc", cfg.Server.GRPCListen,
	)

	if err := srv.Run(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func cmdRun() {
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	store, err := sqlite.New(cfg.Storage.SQLite.Path)
	if err != nil {
		slog.Error("failed to open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	registry := plugins.NewRegistry()
	if err := registry.LoadBuiltins(cfg); err != nil {
		slog.Error("failed to load plugins", "error", err)
		os.Exit(1)
	}

	orch, err := vm.NewOrchestrator(cfg.VM)
	if err != nil {
		slog.Error("failed to init VM orchestrator", "error", err)
		os.Exit(1)
	}
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
		os.Exit(1)
	}

	fmt.Println(result.Output)
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
