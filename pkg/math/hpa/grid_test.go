package hpa

import (
	"testing"
)

// Phase 04.1: Hierarchical Pathfinding (HPA*) Implementation

func TestNewAbstractGrid(t *testing.T) {
	// Map size: 100x100, Region size: 16x16
	mapWidth := 100
	mapHeight := 100
	regionSize := 16

	ag := NewAbstractGrid(mapWidth, mapHeight, regionSize)

	if ag == nil {
		t.Fatalf("Expected AbstractGrid to be created, got nil")
	}

	if ag.RegionWidth != regionSize {
		t.Errorf("Expected RegionWidth to be %d, got %d", regionSize, ag.RegionWidth)
	}

	if ag.RegionHeight != regionSize {
		t.Errorf("Expected RegionHeight to be %d, got %d", regionSize, ag.RegionHeight)
	}

	// Calculate expected clusters (100 / 16) = 6.25 -> 7 cols, 7 rows -> 49 clusters
	expectedClusters := 49
	if len(ag.Clusters) != expectedClusters {
		t.Errorf("Expected %d Clusters, got %d", expectedClusters, len(ag.Clusters))
	}
}

func TestBuildClustersEdgeCases(t *testing.T) {
	// Map size 10x10, Region size 16x16 (Map smaller than region)
	ag1 := NewAbstractGrid(10, 10, 16)
	if len(ag1.Clusters) != 1 {
		t.Errorf("Expected 1 Cluster, got %d", len(ag1.Clusters))
	}

	if len(ag1.Clusters) > 0 {
		c := ag1.Clusters[0]
		if c.Width != 10 || c.Height != 10 {
			t.Errorf("Expected Cluster size 10x10, got %dx%d", c.Width, c.Height)
		}
	}

	// Non-perfect division edge clusters
	ag2 := NewAbstractGrid(100, 100, 16)

	// Check the last cluster (bottom-right edge)
	// Col 6 (x=96), Row 6 (y=96)
	lastCluster := ag2.Clusters[len(ag2.Clusters)-1]
	if lastCluster.X != 96 || lastCluster.Y != 96 {
		t.Errorf("Expected last cluster at X:96, Y:96, got X:%d, Y:%d", lastCluster.X, lastCluster.Y)
	}

	if lastCluster.Width != 4 || lastCluster.Height != 4 {
		t.Errorf("Expected last cluster size 4x4, got %dx%d", lastCluster.Width, lastCluster.Height)
	}

	// Check a normal full cluster
	firstCluster := ag2.Clusters[0]
	if firstCluster.Width != 16 || firstCluster.Height != 16 {
		t.Errorf("Expected first cluster size 16x16, got %dx%d", firstCluster.Width, firstCluster.Height)
	}
}

func TestDeterministicCheck(t *testing.T) {
	ag1 := NewAbstractGrid(50, 50, 16)
	ag2 := NewAbstractGrid(50, 50, 16)

	if len(ag1.Clusters) != len(ag2.Clusters) {
		t.Fatalf("Deterministic Check Failed: Lengths mismatch")
	}

	for i := range ag1.Clusters {
		c1 := ag1.Clusters[i]
		c2 := ag2.Clusters[i]

		if c1.ID != c2.ID || c1.X != c2.X || c1.Y != c2.Y || c1.Width != c2.Width || c1.Height != c2.Height {
			t.Errorf("Deterministic Check Failed at index %d: %+v != %+v", i, c1, c2)
		}
	}
}
