package brain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignClusters(t *testing.T) {
	embeddings := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.9, 0.1, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
		{0.0, 0.1, 0.9, 0.0},
	}

	clusters := AssignClusters(embeddings, 2)
	require.Len(t, clusters, 4)

	assert.Equal(t, clusters[0], clusters[1])
	assert.Equal(t, clusters[2], clusters[3])
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
	embeddings := [][]float32{
		{1.0, 0.0},
	}
	clusters := AssignClusters(embeddings, 3)
	assert.Equal(t, []int{0}, clusters)
}
