package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/forgebox/forgebox/pkg/sdk"
)

// Verify Store implements BrainStore (and StoragePlugin via storage.go).
var _ sdk.BrainStore = (*Store)(nil)
var _ sdk.StoragePlugin = (*Store)(nil)

// CreateBrain persists a new brain record.
func (b *Store) CreateBrain(ctx context.Context, brain *sdk.BrainRecord) error {
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO brains (id, automation_id, embedding_provider, embedding_model, embedding_dimension, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		brain.ID, brain.AutomationID, brain.EmbeddingProvider, brain.EmbeddingModel,
		brain.EmbeddingDimension, brain.CreatedAt, brain.UpdatedAt,
	)
	return err
}

// GetBrain retrieves a brain by ID.
func (b *Store) GetBrain(ctx context.Context, id string) (*sdk.BrainRecord, error) {
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

// GetBrainByAutomation retrieves the brain associated with an automation.
func (b *Store) GetBrainByAutomation(ctx context.Context, automationID string) (*sdk.BrainRecord, error) {
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

// UpdateBrain updates embedding configuration for a brain.
func (b *Store) UpdateBrain(ctx context.Context, brain *sdk.BrainRecord) error {
	brain.UpdatedAt = time.Now()
	_, err := b.db.ExecContext(ctx,
		`UPDATE brains SET embedding_provider=$1, embedding_model=$2, embedding_dimension=$3, updated_at=$4 WHERE id=$5`,
		brain.EmbeddingProvider, brain.EmbeddingModel, brain.EmbeddingDimension, brain.UpdatedAt, brain.ID,
	)
	return err
}

// DeleteBrain removes a brain and all its files.
func (b *Store) DeleteBrain(ctx context.Context, id string) error {
	_, err := b.db.ExecContext(ctx, `DELETE FROM brains WHERE id = $1`, id)
	return err
}

// CreateFile inserts a new brain file with optional embedding.
func (b *Store) CreateFile(ctx context.Context, file *sdk.BrainFile) error {
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

// UpdateFile updates the content and embedding of a brain file.
func (b *Store) UpdateFile(ctx context.Context, file *sdk.BrainFile) error {
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

// DeleteFile soft-deletes a brain file by setting deleted_at.
func (b *Store) DeleteFile(ctx context.Context, fileID string) error {
	_, err := b.db.ExecContext(ctx,
		`UPDATE brain_files SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		fileID,
	)
	return err
}

// HardDeleteFile permanently removes a brain file row.
func (b *Store) HardDeleteFile(ctx context.Context, fileID string) error {
	_, err := b.db.ExecContext(ctx, `DELETE FROM brain_files WHERE id = $1`, fileID)
	return err
}

// ListExpiredArchivedFiles returns soft-deleted files whose deleted_at is before the given time.
func (b *Store) ListExpiredArchivedFiles(ctx context.Context, before time.Time) ([]*sdk.BrainFile, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, brain_id, title, content, cluster_id, created_at, updated_at, created_by
		 FROM brain_files
		 WHERE deleted_at IS NOT NULL AND deleted_at < $1
		 ORDER BY deleted_at`, before,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// GetFile retrieves a single brain file by ID.
func (b *Store) GetFile(ctx context.Context, fileID string) (*sdk.BrainFile, error) {
	var rec sdk.BrainFile
	err := b.db.QueryRowContext(ctx,
		`SELECT id, brain_id, title, content, cluster_id, created_at, updated_at, created_by
		 FROM brain_files WHERE id = $1 AND deleted_at IS NULL`, fileID,
	).Scan(&rec.ID, &rec.BrainID, &rec.Title, &rec.Content,
		&rec.ClusterID, &rec.CreatedAt, &rec.UpdatedAt, &rec.CreatedBy)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// ListFiles returns all active (non-deleted) files for a brain.
func (b *Store) ListFiles(ctx context.Context, brainID string) ([]*sdk.BrainFile, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, brain_id, title, content, cluster_id, created_at, updated_at, created_by
		 FROM brain_files WHERE brain_id = $1 AND deleted_at IS NULL ORDER BY created_at`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// SearchByEmbedding returns the nearest brain files to the given embedding vector.
func (b *Store) SearchByEmbedding(ctx context.Context, brainID string, vec []float32, limit int) ([]*sdk.BrainFileWithMeta, error) {
	v := pgvector.NewVector(vec)
	rows, err := b.db.QueryContext(ctx,
		`SELECT f.id, f.brain_id, f.title, f.content, f.cluster_id, f.created_at, f.updated_at, f.created_by,
		        1 - (f.embedding <=> $1) AS score
		 FROM brain_files f
		 WHERE f.brain_id = $2 AND f.embedding IS NOT NULL AND f.deleted_at IS NULL
		 ORDER BY f.embedding <=> $1
		 LIMIT $3`,
		v.String(), brainID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// SetFileHashtags replaces all hashtags for a brain file atomically.
func (b *Store) SetFileHashtags(ctx context.Context, fileID string, tags []string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

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

// GetFileHashtags returns all hashtags for a brain file.
func (b *Store) GetFileHashtags(ctx context.Context, fileID string) ([]string, error) {
	rows, err := b.db.QueryContext(ctx, `SELECT tag FROM brain_hashtags WHERE file_id = $1 ORDER BY tag`, fileID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// ListHashtags returns all distinct hashtags used across a brain's files.
func (b *Store) ListHashtags(ctx context.Context, brainID string) ([]string, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT DISTINCT h.tag FROM brain_hashtags h
		 JOIN brain_files f ON f.id = h.file_id
		 WHERE f.brain_id = $1 AND f.deleted_at IS NULL
		 ORDER BY h.tag`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// SetFileLinks replaces all outbound links for a source file atomically.
func (b *Store) SetFileLinks(ctx context.Context, sourceFileID string, targetFileIDs []string) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

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

// GetFileLinks returns all active file links within a brain.
func (b *Store) GetFileLinks(ctx context.Context, brainID string) ([]sdk.BrainLink, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT l.source_file_id, l.target_file_id FROM brain_links l
		 JOIN brain_files f ON f.id = l.source_file_id
		 JOIN brain_files t ON t.id = l.target_file_id
		 WHERE f.brain_id = $1 AND f.deleted_at IS NULL AND t.deleted_at IS NULL`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// SaveGraph upserts the precomputed graph for a brain.
func (b *Store) SaveGraph(ctx context.Context, graph *sdk.BrainGraph) error {
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

// GetGraph retrieves the precomputed graph for a brain.
func (b *Store) GetGraph(ctx context.Context, brainID string) (*sdk.BrainGraph, error) {
	var graph sdk.BrainGraph
	var clustersJSON, nodesJSON string
	err := b.db.QueryRowContext(ctx,
		`SELECT brain_id, clusters, nodes, computed_at FROM brain_graph WHERE brain_id = $1`, brainID,
	).Scan(&graph.BrainID, &clustersJSON, &nodesJSON, &graph.ComputedAt)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal([]byte(clustersJSON), &graph.Clusters); err != nil {
		return nil, fmt.Errorf("unmarshal clusters: %w", err)
	}
	if err = json.Unmarshal([]byte(nodesJSON), &graph.Nodes); err != nil {
		return nil, fmt.Errorf("unmarshal nodes: %w", err)
	}

	links, err := b.GetFileLinks(ctx, brainID)
	if err != nil {
		return nil, fmt.Errorf("get links: %w", err)
	}
	if links == nil {
		links = []sdk.BrainLink{}
	}
	graph.Links = links

	return &graph, nil
}

// CreateDreamProposal persists a new dream proposal.
func (b *Store) CreateDreamProposal(ctx context.Context, p *sdk.DreamProposal) error {
	_, err := b.db.ExecContext(ctx,
		`INSERT INTO dream_proposals (id, brain_id, snapshot, changes, summary, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.BrainID, p.Snapshot, p.Changes, p.Summary, p.Status, p.CreatedAt,
	)
	return err
}

func (b *Store) GetDreamProposal(ctx context.Context, proposalID string) (*sdk.DreamProposal, error) {
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

func (b *Store) ListDreamProposals(ctx context.Context, brainID string) ([]*sdk.DreamProposal, error) {
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, brain_id, summary, status, created_at, resolved_at, resolved_by
		 FROM dream_proposals WHERE brain_id = $1 ORDER BY created_at DESC`, brainID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func (b *Store) UpdateDreamProposalStatus(ctx context.Context, proposalID string, status sdk.DreamProposalStatus, resolvedBy string) error {
	now := time.Now()
	_, err := b.db.ExecContext(ctx,
		`UPDATE dream_proposals SET status=$1, resolved_at=$2, resolved_by=$3 WHERE id=$4`,
		status, now, resolvedBy, proposalID,
	)
	return err
}

// ResolveFileIDsByTitle converts [[link]] titles to file IDs within a brain.
func (b *Store) ResolveFileIDsByTitle(ctx context.Context, brainID string, titles []string) (map[string]string, error) {
	if len(titles) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(titles))
	args := []any{brainID}
	for i, title := range titles {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, title)
	}
	//nolint:gosec // placeholders contains only "$N" parameter markers, not user data
	query := fmt.Sprintf(
		`SELECT id, title FROM brain_files WHERE brain_id = $1 AND deleted_at IS NULL AND title IN (%s)`,
		strings.Join(placeholders, ","),
	)
	rows, err := b.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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
