// Package postgres implements ForgeBox storage on PostgreSQL.
//
// A single Store satisfies the full sdk.StoragePlugin (tasks, sessions,
// audit, users, automations, apps, providers) plus sdk.BrainStore (brain
// files, links, hashtags, graph, dream proposals — those use pgvector).
//
// One *sql.DB is shared across all responsibilities; brain features add
// their own tables alongside the core storage tables.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq" // registers the "postgres" database/sql driver
)

// Store holds the PostgreSQL connection pool for all ForgeBox storage.
type Store struct {
	db *sql.DB
}

// New opens a PostgreSQL connection and runs all migrations.
func New(dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	// Bound the pool — lib/pq's default is unlimited, which lets a burst of
	// concurrent gateway requests open arbitrary numbers of connections.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres migration: %w", err)
	}

	slog.Info("storage connected", "driver", "postgres")
	return s, nil
}

// Name returns the storage plugin name.
func (s *Store) Name() string { return "postgres" }

// Version returns the storage plugin version.
func (s *Store) Version() string { return "1.0.0" }

// Init is a no-op; configuration is handled in New.
func (s *Store) Init(_ context.Context, _ map[string]any) error { return nil }

// Shutdown closes the database connection pool.
func (s *Store) Shutdown(_ context.Context) error { return s.Close() }

// Close closes the database connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,

		// --- Core storage tables ---

		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL DEFAULT 'pending',
			prompt TEXT NOT NULL,
			result TEXT,
			provider TEXT,
			model TEXT,
			user_id TEXT,
			session_id TEXT,
			cost DOUBLE PRECISION DEFAULT 0,
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			error TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			started_at TIMESTAMPTZ,
			completed_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user ON tasks(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			provider TEXT,
			model TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT,
			tool_calls TEXT,
			tool_results TEXT,
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id)`,

		`CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL,
			user_id TEXT,
			task_id TEXT,
			action TEXT NOT NULL,
			tool TEXT,
			decision TEXT NOT NULL,
			reason TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_task ON audit_log(task_id)`,

		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			password_hash TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'viewer',
			team_ids TEXT,
			disabled BOOLEAN NOT NULL DEFAULT FALSE
		)`,

		`CREATE TABLE IF NOT EXISTS automations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL,
			sharing TEXT NOT NULL DEFAULT 'personal',
			team_id TEXT,
			trigger_config TEXT NOT NULL DEFAULT '{}',
			nodes TEXT NOT NULL DEFAULT '[]',
			edges TEXT NOT NULL DEFAULT '[]',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_automations_user ON automations(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_automations_team ON automations(team_id)`,

		`CREATE TABLE IF NOT EXISTS apps (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL,
			sharing TEXT NOT NULL DEFAULT 'personal',
			team_id TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			tools TEXT NOT NULL DEFAULT '[]',
			config TEXT NOT NULL DEFAULT '{}',
			url TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_apps_user ON apps(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_apps_team ON apps(team_id)`,

		`CREATE TABLE IF NOT EXISTS providers (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			name TEXT NOT NULL UNIQUE,
			config_encrypted TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,

		// --- Brain tables ---

		`CREATE TABLE IF NOT EXISTS brains (
			id TEXT PRIMARY KEY,
			automation_id TEXT NOT NULL UNIQUE,
			embedding_provider TEXT NOT NULL DEFAULT '',
			embedding_model TEXT NOT NULL DEFAULT '',
			embedding_dimension INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS brain_files (
			id TEXT PRIMARY KEY,
			brain_id TEXT NOT NULL REFERENCES brains(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			embedding vector,
			cluster_id INTEGER,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_by TEXT NOT NULL DEFAULT 'agent'
		)`,

		`CREATE INDEX IF NOT EXISTS idx_brain_files_brain ON brain_files(brain_id)`,

		`ALTER TABLE brain_files ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ`,

		`CREATE INDEX IF NOT EXISTS idx_brain_files_active ON brain_files(brain_id) WHERE deleted_at IS NULL`,

		`CREATE TABLE IF NOT EXISTS brain_links (
			source_file_id TEXT NOT NULL REFERENCES brain_files(id) ON DELETE CASCADE,
			target_file_id TEXT NOT NULL REFERENCES brain_files(id) ON DELETE CASCADE,
			PRIMARY KEY (source_file_id, target_file_id)
		)`,

		`CREATE TABLE IF NOT EXISTS brain_hashtags (
			file_id TEXT NOT NULL REFERENCES brain_files(id) ON DELETE CASCADE,
			tag TEXT NOT NULL,
			PRIMARY KEY (file_id, tag)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_brain_hashtags_tag ON brain_hashtags(tag)`,

		`CREATE TABLE IF NOT EXISTS brain_graph (
			brain_id TEXT PRIMARY KEY REFERENCES brains(id) ON DELETE CASCADE,
			clusters JSONB NOT NULL DEFAULT '[]',
			nodes JSONB NOT NULL DEFAULT '[]',
			computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS dream_proposals (
			id TEXT PRIMARY KEY,
			brain_id TEXT NOT NULL REFERENCES brains(id) ON DELETE CASCADE,
			snapshot JSONB NOT NULL DEFAULT '{}',
			changes JSONB NOT NULL DEFAULT '[]',
			summary TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			resolved_at TIMESTAMPTZ,
			resolved_by TEXT
		)`,

		`CREATE INDEX IF NOT EXISTS idx_dream_proposals_brain ON dream_proposals(brain_id)`,
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(context.Background(), m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	return nil
}
