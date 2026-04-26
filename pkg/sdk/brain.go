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
	FileID    string   `json:"file_id"`
	Title     string   `json:"title"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	ClusterID int      `json:"cluster_id"`
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
	HardDeleteFile(ctx context.Context, fileID string) error
	ListExpiredArchivedFiles(ctx context.Context, before time.Time) ([]*BrainFile, error)
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
