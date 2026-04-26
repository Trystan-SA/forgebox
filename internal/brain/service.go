package brain

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
	"github.com/google/uuid"
)

// ArchiveRetention is how long soft-deleted brain files remain recoverable
// before the cleanup pass purges them.
const ArchiveRetention = 30 * 24 * time.Hour

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

// scheduleGraphRecompute recomputes the graph synchronously so the persisted
// view stays consistent with the latest mutation.
func (s *Service) scheduleGraphRecompute(brainID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, err := s.ComputeGraph(ctx, brainID); err != nil {
		slog.Error("brain graph recompute failed", "brain_id", brainID, "error", err)
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

	s.scheduleGraphRecompute(brainID)
	return file, nil
}

// UpdateFile updates a brain file, re-extracts links/hashtags, and re-embeds.
func (s *Service) UpdateFile(ctx context.Context, fileID, title, content string) (*sdk.BrainFile, error) {
	file, err := s.updateFileNoRecompute(ctx, fileID, title, content)
	if err != nil {
		return nil, err
	}
	s.scheduleGraphRecompute(file.BrainID)
	return file, nil
}

func (s *Service) updateFileNoRecompute(ctx context.Context, fileID, title, content string) (*sdk.BrainFile, error) {
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

// DeleteFile removes a brain file and triggers a graph recompute.
func (s *Service) DeleteFile(ctx context.Context, fileID string) error {
	file, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}
	if err := s.store.DeleteFile(ctx, fileID); err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	s.scheduleGraphRecompute(file.BrainID)
	return nil
}

// PurgeArchived hard-deletes soft-deleted brain files older than retention.
// For each purged file, [[Title]] references in still-active files of the
// same brain are removed from their content so links don't dangle.
// Returns the number of files purged.
func (s *Service) PurgeArchived(ctx context.Context, retention time.Duration) (int, error) {
	cutoff := time.Now().Add(-retention)
	expired, err := s.store.ListExpiredArchivedFiles(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("list expired archived: %w", err)
	}

	purged := 0
	for _, file := range expired {
		if err := s.removeLinkReferences(ctx, file.BrainID, file.Title); err != nil {
			slog.Warn("failed to clean link references before purge",
				"file_id", file.ID, "title", file.Title, "error", err)
		}
		if err := s.store.HardDeleteFile(ctx, file.ID); err != nil {
			slog.Error("hard-delete failed", "file_id", file.ID, "error", err)
			continue
		}
		purged++
		s.scheduleGraphRecompute(file.BrainID)
	}
	return purged, nil
}

// removeLinkReferences strips [[title]] occurrences from the content of all
// active files in the given brain.
func (s *Service) removeLinkReferences(ctx context.Context, brainID, title string) error {
	siblings, err := s.store.ListFiles(ctx, brainID)
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	pattern := regexp.MustCompile(`\[\[\s*` + regexp.QuoteMeta(title) + `\s*\]\]`)
	for _, sibling := range siblings {
		if !pattern.MatchString(sibling.Content) {
			continue
		}
		newContent := pattern.ReplaceAllString(sibling.Content, "")
		newContent = strings.TrimSpace(newContent)
		if _, err := s.updateFileNoRecompute(ctx, sibling.ID, sibling.Title, newContent); err != nil {
			return fmt.Errorf("update sibling %s: %w", sibling.ID, err)
		}
	}
	return nil
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
