package brain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/forgebox/forgebox/pkg/sdk"
)

const (
	graphCanvasCenterX  = 500.0
	graphCanvasCenterY  = 400.0
	graphFallbackRadius = 250.0
)

// ComputeGraph builds and persists the visualization graph for a brain.
// Files with embeddings are clustered (k-means) and projected to 2D.
// Files without embeddings are placed on a fallback circle so they remain visible.
func (s *Service) ComputeGraph(ctx context.Context, brainID string) (*sdk.BrainGraph, error) {
	files, err := s.store.ListFiles(ctx, brainID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	// Hashtags are deterministic from content, so derive them in-process
	// instead of doing one store round-trip per file.
	fileTags := make(map[string][]string, len(files))
	for _, f := range files {
		fileTags[f.ID] = ExtractHashtags(f.Content)
	}

	var (
		embFiles    []*sdk.BrainFile
		embeddings  [][]float32
		noEmbFiles  []*sdk.BrainFile
	)
	for _, f := range files {
		if len(f.Embedding) > 0 {
			embFiles = append(embFiles, f)
			embeddings = append(embeddings, f.Embedding)
		} else {
			noEmbFiles = append(noEmbFiles, f)
		}
	}

	clusters := []sdk.GraphCluster{}
	nodes := []sdk.GraphNode{}

	if len(embFiles) >= 3 {
		k := len(embFiles) / 2
		if k < 2 {
			k = 2
		}
		if k > len(ClusterColors) {
			k = len(ClusterColors)
		}

		assignments := AssignClusters(embeddings, k)
		points := Project2D(embeddings)

		used := make(map[int]bool)
		tagFreq := make(map[int]map[string]int)
		for i, c := range assignments {
			used[c] = true
			if tagFreq[c] == nil {
				tagFreq[c] = map[string]int{}
			}
			for _, t := range fileTags[embFiles[i].ID] {
				tagFreq[c][t]++
			}
		}

		clusterIDs := make([]int, 0, len(used))
		for c := range used {
			clusterIDs = append(clusterIDs, c)
		}
		sort.Ints(clusterIDs)
		for _, c := range clusterIDs {
			clusters = append(clusters, sdk.GraphCluster{
				ID:    c,
				Color: ClusterColors[c%len(ClusterColors)],
				Label: topTag(tagFreq[c]),
			})
		}

		for i, f := range embFiles {
			nodes = append(nodes, sdk.GraphNode{
				FileID:    f.ID,
				Title:     f.Title,
				X:         points[i][0],
				Y:         points[i][1],
				ClusterID: assignments[i],
				Hashtags:  fileTags[f.ID],
			})
		}
	} else {
		// <3 embedded files: skip clustering, single default cluster.
		if len(files) > 0 {
			clusters = append(clusters, sdk.GraphCluster{
				ID:    0,
				Color: ClusterColors[0],
				Label: "",
			})
		}
		nodes = append(nodes, circlePlace(embFiles, fileTags, graphFallbackRadius*0.6)...)
	}

	if len(noEmbFiles) > 0 {
		nodes = append(nodes, circlePlace(noEmbFiles, fileTags, graphFallbackRadius)...)
	}

	links, err := s.store.GetFileLinks(ctx, brainID)
	if err != nil {
		return nil, fmt.Errorf("get links: %w", err)
	}
	if links == nil {
		links = []sdk.BrainLink{}
	}

	graph := &sdk.BrainGraph{
		BrainID:    brainID,
		Clusters:   clusters,
		Nodes:      nodes,
		Links:      links,
		ComputedAt: time.Now(),
	}

	if err := s.store.SaveGraph(ctx, graph); err != nil {
		return nil, fmt.Errorf("save graph: %w", err)
	}
	return graph, nil
}

// circlePlace lays files out on a circle around the canvas center.
func circlePlace(files []*sdk.BrainFile, tags map[string][]string, radius float64) []sdk.GraphNode {
	n := len(files)
	if n == 0 {
		return nil
	}
	out := make([]sdk.GraphNode, 0, n)
	for i, f := range files {
		angle := 2 * math.Pi * float64(i) / float64(n)
		out = append(out, sdk.GraphNode{
			FileID:    f.ID,
			Title:     f.Title,
			X:         graphCanvasCenterX + radius*math.Cos(angle),
			Y:         graphCanvasCenterY + radius*math.Sin(angle),
			ClusterID: 0,
			Hashtags:  tags[f.ID],
		})
	}
	return out
}

// topTag returns the most frequent hashtag in the map, or "" if empty.
func topTag(freq map[string]int) string {
	best := ""
	bestN := 0
	for tag, n := range freq {
		if n > bestN || (n == bestN && tag < best) {
			best = tag
			bestN = n
		}
	}
	return best
}

