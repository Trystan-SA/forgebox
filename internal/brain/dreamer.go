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

	snapshotData, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

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

	var changes []sdk.DreamChange
	if err := json.Unmarshal([]byte(resp.Content), &changes); err != nil {
		return nil, fmt.Errorf("parse LLM response as changes: %w", err)
	}

	if len(changes) == 0 {
		slog.Info("dream produced no changes", "brain_id", brainID)
		return nil, nil
	}

	changesJSON, _ := json.Marshal(changes)

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
