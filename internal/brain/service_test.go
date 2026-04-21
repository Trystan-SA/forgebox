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
