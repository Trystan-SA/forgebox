package brain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockEmbedder(t *testing.T) {
	emb := NewMockEmbedder(4)
	vec, err := emb.Embed(context.Background(), "test")
	require.NoError(t, err)
	assert.Len(t, vec, 4)
	assert.Equal(t, 1, emb.called)
	assert.Equal(t, 4, emb.Dimension())
}
