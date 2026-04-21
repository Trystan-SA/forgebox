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
