package brain

import (
	"math"
	"math/rand"
)

// AssignClusters runs simple k-means on embedding vectors and returns
// the cluster index for each vector. If len(embeddings) < k, all get cluster 0.
func AssignClusters(embeddings [][]float32, k int) []int {
	n := len(embeddings)
	if n == 0 {
		return nil
	}
	if n <= k {
		assignments := make([]int, n)
		for i := range assignments {
			assignments[i] = 0
		}
		return assignments
	}

	dim := len(embeddings[0])
	assignments := make([]int, n)

	rng := rand.New(rand.NewSource(42))
	centroids := make([][]float32, k)
	perm := rng.Perm(n)
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, dim)
		copy(centroids[i], embeddings[perm[i]])
	}

	for iter := 0; iter < 50; iter++ {
		changed := false
		for i, vec := range embeddings {
			best := 0
			bestDist := float32(math.MaxFloat32)
			for c, centroid := range centroids {
				d := euclideanDist(vec, centroid)
				if d < bestDist {
					bestDist = d
					best = c
				}
			}
			if assignments[i] != best {
				assignments[i] = best
				changed = true
			}
		}

		if !changed {
			break
		}

		counts := make([]int, k)
		newCentroids := make([][]float32, k)
		for i := range newCentroids {
			newCentroids[i] = make([]float32, dim)
		}
		for i, vec := range embeddings {
			c := assignments[i]
			counts[c]++
			for d := range vec {
				newCentroids[c][d] += vec[d]
			}
		}
		for c := range centroids {
			if counts[c] > 0 {
				for d := range centroids[c] {
					centroids[c][d] = newCentroids[c][d] / float32(counts[c])
				}
			}
		}
	}

	return assignments
}

// Project2D projects high-dimensional embeddings to 2D using the two
// highest-variance dimensions. Returns [x, y] pairs normalized to [100, 900] x [100, 700].
func Project2D(embeddings [][]float32) [][2]float64 {
	n := len(embeddings)
	if n == 0 {
		return nil
	}

	dim := len(embeddings[0])
	if dim <= 2 {
		points := make([][2]float64, n)
		for i, vec := range embeddings {
			points[i] = [2]float64{float64(vec[0]), 0}
			if len(vec) > 1 {
				points[i][1] = float64(vec[1])
			}
		}
		return normalizePoints(points)
	}

	mean := make([]float64, dim)
	for _, vec := range embeddings {
		for d, v := range vec {
			mean[d] += float64(v) / float64(n)
		}
	}

	variance := make([]float64, dim)
	for _, vec := range embeddings {
		for d, v := range vec {
			diff := float64(v) - mean[d]
			variance[d] += diff * diff
		}
	}

	d1, d2 := 0, 1
	if variance[d2] > variance[d1] {
		d1, d2 = d2, d1
	}
	for d := 2; d < dim; d++ {
		if variance[d] > variance[d1] {
			d2 = d1
			d1 = d
		} else if variance[d] > variance[d2] {
			d2 = d
		}
	}

	points := make([][2]float64, n)
	for i, vec := range embeddings {
		points[i] = [2]float64{
			float64(vec[d1]) - mean[d1],
			float64(vec[d2]) - mean[d2],
		}
	}

	return normalizePoints(points)
}

func normalizePoints(points [][2]float64) [][2]float64 {
	if len(points) == 0 {
		return points
	}
	minX, maxX := points[0][0], points[0][0]
	minY, maxY := points[0][1], points[0][1]
	for _, p := range points {
		if p[0] < minX {
			minX = p[0]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	for i := range points {
		points[i][0] = ((points[i][0]-minX)/rangeX)*800 + 100
		points[i][1] = ((points[i][1]-minY)/rangeY)*600 + 100
	}
	return points
}

func euclideanDist(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return float32(math.Sqrt(float64(sum)))
}

// ClusterColors is a palette of distinct colors for graph clusters.
var ClusterColors = []string{
	"#6366f1", // indigo
	"#10b981", // emerald
	"#f59e0b", // amber
	"#ef4444", // red
	"#8b5cf6", // violet
	"#06b6d4", // cyan
	"#ec4899", // pink
	"#84cc16", // lime
	"#f97316", // orange
	"#14b8a6", // teal
}
