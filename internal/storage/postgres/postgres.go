package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"
)

// BrainDB holds the PostgreSQL connection pool for brain storage.
type BrainDB struct {
	db *sql.DB
}

// New opens a PostgreSQL connection and runs brain migrations.
func New(dsn string) (*BrainDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	bdb := &BrainDB{db: db}
	if err := bdb.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("brain migration: %w", err)
	}

	slog.Info("brain storage connected", "driver", "postgres")
	return bdb, nil
}

// Close closes the database connection pool.
func (b *BrainDB) Close() error {
	return b.db.Close()
}

func (b *BrainDB) migrate() error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,

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
		if _, err := b.db.ExecContext(context.Background(), m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	return nil
}
