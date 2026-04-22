# Brain Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an Obsidian-like markdown knowledge base ("Brain") for ForgeBox agents with `[[links]]`, `#hashtags`, RAG search via pgvector, force-directed graph visualization, and a daily "dream" consolidation cycle.

**Architecture:** Each agent gets one brain stored in PostgreSQL with pgvector for embeddings. A brain service orchestrates CRUD, link/hashtag parsing, embedding, and graph precomputation. Agents interact via an explicit `brain` tool plugin. The frontend renders a force-directed dot graph (d3-force) with a rich markdown editor. A scheduled dream cycle proposes memory consolidation that users approve.

**Tech Stack:** Go 1.23, PostgreSQL 16 + pgvector, `lib/pq` + `pgx`, OpenAI/Anthropic embedding APIs, SvelteKit 2 + Svelte 5, d3-force, tiptap (markdown editor), SCSS

**Spec:** `docs/superpowers/specs/2026-04-20-brain-feature-design.md`

---

## File Structure

### Go Backend (new files)

| File | Responsibility |
|------|---------------|
| `pkg/sdk/brain.go` | Brain record types, BrainStore interface |
| `internal/brain/service.go` | BrainService: CRUD orchestration, coordinates parser + embedder + store |
| `internal/brain/parser.go` | Extract `[[links]]` and `#hashtags` from markdown |
| `internal/brain/parser_test.go` | Parser unit tests |
| `internal/brain/embedder.go` | Embedder interface + provider-based implementation |
| `internal/brain/embedder_test.go` | Embedder tests |
| `internal/brain/clusterer.go` | K-means clustering + 2D layout from embeddings |
| `internal/brain/clusterer_test.go` | Clusterer tests |
| `internal/brain/dreamer.go` | Dream cycle: consolidation prompt + proposal creation |
| `internal/brain/service_test.go` | Service integration tests |
| `internal/storage/postgres/postgres.go` | PostgreSQL connection pool + migration |
| `internal/storage/postgres/brain.go` | BrainStore PostgreSQL implementation |
| `internal/storage/postgres/brain_test.go` | Store integration tests |
| `internal/gateway/brain_handlers.go` | Brain REST API handlers |
| `internal/gateway/brain_handlers_test.go` | Handler unit tests |

### Go Backend (modified files)

| File | Change |
|------|--------|
| `pkg/sdk/plugin.go` | (if needed) no changes expected |
| `internal/config/config.go` | Add `BrainConfig` with embedding settings |
| `internal/plugins/registry.go` | Register brain tool in `registerBuiltinTools()` |
| `internal/gateway/server.go` | Add brain routes in `registerRoutes()`, add `brainService` field |
| `cmd/forgebox/main.go` | Init PostgreSQL connection + BrainService + pass to gateway |
| `go.mod` | Add `github.com/lib/pq`, `github.com/pgvector/pgvector-go` |
| `docker-compose.dev.yml` | Add pgvector extension init SQL |

### Frontend (new files)

| File | Responsibility |
|------|---------------|
| `web/src/lib/api/brain.ts` | Brain API client functions |
| `web/src/lib/stores/brain.svelte.ts` | Brain state store (Svelte 5 runes) |
| `web/src/routes/(app)/agents/[id]/brain/+page.svelte` | Brain page (graph + editor layout) |
| `web/src/lib/components/brain/BrainGraph.svelte` | Force-directed graph canvas |
| `web/src/lib/components/brain/BrainEditor.svelte` | Rich markdown editor with autocomplete |
| `web/src/lib/components/brain/BrainFileMeta.svelte` | File title + metadata bar |
| `web/src/lib/components/brain/BrainSearch.svelte` | Semantic search input |
| `web/src/lib/components/brain/DreamPanel.svelte` | Dream proposals list + approve/reject |

### Frontend (modified files)

| File | Change |
|------|--------|
| `web/src/lib/api/types.ts` | Add Brain types |
| `web/src/routes/(app)/agents/[id]/+page.svelte` | Add "Brain" link/button to agent detail |
| `web/package.json` | Add `d3-force`, `@tiptap/*` dependencies |

---

## Task 1: SDK Types and BrainStore Interface

**Files:**
- Create: `pkg/sdk/brain.go`

- [ ] **Step 1: Write the Brain SDK types and store interface**

```go
// pkg/sdk/brain.go
package sdk

import (
	"context"
	"time"
)

// BrainRecord represents an agent's knowledge base.
type BrainRecord struct {
	ID                 string    `json:"id"`
	AutomationID       string    `json:"automation_id"`
	EmbeddingProvider  string    `json:"embedding_provider,omitempty"`
	EmbeddingModel     string    `json:"embedding_model,omitempty"`
	EmbeddingDimension int       `json:"embedding_dimension,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// BrainFile is a markdown document within a brain.
type BrainFile struct {
	ID        string    `json:"id"`
	BrainID   string    `json:"brain_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"-"`
	ClusterID *int      `json:"cluster_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"` // "agent" or user UUID
}

// BrainFileWithMeta is a BrainFile with extracted metadata for API responses.
type BrainFileWithMeta struct {
	BrainFile
	Hashtags []string `json:"hashtags"`
	Links    []string `json:"links"` // titles of linked files
	Score    float64  `json:"score,omitempty"` // relevance score from search
}

// BrainLink represents a [[link]] between two brain files.
type BrainLink struct {
	SourceFileID string `json:"source_file_id"`
	TargetFileID string `json:"target_file_id"`
}

// BrainGraph is precomputed visualization data for a brain.
type BrainGraph struct {
	BrainID    string           `json:"brain_id"`
	Clusters   []GraphCluster   `json:"clusters"`
	Nodes      []GraphNode      `json:"nodes"`
	Links      []BrainLink      `json:"links"`
	ComputedAt time.Time        `json:"computed_at"`
}

// GraphCluster is a semantic cluster of brain files.
type GraphCluster struct {
	ID    int    `json:"id"`
	Color string `json:"color"`
	Label string `json:"label"`
}

// GraphNode is a positioned brain file in the graph.
type GraphNode struct {
	FileID    string  `json:"file_id"`
	Title     string  `json:"title"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	ClusterID int     `json:"cluster_id"`
	Hashtags  []string `json:"hashtags"`
}

// DreamProposalStatus is the lifecycle state of a dream proposal.
type DreamProposalStatus string

const (
	DreamPending  DreamProposalStatus = "pending"
	DreamApproved DreamProposalStatus = "approved"
	DreamRejected DreamProposalStatus = "rejected"
)

// DreamProposal is a pending brain reorganization from the dream cycle.
type DreamProposal struct {
	ID         string              `json:"id"`
	BrainID    string              `json:"brain_id"`
	Snapshot   string              `json:"snapshot,omitempty"` // JSON blob
	Changes    string              `json:"changes"`           // JSON array of DreamChange
	Summary    string              `json:"summary"`
	Status     DreamProposalStatus `json:"status"`
	CreatedAt  time.Time           `json:"created_at"`
	ResolvedAt *time.Time          `json:"resolved_at,omitempty"`
	ResolvedBy string              `json:"resolved_by,omitempty"`
}

// DreamChange is a single mutation proposed by the dream cycle.
type DreamChange struct {
	Action     string `json:"action"` // "create", "edit", "delete"
	FileID     string `json:"file_id,omitempty"`
	NewTitle   string `json:"new_title,omitempty"`
	NewContent string `json:"new_content,omitempty"`
	Reason     string `json:"reason"`
}

// BrainStore manages brain persistence.
type BrainStore interface {
	CreateBrain(ctx context.Context, brain *BrainRecord) error
	GetBrain(ctx context.Context, id string) (*BrainRecord, error)
	GetBrainByAutomation(ctx context.Context, automationID string) (*BrainRecord, error)
	UpdateBrain(ctx context.Context, brain *BrainRecord) error
	DeleteBrain(ctx context.Context, id string) error

	CreateFile(ctx context.Context, file *BrainFile) error
	UpdateFile(ctx context.Context, file *BrainFile) error
	DeleteFile(ctx context.Context, fileID string) error
	GetFile(ctx context.Context, fileID string) (*BrainFile, error)
	ListFiles(ctx context.Context, brainID string) ([]*BrainFile, error)
	SearchByEmbedding(ctx context.Context, brainID string, vec []float32, limit int) ([]*BrainFileWithMeta, error)

	SetFileHashtags(ctx context.Context, fileID string, tags []string) error
	GetFileHashtags(ctx context.Context, fileID string) ([]string, error)
	ListHashtags(ctx context.Context, brainID string) ([]string, error)

	SetFileLinks(ctx context.Context, sourceFileID string, targetFileIDs []string) error
	GetFileLinks(ctx context.Context, brainID string) ([]BrainLink, error)

	SaveGraph(ctx context.Context, graph *BrainGraph) error
	GetGraph(ctx context.Context, brainID string) (*BrainGraph, error)

	CreateDreamProposal(ctx context.Context, proposal *DreamProposal) error
	GetDreamProposal(ctx context.Context, proposalID string) (*DreamProposal, error)
	ListDreamProposals(ctx context.Context, brainID string) ([]*DreamProposal, error)
	UpdateDreamProposalStatus(ctx context.Context, proposalID string, status DreamProposalStatus, resolvedBy string) error
}
```

- [ ] **Step 2: Verify the file compiles**

Run: `cd /home/trystan/forgebox && go build ./pkg/sdk/...`
Expected: clean build, no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/sdk/brain.go
git commit -m "feat(sdk): add Brain types and BrainStore interface"
```

---

## Task 2: Markdown Parser

**Files:**
- Create: `internal/brain/parser.go`
- Create: `internal/brain/parser_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/brain/parser_test.go
package brain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single link",
			content: "See [[Project Setup]] for details.",
			want:    []string{"Project Setup"},
		},
		{
			name:    "multiple links",
			content: "Check [[Auth Guide]] and [[API Design]].",
			want:    []string{"Auth Guide", "API Design"},
		},
		{
			name:    "no links",
			content: "Plain markdown with no links.",
			want:    nil,
		},
		{
			name:    "link at start",
			content: "[[Quick Start]] is the first step.",
			want:    []string{"Quick Start"},
		},
		{
			name:    "duplicate links deduplicated",
			content: "See [[Auth]] and also [[Auth]] again.",
			want:    []string{"Auth"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLinks(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractHashtags(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single tag",
			content: "This is about #deployment.",
			want:    []string{"deployment"},
		},
		{
			name:    "multiple tags",
			content: "#auth and #security are related.",
			want:    []string{"auth", "security"},
		},
		{
			name:    "tag at line start",
			content: "#setup\nSome content.",
			want:    []string{"setup"},
		},
		{
			name:    "no tags",
			content: "Plain markdown content.",
			want:    nil,
		},
		{
			name:    "normalizes to lowercase",
			content: "About #DevOps and #CI-CD.",
			want:    []string{"devops", "ci-cd"},
		},
		{
			name:    "ignores anchors in links",
			content: "See [link](#heading) for info.",
			want:    nil,
		},
		{
			name:    "duplicates deduplicated",
			content: "#auth and #auth again.",
			want:    []string{"auth"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHashtags(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/trystan/forgebox && go test ./internal/brain/ -v -run "TestExtract"`
Expected: FAIL — `ExtractLinks` and `ExtractHashtags` undefined

- [ ] **Step 3: Implement the parser**

```go
// internal/brain/parser.go
package brain

import (
	"regexp"
	"strings"
)

var (
	linkPattern    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	hashtagPattern = regexp.MustCompile(`(?:^|\s)#([a-zA-Z0-9_-]+)`)
)

// ExtractLinks finds all [[link]] references in markdown content.
// Returns deduplicated titles in order of first appearance.
func ExtractLinks(content string) []string {
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var links []string
	for _, m := range matches {
		title := strings.TrimSpace(m[1])
		if title != "" && !seen[title] {
			seen[title] = true
			links = append(links, title)
		}
	}
	return links
}

// ExtractHashtags finds all #hashtag references in markdown content.
// Returns deduplicated, lowercase tags in order of first appearance.
// Ignores anchor links like [text](#heading).
func ExtractHashtags(content string) []string {
	matches := hashtagPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/trystan/forgebox && go test ./internal/brain/ -v -run "TestExtract"`
Expected: PASS — all test cases green

- [ ] **Step 5: Commit**

```bash
git add internal/brain/parser.go internal/brain/parser_test.go
git commit -m "feat(brain): add markdown parser for [[links]] and #hashtags"
```

---

## Task 3: Embedder Interface and Provider Implementation

**Files:**
- Create: `internal/brain/embedder.go`
- Create: `internal/brain/embedder_test.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add BrainConfig to config**

Add to `internal/config/config.go`:

```go
// In the Config struct, add:
Brain BrainConfig `yaml:"brain"`

// Add the new config type:
// BrainConfig configures the brain knowledge base feature.
type BrainConfig struct {
	EmbeddingProvider string `yaml:"embedding_provider"` // default provider for embeddings
	EmbeddingModel   string `yaml:"embedding_model"`    // default model for embeddings
	PostgresDSN      string `yaml:"postgres_dsn"`       // pgvector-enabled PostgreSQL DSN
	DreamSchedule    string `yaml:"dream_schedule"`     // cron expression, default "0 2 * * *"
}
```

In `Defaults()`, add:

```go
Brain: BrainConfig{
	DreamSchedule: "0 2 * * *",
},
```

In `applyEnvOverrides()`, add:

```go
if v := os.Getenv("FORGEBOX_BRAIN_POSTGRES_DSN"); v != "" {
	c.Brain.PostgresDSN = v
}
```

- [ ] **Step 2: Write the embedder interface and implementation**

```go
// internal/brain/embedder.go
package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder computes vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimension() int
}

// OpenAIEmbedder calls the OpenAI embeddings API.
type OpenAIEmbedder struct {
	apiKey     string
	model      string
	dimension  int
	httpClient *http.Client
}

// NewOpenAIEmbedder creates an embedder using the OpenAI API.
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	dim := 1536
	if model == "text-embedding-3-large" {
		dim = 3072
	}
	return &OpenAIEmbedder{
		apiKey:    apiKey,
		model:     model,
		dimension: dim,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *OpenAIEmbedder) Dimension() int { return e.dimension }

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(map[string]any{
		"model": e.model,
		"input": text,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse embedding response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return result.Data[0].Embedding, nil
}
```

- [ ] **Step 3: Write a test with a mock embedder**

```go
// internal/brain/embedder_test.go
package brain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmbedder is a test double that returns a fixed-dimension vector.
type MockEmbedder struct {
	dim    int
	called int
}

func NewMockEmbedder(dim int) *MockEmbedder {
	return &MockEmbedder{dim: dim}
}

func (m *MockEmbedder) Dimension() int { return m.dim }

func (m *MockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	m.called++
	vec := make([]float32, m.dim)
	// Simple deterministic embedding: hash-like based on text length
	for i := range vec {
		vec[i] = float32(len(text)+i) / float32(m.dim)
	}
	return vec, nil
}

func TestMockEmbedder(t *testing.T) {
	emb := NewMockEmbedder(4)
	vec, err := emb.Embed(context.Background(), "test")
	require.NoError(t, err)
	assert.Len(t, vec, 4)
	assert.Equal(t, 1, emb.called)
	assert.Equal(t, 4, emb.Dimension())
}
```

- [ ] **Step 4: Run tests**

Run: `cd /home/trystan/forgebox && go test ./internal/brain/ -v -run "TestMockEmbedder"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/embedder.go internal/brain/embedder_test.go internal/config/config.go
git commit -m "feat(brain): add embedder interface and OpenAI implementation"
```

---

## Task 4: PostgreSQL Connection and Migrations

**Files:**
- Create: `internal/storage/postgres/postgres.go`
- Modify: `docker-compose.dev.yml`
- Modify: `go.mod` (via go get)

- [ ] **Step 1: Add Go dependencies**

Run: `cd /home/trystan/forgebox && go get github.com/lib/pq github.com/pgvector/pgvector-go`

- [ ] **Step 2: Update docker-compose for pgvector**

In `docker-compose.dev.yml`, change the postgres image and add an init script:

Change `image: postgres:16-alpine` to `image: pgvector/pgvector:pg16` and add init environment:

```yaml
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: forgebox
      POSTGRES_PASSWORD: forgebox
      POSTGRES_DB: forgebox
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U forgebox"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped
```

Add the brain DSN env var to the backend service:

```yaml
      - FORGEBOX_BRAIN_POSTGRES_DSN=postgres://forgebox:forgebox@postgres:5432/forgebox?sslmode=disable
```

- [ ] **Step 3: Write the PostgreSQL connection and migration code**

```go
// internal/storage/postgres/postgres.go
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
```

- [ ] **Step 4: Verify it compiles**

Run: `cd /home/trystan/forgebox && go build ./internal/storage/postgres/...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/storage/postgres/postgres.go docker-compose.dev.yml go.mod go.sum
git commit -m "feat(brain): add PostgreSQL connection with pgvector migrations"
```

---

## Task 5: BrainStore PostgreSQL Implementation

**Files:**
- Create: `internal/storage/postgres/brain.go`
- Create: `internal/storage/postgres/brain_test.go`

- [ ] **Step 1: Write the store implementation**

```go
// internal/storage/postgres/brain.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// Verify BrainDB implements BrainStore.
var _ sdk.BrainStore = (*BrainDB)(nil)

func (b *BrainDB) CreateBrain(ctx context.Context, brain *sdk.BrainRecord) error {
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO brains (id, automation_id, embedding_provider, embedding_model, embedding_dimension, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		brain.ID, brain.AutomationID, brain.EmbeddingProvider, brain.EmbeddingModel,
		brain.EmbeddingDimension, brain.CreatedAt, brain.UpdatedAt,
	)
	return err
}

func (b *BrainDB) GetBrain(ctx context.Context, id string) (*sdk.BrainRecord, error) {
	var rec sdk.BrainRecord
	err := b.db.QueryRowContext(ctx,
		`SELECT id, automation_id, embedding_provider, embedding_model, embedding_dimension, created_at, updated_at
		 FROM brains WHERE id = $1`, id,
	).Scan(&rec.ID, &rec.AutomationID, &rec.EmbeddingProvider, &rec.EmbeddingModel,
		&rec.EmbeddingDimension, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (b *BrainDB) GetBrainByAutomation(ctx context.Context, automationID string) (*sdk.BrainRecord, error) {
	var rec sdk.BrainRecord
	err := b.db.QueryRowContext(ctx,
		`SELECT id, automation_id, embedding_provider, embedding_model, embedding_dimension, created_at, updated_at
		 FROM brains WHERE automation_id = $1`, automationID,
	).Scan(&rec.ID, &rec.AutomationID, &rec.EmbeddingProvider, &rec.EmbeddingModel,
		&rec.EmbeddingDimension, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (b *BrainDB) UpdateBrain(ctx context.Context, brain *sdk.BrainRecord) error {
	brain.UpdatedAt = time.Now()
	_, err := b.db.ExecContext(ctx,
		`UPDATE brains SET embedding_provider=$1, embedding_model=$2, embedding_dimension=$3, updated_at=$4 WHERE id=$5`,
		brain.EmbeddingProvider, brain.EmbeddingModel, brain.EmbeddingDimension, brain.UpdatedAt, brain.ID,
	)
	return err
}

func (b *BrainDB) DeleteBrain(ctx context.Context, id string) error {
	_, err := b.db.ExecContext(ctx, `DELETE FROM brains WHERE id = $1`, id)
	return err
}

func (b *BrainDB) CreateFile(ctx context.Context, file *sdk.BrainFile) error {
	var embeddingStr *string
	if file.Embedding != nil {
		v := pgvector.NewVector(file.Embedding)
		s := v.String()
		embeddingStr = &s
	}
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO brain_files (id, brain_id, title, content, embedding, cluster_id, created_at, updated_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		file.ID, file.BrainID, file.Title, file.Content, embeddingStr,
		file.ClusterID, file.CreatedAt, file.UpdatedAt, file.CreatedBy,
	)
	return err
}

func (b *BrainDB) UpdateFile(ctx context.Context, file *sdk.BrainFile) error {
	file.UpdatedAt = time.Now()
	var embeddingStr *string
	if file.Embedding != nil {
		v := pgvector.NewVector(file.Embedding)
		s := v.String()
		embeddingStr = &s
	}
	_, err := b.db.ExecContext(ctx,
		`UPDATE brain_files SET title=$1, content=$2, embedding=$3, cluster_id=$4, updated_at=$5 WHERE id=$6`,
		file.Title, file.Content, embeddingStr, file.ClusterID, file.UpdatedAt, file.ID,
	)
	return err
}

func (b *BrainDB) DeleteFile(ctx context.Context, fileID string) error {
	_, err := b.db.ExecContext(ctx, `DELETE FROM brain_files WHERE id = $1`, fileID)
	return err
}

func (b *BrainDB) GetFile(ctx context.Context, fileID string) (*sdk.BrainFile, error) {
	var rec sdk.BrainFile
	err := b.db.QueryRowContext(ctx,
		`SELECT id, brain_id, title, content, cluster_id, created_at, updated_at, created_by
		 FROM brain_files WHERE id = $1`, fileID,
	).Scan(&rec.ID, &rec.BrainID, &rec.Title, &rec.Content,
		&rec.ClusterID, &rec.CreatedAt, &rec.UpdatedAt, &rec.CreatedBy)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (b *BrainDB) ListFiles(ctx context.Context, brainID string) ([]*sdk.BrainFile, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, brain_id, title, content, cluster_id, created_at, updated_at, created_by
		 FROM brain_files WHERE brain_id = $1 ORDER BY created_at`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*sdk.BrainFile
	for rows.Next() {
		var f sdk.BrainFile
		if err := rows.Scan(&f.ID, &f.BrainID, &f.Title, &f.Content,
			&f.ClusterID, &f.CreatedAt, &f.UpdatedAt, &f.CreatedBy); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, rows.Err()
}

func (b *BrainDB) SearchByEmbedding(ctx context.Context, brainID string, vec []float32, limit int) ([]*sdk.BrainFileWithMeta, error) {
	v := pgvector.NewVector(vec)
	rows, err := b.db.QueryContext(ctx,
		`SELECT f.id, f.brain_id, f.title, f.content, f.cluster_id, f.created_at, f.updated_at, f.created_by,
		        1 - (f.embedding <=> $1) AS score
		 FROM brain_files f
		 WHERE f.brain_id = $2 AND f.embedding IS NOT NULL
		 ORDER BY f.embedding <=> $1
		 LIMIT $3`,
		v.String(), brainID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*sdk.BrainFileWithMeta
	for rows.Next() {
		var f sdk.BrainFileWithMeta
		if err := rows.Scan(&f.ID, &f.BrainID, &f.Title, &f.Content,
			&f.ClusterID, &f.CreatedAt, &f.UpdatedAt, &f.CreatedBy, &f.Score); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, rows.Err()
}

func (b *BrainDB) SetFileHashtags(ctx context.Context, fileID string, tags []string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM brain_hashtags WHERE file_id = $1`, fileID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO brain_hashtags (file_id, tag) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			fileID, tag,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (b *BrainDB) GetFileHashtags(ctx context.Context, fileID string) ([]string, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT tag FROM brain_hashtags WHERE file_id = $1 ORDER BY tag`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (b *BrainDB) ListHashtags(ctx context.Context, brainID string) ([]string, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT DISTINCT h.tag FROM brain_hashtags h
		 JOIN brain_files f ON f.id = h.file_id
		 WHERE f.brain_id = $1
		 ORDER BY h.tag`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (b *BrainDB) SetFileLinks(ctx context.Context, sourceFileID string, targetFileIDs []string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM brain_links WHERE source_file_id = $1`, sourceFileID); err != nil {
		return err
	}
	for _, targetID := range targetFileIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO brain_links (source_file_id, target_file_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			sourceFileID, targetID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (b *BrainDB) GetFileLinks(ctx context.Context, brainID string) ([]sdk.BrainLink, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT l.source_file_id, l.target_file_id FROM brain_links l
		 JOIN brain_files f ON f.id = l.source_file_id
		 WHERE f.brain_id = $1`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []sdk.BrainLink
	for rows.Next() {
		var l sdk.BrainLink
		if err := rows.Scan(&l.SourceFileID, &l.TargetFileID); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (b *BrainDB) SaveGraph(ctx context.Context, graph *sdk.BrainGraph) error {
	clusters, err := json.Marshal(graph.Clusters)
	if err != nil {
		return fmt.Errorf("marshal clusters: %w", err)
	}
	nodes, err := json.Marshal(graph.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}

	_, err = b.db.ExecContext(ctx,
		`INSERT INTO brain_graph (brain_id, clusters, nodes, computed_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (brain_id) DO UPDATE SET clusters=$2, nodes=$3, computed_at=$4`,
		graph.BrainID, string(clusters), string(nodes), graph.ComputedAt,
	)
	return err
}

func (b *BrainDB) GetGraph(ctx context.Context, brainID string) (*sdk.BrainGraph, error) {
	var graph sdk.BrainGraph
	var clustersJSON, nodesJSON string
	err := b.db.QueryRowContext(ctx,
		`SELECT brain_id, clusters, nodes, computed_at FROM brain_graph WHERE brain_id = $1`, brainID,
	).Scan(&graph.BrainID, &clustersJSON, &nodesJSON, &graph.ComputedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(clustersJSON), &graph.Clusters); err != nil {
		return nil, fmt.Errorf("unmarshal clusters: %w", err)
	}
	if err := json.Unmarshal([]byte(nodesJSON), &graph.Nodes); err != nil {
		return nil, fmt.Errorf("unmarshal nodes: %w", err)
	}

	links, err := b.GetFileLinks(ctx, brainID)
	if err != nil {
		return nil, fmt.Errorf("get links: %w", err)
	}
	graph.Links = links

	return &graph, nil
}

func (b *BrainDB) CreateDreamProposal(ctx context.Context, p *sdk.DreamProposal) error {
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO dream_proposals (id, brain_id, snapshot, changes, summary, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.BrainID, p.Snapshot, p.Changes, p.Summary, p.Status, p.CreatedAt,
	)
	return err
}

func (b *BrainDB) GetDreamProposal(ctx context.Context, proposalID string) (*sdk.DreamProposal, error) {
	var p sdk.DreamProposal
	err := b.db.QueryRowContext(ctx,
		`SELECT id, brain_id, snapshot, changes, summary, status, created_at, resolved_at, resolved_by
		 FROM dream_proposals WHERE id = $1`, proposalID,
	).Scan(&p.ID, &p.BrainID, &p.Snapshot, &p.Changes, &p.Summary, &p.Status,
		&p.CreatedAt, &p.ResolvedAt, &p.ResolvedBy)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (b *BrainDB) ListDreamProposals(ctx context.Context, brainID string) ([]*sdk.DreamProposal, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, brain_id, summary, status, created_at, resolved_at, resolved_by
		 FROM dream_proposals WHERE brain_id = $1 ORDER BY created_at DESC`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proposals []*sdk.DreamProposal
	for rows.Next() {
		var p sdk.DreamProposal
		if err := rows.Scan(&p.ID, &p.BrainID, &p.Summary, &p.Status,
			&p.CreatedAt, &p.ResolvedAt, &p.ResolvedBy); err != nil {
			return nil, err
		}
		proposals = append(proposals, &p)
	}
	return proposals, rows.Err()
}

func (b *BrainDB) UpdateDreamProposalStatus(ctx context.Context, proposalID string, status sdk.DreamProposalStatus, resolvedBy string) error {
	now := time.Now()
	_, err := b.db.ExecContext(ctx,
		`UPDATE dream_proposals SET status=$1, resolved_at=$2, resolved_by=$3 WHERE id=$4`,
		status, now, resolvedBy, proposalID,
	)
	return err
}

// resolveFileIDsByTitle converts [[link]] titles to file IDs within a brain.
func (b *BrainDB) ResolveFileIDsByTitle(ctx context.Context, brainID string, titles []string) (map[string]string, error) {
	if len(titles) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(titles))
	args := []any{brainID}
	for i, title := range titles {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, title)
	}
	query := fmt.Sprintf(
		`SELECT id, title FROM brain_files WHERE brain_id = $1 AND title IN (%s)`,
		strings.Join(placeholders, ","),
	)
	rows, err := b.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		result[title] = id
	}
	return result, rows.Err()
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/trystan/forgebox && go build ./internal/storage/postgres/...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/storage/postgres/brain.go
git commit -m "feat(brain): implement BrainStore for PostgreSQL with pgvector"
```

---

## Task 6: Brain Service

**Files:**
- Create: `internal/brain/service.go`
- Create: `internal/brain/service_test.go`

- [ ] **Step 1: Write the brain service**

```go
// internal/brain/service.go
package brain

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Service orchestrates brain operations.
type Service struct {
	store    sdk.BrainStore
	embedder Embedder
}

// NewService creates a new brain service.
func NewService(store sdk.BrainStore, embedder Embedder) *Service {
	return &Service{
		store:    store,
		embedder: embedder,
	}
}

// GetOrCreateBrain returns the brain for an automation, creating one if needed.
func (s *Service) GetOrCreateBrain(ctx context.Context, automationID string) (*sdk.BrainRecord, error) {
	brain, err := s.store.GetBrainByAutomation(ctx, automationID)
	if err == nil {
		return brain, nil
	}

	now := time.Now()
	brain = &sdk.BrainRecord{
		ID:                 uuid.New().String(),
		AutomationID:       automationID,
		EmbeddingDimension: s.embedder.Dimension(),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.store.CreateBrain(ctx, brain); err != nil {
		return nil, fmt.Errorf("create brain: %w", err)
	}
	return brain, nil
}

// CreateFile creates a new brain file, extracts links/hashtags, and embeds it.
func (s *Service) CreateFile(ctx context.Context, brainID, title, content, createdBy string) (*sdk.BrainFile, error) {
	now := time.Now()
	file := &sdk.BrainFile{
		ID:        uuid.New().String(),
		BrainID:   brainID,
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
	}

	// Embed the content.
	embedding, err := s.embedder.Embed(ctx, title+"\n"+content)
	if err != nil {
		slog.Warn("embedding failed, saving file without embedding", "error", err, "file_id", file.ID)
	} else {
		file.Embedding = embedding
	}

	if err := s.store.CreateFile(ctx, file); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	// Extract and save hashtags.
	tags := ExtractHashtags(content)
	if len(tags) > 0 {
		if err := s.store.SetFileHashtags(ctx, file.ID, tags); err != nil {
			slog.Warn("failed to save hashtags", "error", err, "file_id", file.ID)
		}
	}

	// Extract and resolve links.
	linkTitles := ExtractLinks(content)
	if err := s.resolveAndSaveLinks(ctx, brainID, file.ID, linkTitles); err != nil {
		slog.Warn("failed to save links", "error", err, "file_id", file.ID)
	}

	return file, nil
}

// UpdateFile updates a brain file, re-extracts links/hashtags, and re-embeds.
func (s *Service) UpdateFile(ctx context.Context, fileID, title, content string) (*sdk.BrainFile, error) {
	file, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	file.Title = title
	file.Content = content

	embedding, err := s.embedder.Embed(ctx, title+"\n"+content)
	if err != nil {
		slog.Warn("embedding failed, keeping old embedding", "error", err, "file_id", file.ID)
	} else {
		file.Embedding = embedding
	}

	if err := s.store.UpdateFile(ctx, file); err != nil {
		return nil, fmt.Errorf("update file: %w", err)
	}

	tags := ExtractHashtags(content)
	if err := s.store.SetFileHashtags(ctx, file.ID, tags); err != nil {
		slog.Warn("failed to save hashtags", "error", err, "file_id", file.ID)
	}

	linkTitles := ExtractLinks(content)
	if err := s.resolveAndSaveLinks(ctx, file.BrainID, file.ID, linkTitles); err != nil {
		slog.Warn("failed to save links", "error", err, "file_id", file.ID)
	}

	return file, nil
}

// Search performs a semantic search across brain files using RAG.
func (s *Service) Search(ctx context.Context, brainID, query string, limit int) ([]*sdk.BrainFileWithMeta, error) {
	vec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	return s.store.SearchByEmbedding(ctx, brainID, vec, limit)
}

func (s *Service) resolveAndSaveLinks(ctx context.Context, brainID, sourceFileID string, titles []string) error {
	if len(titles) == 0 {
		return s.store.SetFileLinks(ctx, sourceFileID, nil)
	}

	// We need the store to support title lookup. Use the postgres-specific method
	// if available, otherwise skip link resolution.
	type titleResolver interface {
		ResolveFileIDsByTitle(ctx context.Context, brainID string, titles []string) (map[string]string, error)
	}
	resolver, ok := s.store.(titleResolver)
	if !ok {
		return nil
	}

	titleToID, err := resolver.ResolveFileIDsByTitle(ctx, brainID, titles)
	if err != nil {
		return fmt.Errorf("resolve titles: %w", err)
	}

	var targetIDs []string
	for _, title := range titles {
		if id, found := titleToID[title]; found {
			targetIDs = append(targetIDs, id)
		}
	}
	return s.store.SetFileLinks(ctx, sourceFileID, targetIDs)
}
```

- [ ] **Step 2: Write service test**

```go
// internal/brain/service_test.go
package brain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test verifies the service orchestration with a mock embedder.
// Full integration tests with PostgreSQL are in internal/storage/postgres/brain_test.go.
func TestServiceCreateFile_ExtractsMetadata(t *testing.T) {
	// This tests the parser integration — not the store.
	content := "# Deployment Guide\n\nSee [[Auth Setup]] for auth. #deployment #infrastructure"
	links := ExtractLinks(content)
	tags := ExtractHashtags(content)

	assert.Equal(t, []string{"Auth Setup"}, links)
	assert.Equal(t, []string{"deployment", "infrastructure"}, tags)
}

func TestMockEmbedder_DeterministicOutput(t *testing.T) {
	emb := NewMockEmbedder(8)
	vec1, err := emb.Embed(context.Background(), "hello")
	require.NoError(t, err)
	vec2, err := emb.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, vec1, vec2, "same input should produce same embedding")
}
```

- [ ] **Step 3: Run tests**

Run: `cd /home/trystan/forgebox && go test ./internal/brain/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/brain/service.go internal/brain/service_test.go
git commit -m "feat(brain): add BrainService for CRUD orchestration with embedding and parsing"
```

---

## Task 7: Brain API Handlers

**Files:**
- Create: `internal/gateway/brain_handlers.go`
- Modify: `internal/gateway/server.go`

- [ ] **Step 1: Write the brain handlers**

```go
// internal/gateway/brain_handlers.go
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
		// No graph computed yet — return empty.
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

	// Apply changes.
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
```

- [ ] **Step 2: Add brain fields and routes to server.go**

In `internal/gateway/server.go`, add to the `Server` struct:

```go
brainService *brain.Service
brainStore   sdk.BrainStore
```

In `Config` struct add:

```go
BrainService *brain.Service
BrainStore   sdk.BrainStore
```

In `New()`, add:

```go
s.brainService = cfg.BrainService
s.brainStore = cfg.BrainStore
```

In `registerRoutes()`, add the brain routes:

```go
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
```

Add the import: `"github.com/forgebox/forgebox/internal/brain"`

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/trystan/forgebox && go build ./internal/gateway/...`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/brain_handlers.go internal/gateway/server.go
git commit -m "feat(brain): add REST API handlers for brain CRUD, search, and dreams"
```

---

## Task 8: Brain Tool Plugin

**Files:**
- Modify: `internal/plugins/registry.go`

- [ ] **Step 1: Register the brain tool**

In `internal/plugins/registry.go`, add the brain tool to `registerBuiltinTools()`:

```go
&builtinTool{
	name: "brain",
	desc: "Search, read, and write to your persistent memory. Actions: search (query your memory), read (get a specific file), write (create or update a file), list (list all files), delete (remove a file).",
},
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/trystan/forgebox && go build ./internal/plugins/...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/plugins/registry.go
git commit -m "feat(brain): register brain tool plugin for agent use"
```

---

## Task 9: Startup Wiring

**Files:**
- Modify: `cmd/forgebox/main.go`

- [ ] **Step 1: Wire brain service into startup**

In `cmd/forgebox/main.go`, in `cmdServe()` after the registry is loaded, add brain initialization:

```go
// Initialize brain storage (PostgreSQL with pgvector).
var brainSvc *brain.Service
var brainStore sdk.BrainStore
if cfg.Brain.PostgresDSN != "" {
	brainDB, err := postgres.New(cfg.Brain.PostgresDSN)
	if err != nil {
		slog.Error("failed to init brain storage", "error", err)
		os.Exit(1)
	}
	defer brainDB.Close()

	// Resolve embedder from configured providers.
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
} else {
	slog.Info("brain feature disabled (no FORGEBOX_BRAIN_POSTGRES_DSN)")
}
```

Pass to the gateway:

```go
srv := gateway.New(gateway.Config{
	// ... existing fields ...
	BrainService: brainSvc,
	BrainStore:   brainStore,
})
```

Add imports:

```go
"github.com/forgebox/forgebox/internal/brain"
"github.com/forgebox/forgebox/internal/storage/postgres"
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/trystan/forgebox && go build ./cmd/forgebox/...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add cmd/forgebox/main.go
git commit -m "feat(brain): wire brain service into server startup"
```

---

## Task 10: Frontend Types and API Client

**Files:**
- Modify: `web/src/lib/api/types.ts`
- Create: `web/src/lib/api/brain.ts`

- [ ] **Step 1: Add brain types**

Add to `web/src/lib/api/types.ts`:

```typescript
// --- Brain ---

export interface Brain {
	id: string;
	automation_id: string;
	embedding_provider?: string;
	embedding_model?: string;
	embedding_dimension: number;
	created_at: string;
	updated_at: string;
}

export interface BrainFile {
	id: string;
	brain_id: string;
	title: string;
	content: string;
	cluster_id?: number;
	created_at: string;
	updated_at: string;
	created_by: string;
}

export interface BrainFileWithMeta extends BrainFile {
	hashtags: string[];
	links: string[];
	score?: number;
}

export interface BrainLink {
	source_file_id: string;
	target_file_id: string;
}

export interface GraphCluster {
	id: number;
	color: string;
	label: string;
}

export interface GraphNode {
	file_id: string;
	title: string;
	x: number;
	y: number;
	cluster_id: number;
	hashtags: string[];
}

export interface BrainGraph {
	brain_id: string;
	clusters: GraphCluster[];
	nodes: GraphNode[];
	links: BrainLink[];
	computed_at: string;
}

export type DreamProposalStatus = 'pending' | 'approved' | 'rejected';

export interface DreamProposal {
	id: string;
	brain_id: string;
	snapshot?: string;
	changes: string;
	summary: string;
	status: DreamProposalStatus;
	created_at: string;
	resolved_at?: string;
	resolved_by?: string;
}

export interface DreamChange {
	action: 'create' | 'edit' | 'delete';
	file_id?: string;
	new_title?: string;
	new_content?: string;
	reason: string;
}
```

- [ ] **Step 2: Create the brain API client**

```typescript
// web/src/lib/api/brain.ts
import type {
	Brain,
	BrainFile,
	BrainFileWithMeta,
	BrainGraph,
	DreamProposal
} from './types';
import { getBaseUrl } from '$lib/platform';

function getToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem('forgebox_token');
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const base = getBaseUrl();
	const token = getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(token ? { Authorization: `Bearer ${token}` } : {})
	};

	const res = await fetch(`${base}${path}`, { headers, ...init });

	if (!res.ok) {
		const body = await res.json().catch(() => ({}));
		throw new Error(body.error || `HTTP ${res.status}`);
	}

	return res.json();
}

// --- Brain ---
export async function getBrain(agentId: string): Promise<Brain> {
	return request(`/agents/${agentId}/brain`);
}

// --- Brain Files ---
export async function listBrainFiles(agentId: string): Promise<BrainFile[]> {
	return (await request<BrainFile[] | null>(`/agents/${agentId}/brain/files`)) ?? [];
}

export async function createBrainFile(
	agentId: string,
	title: string,
	content: string
): Promise<BrainFile> {
	return request(`/agents/${agentId}/brain/files`, {
		method: 'POST',
		body: JSON.stringify({ title, content })
	});
}

export async function getBrainFile(agentId: string, fileId: string): Promise<BrainFile> {
	return request(`/agents/${agentId}/brain/files/${fileId}`);
}

export async function updateBrainFile(
	agentId: string,
	fileId: string,
	updates: { title?: string; content?: string }
): Promise<BrainFile> {
	return request(`/agents/${agentId}/brain/files/${fileId}`, {
		method: 'PUT',
		body: JSON.stringify(updates)
	});
}

export async function deleteBrainFile(
	agentId: string,
	fileId: string
): Promise<{ status: string }> {
	return request(`/agents/${agentId}/brain/files/${fileId}`, { method: 'DELETE' });
}

// --- Graph ---
export async function getBrainGraph(agentId: string): Promise<BrainGraph> {
	return request(`/agents/${agentId}/brain/graph`);
}

// --- Search ---
export async function searchBrainFiles(
	agentId: string,
	query: string,
	limit = 10
): Promise<BrainFileWithMeta[]> {
	return (
		(await request<BrainFileWithMeta[] | null>(`/agents/${agentId}/brain/search`, {
			method: 'POST',
			body: JSON.stringify({ query, limit })
		})) ?? []
	);
}

// --- Dreams ---
export async function listDreamProposals(agentId: string): Promise<DreamProposal[]> {
	return (await request<DreamProposal[] | null>(`/agents/${agentId}/brain/dreams`)) ?? [];
}

export async function getDreamProposal(
	agentId: string,
	dreamId: string
): Promise<DreamProposal> {
	return request(`/agents/${agentId}/brain/dreams/${dreamId}`);
}

export async function approveDream(
	agentId: string,
	dreamId: string
): Promise<{ status: string }> {
	return request(`/agents/${agentId}/brain/dreams/${dreamId}/approve`, { method: 'POST' });
}

export async function rejectDream(
	agentId: string,
	dreamId: string
): Promise<{ status: string }> {
	return request(`/agents/${agentId}/brain/dreams/${dreamId}/reject`, { method: 'POST' });
}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/api/types.ts web/src/lib/api/brain.ts
git commit -m "feat(web): add brain API types and client functions"
```

---

## Task 11: Frontend Brain Store

**Files:**
- Create: `web/src/lib/stores/brain.svelte.ts`

- [ ] **Step 1: Write the brain store**

```typescript
// web/src/lib/stores/brain.svelte.ts
import type { BrainFile, BrainGraph, DreamProposal, BrainFileWithMeta } from '$lib/api/types';
import * as api from '$lib/api/brain';

export let files = $state<BrainFile[]>([]);
export let graph = $state<BrainGraph | null>(null);
export let selectedFileId = $state<string | null>(null);
export let selectedFile = $state<BrainFile | null>(null);
export let searchResults = $state<BrainFileWithMeta[]>([]);
export let dreamProposals = $state<DreamProposal[]>([]);
export let loading = $state(false);

let currentAgentId = '';

export async function loadBrain(agentId: string) {
	currentAgentId = agentId;
	loading = true;
	try {
		const [fileList, graphData, dreams] = await Promise.all([
			api.listBrainFiles(agentId),
			api.getBrainGraph(agentId),
			api.listDreamProposals(agentId)
		]);
		files = fileList;
		graph = graphData;
		dreamProposals = dreams;
	} finally {
		loading = false;
	}
}

export async function selectFile(fileId: string) {
	selectedFileId = fileId;
	selectedFile = await api.getBrainFile(currentAgentId, fileId);
}

export function clearSelection() {
	selectedFileId = null;
	selectedFile = null;
}

export async function createFile(title: string, content: string) {
	const file = await api.createBrainFile(currentAgentId, title, content);
	files = [...files, file];
	selectedFileId = file.id;
	selectedFile = file;
	return file;
}

export async function updateFile(fileId: string, title: string, content: string) {
	const updated = await api.updateBrainFile(currentAgentId, fileId, { title, content });
	files = files.map((f) => (f.id === fileId ? updated : f));
	if (selectedFileId === fileId) {
		selectedFile = updated;
	}
	return updated;
}

export async function deleteFile(fileId: string) {
	await api.deleteBrainFile(currentAgentId, fileId);
	files = files.filter((f) => f.id !== fileId);
	if (selectedFileId === fileId) {
		clearSelection();
	}
}

export async function search(query: string) {
	if (!query.trim()) {
		searchResults = [];
		return;
	}
	searchResults = await api.searchBrainFiles(currentAgentId, query);
}

export async function approveDream(dreamId: string) {
	await api.approveDream(currentAgentId, dreamId);
	dreamProposals = dreamProposals.map((d) =>
		d.id === dreamId ? { ...d, status: 'approved' as const } : d
	);
	await loadBrain(currentAgentId);
}

export async function rejectDream(dreamId: string) {
	await api.rejectDream(currentAgentId, dreamId);
	dreamProposals = dreamProposals.map((d) =>
		d.id === dreamId ? { ...d, status: 'rejected' as const } : d
	);
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/lib/stores/brain.svelte.ts
git commit -m "feat(web): add brain Svelte store with runes-based state"
```

---

## Task 12: Frontend Brain Page and Components

**Files:**
- Install: `d3-force`, `@types/d3-force` npm packages
- Create: `web/src/lib/components/brain/BrainGraph.svelte`
- Create: `web/src/lib/components/brain/BrainEditor.svelte`
- Create: `web/src/lib/components/brain/BrainSearch.svelte`
- Create: `web/src/lib/components/brain/DreamPanel.svelte`
- Create: `web/src/lib/components/brain/BrainFileMeta.svelte`
- Create: `web/src/routes/(app)/agents/[id]/brain/+page.svelte`

This is the largest task. Due to the complexity of d3-force integration and the markdown editor, this task should be executed by the implementing agent reading the design spec and frontend patterns carefully. The key implementation notes:

- [ ] **Step 1: Install frontend dependencies**

Run: `cd /home/trystan/forgebox/web && npm install d3-force @types/d3-force @tiptap/core @tiptap/starter-kit @tiptap/extension-link @tiptap/extension-placeholder`

- [ ] **Step 2: Create BrainGraph.svelte**

The graph component uses d3-force for physics simulation rendered onto an SVG element. Key implementation:
- Accept `graph: BrainGraph` and `selectedFileId: string | null` as props
- Create a d3 force simulation with `forceLink`, `forceManyBody`, `forceCenter`, `forceCollide`
- Render circles colored by cluster, lines for links
- Handle click (dispatch `select` event with file_id), hover (show tooltip)
- Use `$effect` to restart simulation when graph data changes
- Support zoom/pan via d3-zoom or manual SVG viewBox transform
- Highlight selected node with a ring
- Highlight search results with a pulsing animation

- [ ] **Step 3: Create BrainEditor.svelte**

The editor uses tiptap for rich markdown editing. Key implementation:
- Accept `file: BrainFile`, `allFiles: BrainFile[]`, `allHashtags: string[]` as props
- Initialize tiptap editor with StarterKit + Link extension
- Add custom `[[` input rule that triggers a suggestion popup listing file titles
- Add custom `#` input rule that triggers a suggestion popup listing hashtags
- Toolbar with bold, italic, h1-h3, bullet list, code block buttons
- Save button calls `updateFile` from the brain store
- Delete button with confirmation

- [ ] **Step 4: Create BrainFileMeta.svelte**

Simple metadata bar above the editor:
- Editable title input
- "Created by" badge (agent or user)
- Timestamps in relative format (via date-fns `formatDistanceToNow`)
- Hashtag pills (styled as badges)

- [ ] **Step 5: Create BrainSearch.svelte**

Search input component:
- Text input that calls `search()` from the brain store on input (debounced 300ms)
- Dropdown list of results below the input showing title + score + snippet
- Clicking a result calls `selectFile`
- Dispatch event to highlight matching nodes in graph

- [ ] **Step 6: Create DreamPanel.svelte**

Modal/slide panel for dream proposals:
- List of proposals with status badges (pending amber, approved green, rejected red)
- Clicking a pending proposal shows summary + changes list
- Each change shows action (create/edit/delete), file title, and reason
- Approve and Reject buttons
- "Dream Now" button (calls trigger endpoint, to be added later)

- [ ] **Step 7: Create the brain page**

`web/src/routes/(app)/agents/[id]/brain/+page.svelte`:
- Split layout: left 60% (graph), right 40% (editor)
- Top bar with "New File" button, search input, "Dreams" button
- On mount: call `loadBrain(agentId)`
- Wire component events together (graph click -> select file -> open editor)
- Empty state when no files exist

- [ ] **Step 8: Commit**

```bash
git add web/src/lib/components/brain/ web/src/routes/\(app\)/agents/\[id\]/brain/ web/package.json web/package-lock.json
git commit -m "feat(web): add brain page with graph visualization and markdown editor"
```

---

## Task 13: Navigation Integration

**Files:**
- Modify: `web/src/routes/(app)/agents/[id]/+page.svelte`

- [ ] **Step 1: Add Brain link to agent detail page**

Add a "Brain" button or tab link on the agent detail page that navigates to `/agents/{id}/brain`. Style it as a secondary button or a tab in the page header.

```svelte
<a href="/agents/{agentId}/brain" class="btn-secondary">
  Brain
</a>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/routes/\(app\)/agents/\[id\]/+page.svelte
git commit -m "feat(web): add brain navigation link to agent detail page"
```

---

## Task 14: Clusterer (Graph Precomputation)

**Files:**
- Create: `internal/brain/clusterer.go`
- Create: `internal/brain/clusterer_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/brain/clusterer_test.go
package brain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignClusters(t *testing.T) {
	// 4 vectors forming 2 clear clusters.
	embeddings := [][]float32{
		{1.0, 0.0, 0.0, 0.0}, // cluster A
		{0.9, 0.1, 0.0, 0.0}, // cluster A
		{0.0, 0.0, 1.0, 0.0}, // cluster B
		{0.0, 0.1, 0.9, 0.0}, // cluster B
	}

	clusters := AssignClusters(embeddings, 2)
	require.Len(t, clusters, 4)

	// Vectors 0 and 1 should be in the same cluster.
	assert.Equal(t, clusters[0], clusters[1])
	// Vectors 2 and 3 should be in the same cluster.
	assert.Equal(t, clusters[2], clusters[3])
	// The two clusters should be different.
	assert.NotEqual(t, clusters[0], clusters[2])
}

func TestProject2D(t *testing.T) {
	embeddings := [][]float32{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0},
	}
	points := Project2D(embeddings)
	require.Len(t, points, 3)
	for _, p := range points {
		assert.Len(t, p, 2)
	}
}

func TestAssignClusters_TooFewPoints(t *testing.T) {
	// Fewer than k points — should assign all to cluster 0.
	embeddings := [][]float32{
		{1.0, 0.0},
	}
	clusters := AssignClusters(embeddings, 3)
	assert.Equal(t, []int{0}, clusters)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/trystan/forgebox && go test ./internal/brain/ -v -run "TestAssign|TestProject"`
Expected: FAIL — functions undefined

- [ ] **Step 3: Implement the clusterer**

```go
// internal/brain/clusterer.go
package brain

import (
	"math"
	"math/rand"
)

// AssignClusters runs simple k-means on embedding vectors and returns
// the cluster index for each vector. If len(embeddings) < k, all get cluster 0.
func AssignClusters(embeddings [][]float32, k int) []int {
	n := len(embeddings)
	if n == 0 {
		return nil
	}
	if n <= k {
		assignments := make([]int, n)
		for i := range assignments {
			assignments[i] = 0
		}
		return assignments
	}

	dim := len(embeddings[0])
	assignments := make([]int, n)

	// Initialize centroids using k random points.
	rng := rand.New(rand.NewSource(42))
	centroids := make([][]float32, k)
	perm := rng.Perm(n)
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, dim)
		copy(centroids[i], embeddings[perm[i]])
	}

	for iter := 0; iter < 50; iter++ {
		// Assign each point to nearest centroid.
		changed := false
		for i, vec := range embeddings {
			best := 0
			bestDist := float32(math.MaxFloat32)
			for c, centroid := range centroids {
				d := euclideanDist(vec, centroid)
				if d < bestDist {
					bestDist = d
					best = c
				}
			}
			if assignments[i] != best {
				assignments[i] = best
				changed = true
			}
		}

		if !changed {
			break
		}

		// Recompute centroids.
		counts := make([]int, k)
		newCentroids := make([][]float32, k)
		for i := range newCentroids {
			newCentroids[i] = make([]float32, dim)
		}
		for i, vec := range embeddings {
			c := assignments[i]
			counts[c]++
			for d := range vec {
				newCentroids[c][d] += vec[d]
			}
		}
		for c := range centroids {
			if counts[c] > 0 {
				for d := range centroids[c] {
					centroids[c][d] = newCentroids[c][d] / float32(counts[c])
				}
			}
		}
	}

	return assignments
}

// Project2D projects high-dimensional embeddings to 2D using PCA (first 2 components).
// Returns an array of [x, y] pairs normalized to a [0, 1000] range.
func Project2D(embeddings [][]float32) [][2]float64 {
	n := len(embeddings)
	if n == 0 {
		return nil
	}

	dim := len(embeddings[0])
	if dim <= 2 {
		points := make([][2]float64, n)
		for i, vec := range embeddings {
			points[i] = [2]float64{float64(vec[0]), 0}
			if len(vec) > 1 {
				points[i][1] = float64(vec[1])
			}
		}
		return normalizePoints(points)
	}

	// Simple PCA: center data, use first two dimensions of the covariance.
	// For a production system, use SVD. This is a pragmatic approximation
	// that works well for visualization — the force simulation will refine positions.
	mean := make([]float64, dim)
	for _, vec := range embeddings {
		for d, v := range vec {
			mean[d] += float64(v) / float64(n)
		}
	}

	// Project onto the two dimensions with highest variance.
	variance := make([]float64, dim)
	for _, vec := range embeddings {
		for d, v := range vec {
			diff := float64(v) - mean[d]
			variance[d] += diff * diff
		}
	}

	// Find top 2 dimensions by variance.
	d1, d2 := 0, 1
	if variance[d2] > variance[d1] {
		d1, d2 = d2, d1
	}
	for d := 2; d < dim; d++ {
		if variance[d] > variance[d1] {
			d2 = d1
			d1 = d
		} else if variance[d] > variance[d2] {
			d2 = d
		}
	}

	points := make([][2]float64, n)
	for i, vec := range embeddings {
		points[i] = [2]float64{
			float64(vec[d1]) - mean[d1],
			float64(vec[d2]) - mean[d2],
		}
	}

	return normalizePoints(points)
}

func normalizePoints(points [][2]float64) [][2]float64 {
	if len(points) == 0 {
		return points
	}
	minX, maxX := points[0][0], points[0][0]
	minY, maxY := points[0][1], points[0][1]
	for _, p := range points {
		if p[0] < minX { minX = p[0] }
		if p[0] > maxX { maxX = p[0] }
		if p[1] < minY { minY = p[1] }
		if p[1] > maxY { maxY = p[1] }
	}
	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 { rangeX = 1 }
	if rangeY == 0 { rangeY = 1 }

	for i := range points {
		points[i][0] = ((points[i][0] - minX) / rangeX) * 800 + 100
		points[i][1] = ((points[i][1] - minY) / rangeY) * 600 + 100
	}
	return points
}

func euclideanDist(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return float32(math.Sqrt(float64(sum)))
}

// ClusterColors is a palette of distinct colors for graph clusters.
var ClusterColors = []string{
	"#6366f1", // indigo
	"#10b981", // emerald
	"#f59e0b", // amber
	"#ef4444", // red
	"#8b5cf6", // violet
	"#06b6d4", // cyan
	"#ec4899", // pink
	"#84cc16", // lime
	"#f97316", // orange
	"#14b8a6", // teal
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/trystan/forgebox && go test ./internal/brain/ -v -run "TestAssign|TestProject"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/brain/clusterer.go internal/brain/clusterer_test.go
git commit -m "feat(brain): add k-means clusterer and 2D projection for graph layout"
```

---

## Task 15: Dream Cycle

**Files:**
- Create: `internal/brain/dreamer.go`

- [ ] **Step 1: Write the dreamer**

```go
// internal/brain/dreamer.go
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// Dreamer runs the dream cycle for brain consolidation.
type Dreamer struct {
	store    sdk.BrainStore
	provider sdk.ProviderPlugin
	model    string
}

// NewDreamer creates a new dreamer.
func NewDreamer(store sdk.BrainStore, provider sdk.ProviderPlugin, model string) *Dreamer {
	return &Dreamer{
		store:    store,
		provider: provider,
		model:    model,
	}
}

// Dream runs the consolidation cycle for a single brain.
func (d *Dreamer) Dream(ctx context.Context, brainID string) (*sdk.DreamProposal, error) {
	files, err := d.store.ListFiles(ctx, brainID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	if len(files) < 2 {
		slog.Info("brain too small for dreaming", "brain_id", brainID, "file_count", len(files))
		return nil, nil
	}

	// Build snapshot.
	snapshotData, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	// Build the consolidation prompt.
	var fileList string
	for _, f := range files {
		tags, _ := d.store.GetFileHashtags(ctx, f.ID)
		tagStr := ""
		if len(tags) > 0 {
			tagStr = " [tags: " + fmt.Sprintf("%v", tags) + "]"
		}
		fileList += fmt.Sprintf("### %s (id: %s)%s\n%s\n\n", f.Title, f.ID, tagStr, f.Content)
	}

	prompt := fmt.Sprintf(`You are reviewing a knowledge base containing %d files. Your job is to consolidate, deduplicate, and reorganize this memory.

Review these files and propose changes:

%s

Return a JSON array of changes. Each change has: action ("create", "edit", or "delete"), file_id (for edit/delete), new_title (for create/edit), new_content (for create/edit), and reason.

Only propose changes that meaningfully improve the knowledge base. Do not change files that are already well-organized.

Respond with ONLY valid JSON — no markdown fences, no explanation.`, len(files), fileList)

	resp, err := d.provider.Complete(ctx, &sdk.CompletionRequest{
		Model: d.model,
		Messages: []sdk.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 4096,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}

	// Validate the response is valid JSON.
	var changes []sdk.DreamChange
	if err := json.Unmarshal([]byte(resp.Content), &changes); err != nil {
		return nil, fmt.Errorf("parse LLM response as changes: %w", err)
	}

	if len(changes) == 0 {
		slog.Info("dream produced no changes", "brain_id", brainID)
		return nil, nil
	}

	changesJSON, _ := json.Marshal(changes)

	// Build summary.
	summary := fmt.Sprintf("Proposed %d changes: ", len(changes))
	creates, edits, deletes := 0, 0, 0
	for _, c := range changes {
		switch c.Action {
		case "create":
			creates++
		case "edit":
			edits++
		case "delete":
			deletes++
		}
	}
	if creates > 0 {
		summary += fmt.Sprintf("%d new files, ", creates)
	}
	if edits > 0 {
		summary += fmt.Sprintf("%d edits, ", edits)
	}
	if deletes > 0 {
		summary += fmt.Sprintf("%d deletions, ", deletes)
	}

	proposal := &sdk.DreamProposal{
		ID:        uuid.New().String(),
		BrainID:   brainID,
		Snapshot:  string(snapshotData),
		Changes:   string(changesJSON),
		Summary:   summary,
		Status:    sdk.DreamPending,
		CreatedAt: time.Now(),
	}

	if err := d.store.CreateDreamProposal(ctx, proposal); err != nil {
		return nil, fmt.Errorf("save dream proposal: %w", err)
	}

	slog.Info("dream proposal created", "brain_id", brainID, "proposal_id", proposal.ID, "changes", len(changes))
	return proposal, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/trystan/forgebox && go build ./internal/brain/...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/brain/dreamer.go
git commit -m "feat(brain): add dream cycle for memory consolidation"
```

---

## Summary

This plan covers 15 tasks across the full stack:

| # | Task | Layer |
|---|------|-------|
| 1 | SDK types + BrainStore interface | Go types |
| 2 | Markdown parser | Go logic |
| 3 | Embedder interface | Go logic |
| 4 | PostgreSQL connection + migrations | Go storage |
| 5 | BrainStore PostgreSQL implementation | Go storage |
| 6 | Brain service | Go service |
| 7 | Brain API handlers | Go API |
| 8 | Brain tool plugin | Go plugin |
| 9 | Startup wiring | Go main |
| 10 | Frontend types + API client | TS/Svelte |
| 11 | Frontend brain store | TS/Svelte |
| 12 | Frontend brain page + components | TS/Svelte |
| 13 | Navigation integration | TS/Svelte |
| 14 | Clusterer | Go logic |
| 15 | Dream cycle | Go logic |

Tasks 1-9 are backend (Go), 10-13 are frontend (Svelte), 14-15 are backend logic. Tasks 1-9 should be done first since the frontend depends on the API being available. Tasks 14-15 can be done after the core CRUD is working.
