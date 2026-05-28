package cluster

import (
	"math"
)

type Point struct {
	ID   int64
	Vec  []float32
}

type Cluster struct {
	ID    int
	Points []Point
}

func DBSCAN(points []Point, eps float32, minPts int) []Cluster {
	if len(points) == 0 {
		return nil
	}

	n := len(points)
	labels := make([]int, n)
	for i := range labels {
		labels[i] = -1
	}

	clusterID := 0

	distanceMatrix := precomputeDistances(points)

	for i := 0; i < n; i++ {
		if labels[i] != -1 {
			continue
		}

		neighbors := regionQuery(i, distanceMatrix, eps, n)

		if len(neighbors) < minPts {
			labels[i] = 0
			continue
		}

		clusterID++
		labels[i] = clusterID

		queue := make([]int, len(neighbors))
		copy(queue, neighbors)

		expandCluster(points, labels, &queue, clusterID, eps, minPts, distanceMatrix, n)
	}

	return buildClusters(points, labels, clusterID)
}

func precomputeDistances(points []Point) [][]float32 {
	n := len(points)
	dist := make([][]float32, n)
	for i := range dist {
		dist[i] = make([]float32, n)
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			d := euclideanDist(points[i].Vec, points[j].Vec)
			dist[i][j] = d
			dist[j][i] = d
		}
	}

	return dist
}

func euclideanDist(a, b []float32) float32 {
	var sum float32
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return float32(math.Sqrt(float64(sum)))
}

func regionQuery(idx int, dist [][]float32, eps float32, n int) []int {
	var neighbors []int
	for i := 0; i < n; i++ {
		if dist[idx][i] <= eps {
			neighbors = append(neighbors, i)
		}
	}
	return neighbors
}

func expandCluster(points []Point, labels []int, queue *[]int, clusterID int, eps float32, minPts int, dist [][]float32, n int) {
	processed := make(map[int]bool)

	for len(*queue) > 0 {
		curr := (*queue)[0]
		*queue = (*queue)[1:]

		if processed[curr] {
			continue
		}
		processed[curr] = true

		if labels[curr] == 0 {
			labels[curr] = clusterID
		}

		if labels[curr] != -1 && labels[curr] != clusterID {
			continue
		}

		labels[curr] = clusterID

		neighbors := regionQuery(curr, dist, eps, n)

		if len(neighbors) >= minPts {
			for _, nb := range neighbors {
				if !processed[nb] {
					*queue = append(*queue, nb)
				}
			}
		}
	}
}

func buildClusters(points []Point, labels []int, numClusters int) []Cluster {
	clusters := make([]Cluster, numClusters)
	for i := range clusters {
		clusters[i].ID = i + 1
	}

	for i, label := range labels {
		if label > 0 {
			clusters[label-1].Points = append(clusters[label-1].Points, points[i])
		}
	}

	return clusters
}
