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

	embedding, err := s.embedder.Embed(ctx, title+"\n"+content)
	if err != nil {
		slog.Warn("embedding failed, saving file without embedding", "error", err, "file_id", file.ID)
	} else {
		file.Embedding = embedding
	}

	if err := s.store.CreateFile(ctx, file); err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	tags := ExtractHashtags(content)
	if len(tags) > 0 {
		if err := s.store.SetFileHashtags(ctx, file.ID, tags); err != nil {
			slog.Warn("failed to save hashtags", "error", err, "file_id", file.ID)
		}
	}

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
