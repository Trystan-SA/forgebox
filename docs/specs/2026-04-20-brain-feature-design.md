# Brain Feature -- Design Spec

**Date:** 2026-04-20
**Status:** Approved
**Approach:** Precomputed Graph (Approach B)

## Overview

Brain is an Obsidian-like markdown knowledge base for ForgeBox agents. Each agent has
its own brain -- a collection of `.md` files linked via `[[double bracket]]` references
and indexed with `#hashtags`. Agents interact with their brain through an explicit tool
during task execution, using RAG (pgvector) for semantic search. The brain is visualized
as a force-directed graph of colored dots, with colors derived from embedding-based
semantic clustering.

A daily "dream" cycle allows the agent to consolidate, deduplicate, and reorganize its
memory. Dream proposals require user approval before being applied.

## Requirements Summary

- One brain per agent (1:1 with automations)
- Both agents and humans can create/edit/delete brain files
- `[[file-name]]` links between brain files
- `#hashtags` for keyword indexing
- Agent accesses brain via explicit `brain` tool calls (not automatic injection)
- RAG powered by pgvector in existing PostgreSQL
- Embeddings via configured LLM provider API, overridable per-brain in settings
- View-only force-directed graph visualization; click dot to open file
- Dot colors from semantic clustering of embeddings
- No file size limits
- Daily dream cycle with user-approved proposals
- Rich markdown editor with `[[link]]` and `#hashtag` autocomplete

---

## Data Model

### `brains`

One per agent.

| Column             | Type      | Notes                                      |
|--------------------|-----------|--------------------------------------------|
| id                 | UUID      | PK                                         |
| automation_id      | UUID      | FK to automations, unique (1:1)            |
| embedding_provider | TEXT      | nullable, override for embedding endpoint  |
| embedding_model    | TEXT      | nullable, override for embedding model     |
| embedding_dimension| INT       | vector dimension, set on first embed (e.g., 1536) |
| created_at         | TIMESTAMP |                                            |
| updated_at         | TIMESTAMP |                                            |

### `brain_files`

Markdown documents within a brain.

| Column     | Type         | Notes                                          |
|------------|--------------|-------------------------------------------------|
| id         | UUID         | PK                                             |
| brain_id   | UUID         | FK to brains                                   |
| title      | TEXT         | filename / display name                        |
| content    | TEXT         | full markdown content                          |
| embedding  | vector       | pgvector column, dimension set at brain creation based on provider model (e.g., 1536 for OpenAI text-embedding-3-small). Re-embedded on content change |
| cluster_id | INT          | nullable, assigned by clustering job           |
| created_at | TIMESTAMP    |                                                |
| updated_at | TIMESTAMP    |                                                |
| created_by | TEXT         | "agent" or user UUID                           |

### `brain_links`

Extracted `[[links]]` between files.

| Column         | Type | Notes            |
|----------------|------|------------------|
| source_file_id | UUID | FK to brain_files|
| target_file_id | UUID | FK to brain_files|

Composite PK: (source_file_id, target_file_id).

### `brain_hashtags`

Extracted `#hashtags` from file content.

| Column  | Type | Notes                             |
|---------|------|-----------------------------------|
| file_id | UUID | FK to brain_files                 |
| tag     | TEXT | normalized hashtag (lowercase, no #) |

Composite PK: (file_id, tag).

### `brain_graph`

Precomputed visualization data.

| Column      | Type      | Notes                                    |
|-------------|-----------|------------------------------------------|
| brain_id    | UUID      | FK to brains, unique                     |
| clusters    | JSONB     | array of {id, color, label}              |
| nodes       | JSONB     | array of {file_id, x, y, cluster_id}    |
| computed_at | TIMESTAMP |                                          |

### `dream_proposals`

Pending brain reorganizations from the dream cycle.

| Column      | Type      | Notes                                          |
|-------------|-----------|-------------------------------------------------|
| id          | UUID      | PK                                             |
| brain_id    | UUID      | FK to brains                                   |
| snapshot    | JSONB     | full brain state before changes                |
| changes     | JSONB     | array of proposed mutations                    |
| summary     | TEXT      | LLM-generated description of changes           |
| status      | TEXT      | "pending", "approved", "rejected"              |
| created_at  | TIMESTAMP |                                                |
| resolved_at | TIMESTAMP | nullable                                       |
| resolved_by | UUID      | nullable, FK to users                          |

---

## Backend Architecture

### New Package: `internal/brain/`

**`service.go`** -- BrainService

Core orchestrator for brain operations. Handles CRUD for brains and files. On every
file write/update: parses content to extract links and hashtags, triggers embedding
computation, queues graph recomputation (debounced).

**`parser.go`** -- Markdown Parser

Extracts `[[links]]` and `#hashtags` from markdown content via regex.

- `[[link]]` pattern: `\[\[([^\]]+)\]\]` -- extracts the inner text, resolves to
  a brain_file by title match within the same brain.
- `#hashtag` pattern: `(?:^|\s)#([a-zA-Z0-9_-]+)` -- extracts tag text, normalizes
  to lowercase.

**`embedder.go`** -- Embedding Service

Interface:
```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}
```

Resolution order for which provider/model to use:
1. Brain-level override (embedding_provider + embedding_model fields)
2. System-level embedding setting (configurable in settings)
3. First available provider that supports embeddings

**`clusterer.go`** -- Graph Clustering

Runs on the set of file embeddings for a brain:
1. K-means or DBSCAN clustering on the embedding vectors to assign cluster IDs
2. Dimensionality reduction (t-SNE or UMAP) to project embeddings to 2D coordinates
3. Auto-generates cluster labels from the most frequent hashtags in each cluster
4. Assigns colors from a predefined palette (one color per cluster)
5. Writes results to `brain_graph` table

Triggered by file mutations (debounced -- waits for a quiet period before recomputing).

**`dreamer.go`** -- Dream Cycle

Process:
1. Check if brain has been modified since last dream (skip if not)
2. Load all brain files
3. Create snapshot (serialize full brain state to JSONB)
4. Send to the agent's LLM with consolidation prompt:
   - "Review your memory files. Identify duplicates, merge related content, remove
     outdated information, improve organization. Return structured changes."
5. Parse LLM response into structured changes
6. Create `dream_proposal` record with status "pending"

Dream proposal `changes` format:
```json
[
  {
    "action": "edit",
    "file_id": "uuid",
    "new_title": "...",
    "new_content": "...",
    "reason": "Merged with duplicate file X"
  },
  {
    "action": "delete",
    "file_id": "uuid",
    "reason": "Outdated, superseded by file Y"
  },
  {
    "action": "create",
    "title": "...",
    "content": "...",
    "reason": "Consolidated 3 fragments into one document"
  }
]
```

Schedule: configurable per-brain, defaults to daily at 02:00 server time. Can be
triggered manually from the UI.

### Storage Implementation: `internal/storage/postgres/brain.go`

Implements the BrainStore interface:

```go
type BrainStore interface {
    CreateBrain(ctx context.Context, brain *Brain) error
    GetBrain(ctx context.Context, brainID uuid.UUID) (*Brain, error)
    GetBrainByAutomation(ctx context.Context, automationID uuid.UUID) (*Brain, error)

    CreateFile(ctx context.Context, file *BrainFile) error
    UpdateFile(ctx context.Context, file *BrainFile) error
    DeleteFile(ctx context.Context, fileID uuid.UUID) error
    GetFile(ctx context.Context, fileID uuid.UUID) (*BrainFile, error)
    ListFiles(ctx context.Context, brainID uuid.UUID) ([]*BrainFile, error)
    SearchByEmbedding(ctx context.Context, brainID uuid.UUID, vec []float32, limit int) ([]*BrainFile, error)

    SaveGraph(ctx context.Context, graph *BrainGraph) error
    GetGraph(ctx context.Context, brainID uuid.UUID) (*BrainGraph, error)

    CreateDreamProposal(ctx context.Context, proposal *DreamProposal) error
    GetDreamProposal(ctx context.Context, proposalID uuid.UUID) (*DreamProposal, error)
    ListDreamProposals(ctx context.Context, brainID uuid.UUID) ([]*DreamProposal, error)
    UpdateDreamProposalStatus(ctx context.Context, proposalID uuid.UUID, status string, resolvedBy uuid.UUID) error
}
```

Similarity search uses pgvector's cosine distance operator:
`SELECT * FROM brain_files WHERE brain_id = $1 ORDER BY embedding <=> $2 LIMIT $3`

### Tool Plugin: `plugins/tools/brain/`

Registered as an agent-callable tool. The agent does not get automatic brain injection --
it explicitly calls this tool when it needs context.

Schema:
```json
{
  "name": "brain",
  "description": "Search, read, and write to your persistent memory",
  "input_schema": {
    "type": "object",
    "properties": {
      "action": {
        "type": "string",
        "enum": ["search", "read", "write", "list", "delete"]
      },
      "query": {
        "type": "string",
        "description": "Semantic search query (required for search action)"
      },
      "file_id": {
        "type": "string",
        "description": "Target file ID (required for read and delete)"
      },
      "title": {
        "type": "string",
        "description": "File title (required for write)"
      },
      "content": {
        "type": "string",
        "description": "Markdown content (required for write)"
      }
    },
    "required": ["action"]
  }
}
```

Actions:
- **search** -- Embed query, run pgvector similarity search, return top-N files with
  title, relevance score, and content snippet
- **read** -- Return full content of a specific file by ID
- **write** -- Create or update a brain file. If file_id is provided, updates existing.
  Otherwise creates new. Triggers link/hashtag parsing, embedding, graph recompute
- **list** -- Return all file titles, IDs, hashtags, and timestamps
- **delete** -- Remove a file and its associated links/hashtags

### API Routes

Added to gateway under the agent (automation) resource:

```
GET    /agents/{id}/brain                        → getBrain
GET    /agents/{id}/brain/files                  → listBrainFiles
POST   /agents/{id}/brain/files                  → createBrainFile
GET    /agents/{id}/brain/files/{fid}            → getBrainFile
PUT    /agents/{id}/brain/files/{fid}            → updateBrainFile
DELETE /agents/{id}/brain/files/{fid}            → deleteBrainFile
GET    /agents/{id}/brain/graph                  → getBrainGraph
POST   /agents/{id}/brain/search                 → searchBrainFiles
GET    /agents/{id}/brain/dreams                 → listDreamProposals
GET    /agents/{id}/brain/dreams/{did}           → getDreamProposal
POST   /agents/{id}/brain/dreams/{did}/approve   → approveDream
POST   /agents/{id}/brain/dreams/{did}/reject    → rejectDream
POST   /agents/{id}/brain/dreams/trigger         → triggerDreamManually
```

### Background Jobs

Two background jobs managed by the scheduler:

1. **Graph recomputation** -- Debounced. After any brain file mutation, waits 5 seconds
   of inactivity then recomputes clustering and 2D layout for the affected brain.

2. **Dream cycle** -- Cron-scheduled. Default: daily at 02:00. Iterates over all brains
   with modifications since last dream, runs the dreamer for each.

---

## Frontend Architecture

### Route: `(app)/agents/[id]/brain/+page.svelte`

Split layout with graph on the left and editor on the right.

### Components

**`BrainGraph.svelte`** -- Force-Directed Graph (Left Panel, 60% width)

- Renders using d3-force (or Svelte-compatible force simulation library)
- Data source: precomputed `brain_graph` API response (nodes with x/y/cluster_id,
  clusters with color/label, links from `brain_links`)
- Force simulation: linked nodes attract, unlinked nodes repel, cluster gravity
- Dots are uniformly sized circles, colored by cluster
- Lines between dots represent `[[links]]`
- Hover: tooltip with file title and hashtags
- Click: selects the file, opens in editor panel
- Zoom and pan via mouse/touch
- Legend in corner showing cluster colors and labels
- Selected dot has a highlight ring

**`BrainEditor.svelte`** -- Rich Markdown Editor (Right Panel, 40% width)

- Built on a markdown editor library (tiptap with markdown extension, or milkdown)
- Toolbar: bold, italic, h1-h3, bullet list, numbered list, code block, link
- `[[` keystroke triggers autocomplete dropdown listing brain file titles
- `#` keystroke triggers autocomplete dropdown listing existing hashtags
- Live rendered preview below the editor (toggleable)
- Save button: calls PUT endpoint, triggers re-embedding and graph recompute
- Delete button: confirmation dialog, calls DELETE endpoint

**`BrainFileMeta.svelte`** -- File Metadata Bar

- Displayed above the editor when a file is selected
- Editable title field
- Created by badge ("agent" or user name)
- Created/updated timestamps
- Hashtag pills (clickable -- highlights related dots in graph)

**`BrainSearch.svelte`** -- Semantic Search

- Search input in the top bar area
- Calls `POST /agents/{id}/brain/search` with the query
- Results: highlights matching dots in the graph (pulsing animation)
- Dropdown list of results below the search bar with titles and snippets
- Clicking a result selects the file and opens it in the editor

**`DreamPanel.svelte`** -- Dream Proposals

- Opened via "Dreams" button in the top bar
- Slides in as a panel or modal
- Lists dream proposals with status badges (pending/approved/rejected)
- Clicking a pending proposal shows:
  - LLM-generated summary of changes
  - Per-file diffs (created, edited with before/after, deleted)
  - Approve and Reject buttons
- "Dream Now" button to manually trigger a dream cycle

### State Management

**`brainStore.ts`** -- Svelte writable store

Holds:
- `brain` -- brain metadata (id, settings)
- `files` -- array of brain file summaries (id, title, hashtags, cluster_id)
- `graph` -- precomputed graph data (nodes, clusters, edges)
- `selectedFileId` -- currently open file
- `selectedFileContent` -- full content of selected file (loaded on demand)
- `searchResults` -- array of file IDs matching current search
- `dreamProposals` -- array of dream proposals

Fetches brain + files + graph on mount. Updates optimistically on file edits.

---

## Embedding Settings

### System-Level Setting

A new "Embedding" section in the Settings page:
- Provider dropdown (from configured providers that support embeddings)
- Model text input (e.g., `text-embedding-3-small`)
- These serve as the default for all brains

### Per-Brain Override

On the brain page, a settings gear icon opens a small config panel:
- Override provider and model for this specific brain
- "Use system default" option to clear overrides

### Resolution Order

1. Brain-level `embedding_provider` + `embedding_model` (if set)
2. System-level embedding settings (if configured)
3. First available provider that supports embeddings

---

## Error Handling

- **Embedding failure** -- If the embedding API call fails, the file is still saved
  but `embedding` is set to NULL. A warning indicator shows on the file in the UI.
  Files with NULL embeddings are excluded from RAG search but still visible in the
  graph (placed at a default position). Embedding is retried on next file update.

- **Clustering with few files** -- If a brain has fewer than 3 files, skip clustering.
  All dots get a single default color. Clustering kicks in at 3+ files.

- **Dream LLM failure** -- If the dream cycle's LLM call fails, log the error and
  skip. No dream proposal is created. Retry on next scheduled run.

- **Broken links** -- If a `[[link]]` references a file title that doesn't exist,
  the link is stored but rendered as a dashed line in the graph (or hidden). The
  editor shows the link text in a warning color.

---

## Testing Strategy

- **Unit tests**: parser (link/hashtag extraction), clusterer (cluster assignment),
  dreamer (change parsing), embedder (provider resolution)
- **Integration tests**: BrainStore against real PostgreSQL with pgvector, full
  write-parse-embed-cluster pipeline
- **E2E tests**: Create brain file via API, verify search returns it, verify graph
  includes it
- **Frontend**: Component tests for BrainGraph (renders nodes/edges), BrainEditor
  (autocomplete triggers), DreamPanel (approve/reject flow)
