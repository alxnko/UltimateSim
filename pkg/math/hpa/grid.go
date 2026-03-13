package hpa

// Phase 04.1: Hierarchical Pathfinding (HPA*) Implementation

// Cluster represents a partitioned region (e.g. 16x16 or 32x32) of the global MapGrid.
// Using uint16 for ID mapping bounds and int for coordinates to match MapGrid access.
type Cluster struct {
	ID     uint16
	X      int
	Y      int
	Width  int
	Height int
}

// Node represents a specific coordinate point on the tactical grid level,
// frequently used to represent gateways or sub-paths.
// Uses uint16 for coordinates to minimize memory footprint in arrays, assuming max map dimension <= 65535.
type Node struct {
	X    uint16
	Y    uint16
	Cost float32 // float32 to adhere strictly to DOD guidelines
}

// Gateway maps connected portals between two distinct Clusters.
// Allows for fast strategic pathfinding cross-map without tactical node evaluation.
type Gateway struct {
	ID         uint16
	Cluster1ID uint16
	Cluster2ID uint16
	Nodes      []Node // Actual passable tiles connecting the clusters
}

// AbstractGrid holds the macro-level representation of the terrain.
// Maintains contiguous 1D arrays of Clusters and Gateways for cache locality.
type AbstractGrid struct {
	Clusters     []Cluster
	Gateways     []Gateway
	RegionWidth  int
	RegionHeight int
}

// NewAbstractGrid initializes a new AbstractGrid based on raw map dimensions and a desired region size.
func NewAbstractGrid(mapWidth, mapHeight, regionSize int) *AbstractGrid {
	ag := &AbstractGrid{
		RegionWidth:  regionSize,
		RegionHeight: regionSize,
	}
	ag.BuildClusters(mapWidth, mapHeight)
	return ag
}

// BuildNavMesh specifically maps over the `BiomeOcean` tile array to generate a secondary
// nav-mesh for oceanic pathfinding, skipping landmasses entirely.
// `gridTiles` represents the 1D flat array from `engine.MapGrid.Tiles`.
// We use a simplified threshold representation here: BiomeID 0 is BiomeOcean.
func (ag *AbstractGrid) BuildNavMesh(gridTiles []uint8, mapWidth int) {
	// For Phase 17.2, we iterate over the cluster grid and only map nodes
	// that align with BiomeOcean.
	// We iterate over the pre-built clusters, evaluating passable oceanic gateways.

	ag.Gateways = make([]Gateway, 0)
	var gatewayID uint16 = 0

	// Compare adjacent clusters
	for i := 0; i < len(ag.Clusters); i++ {
		c1 := ag.Clusters[i]

		for j := i + 1; j < len(ag.Clusters); j++ {
			c2 := ag.Clusters[j]

			// Check if c1 and c2 are adjacent
			isAdjacent := false
			if c1.X+c1.Width == c2.X && c1.Y == c2.Y {
				isAdjacent = true // Horizontal adjacency
			} else if c1.Y+c1.Height == c2.Y && c1.X == c2.X {
				isAdjacent = true // Vertical adjacency
			}

			if isAdjacent {
				// We scan the shared boundary looking for consecutive BiomeOcean tiles.
				// In a full implementation, we build the Node array for the Gateway.
				// Here, we'll extract the first matching node for DOD scaling.

				var nodes []Node

				// Simplistic boundary check (vertical adjacency)
				if c1.Y+c1.Height == c2.Y {
					for x := c1.X; x < c1.X+c1.Width; x++ {
						idx := (c1.Y+c1.Height-1)*mapWidth + x
						if idx < len(gridTiles) && gridTiles[idx] == 0 { // 0 == BiomeOcean
							nodes = append(nodes, Node{X: uint16(x), Y: uint16(c1.Y + c1.Height - 1), Cost: 1.0})
						}
					}
				} else if c1.X+c1.Width == c2.X {
					// Horizontal adjacency check
					for y := c1.Y; y < c1.Y+c1.Height; y++ {
						idx := y*mapWidth + (c1.X + c1.Width - 1)
						if idx < len(gridTiles) && gridTiles[idx] == 0 { // 0 == BiomeOcean
							nodes = append(nodes, Node{X: uint16(c1.X + c1.Width - 1), Y: uint16(y), Cost: 1.0})
						}
					}
				}

				if len(nodes) > 0 {
					ag.Gateways = append(ag.Gateways, Gateway{
						ID:         gatewayID,
						Cluster1ID: c1.ID,
						Cluster2ID: c2.ID,
						Nodes:      nodes,
					})
					gatewayID++
				}
			}
		}
	}
}

// BuildClusters partitions the global map dimensions into a flat array of Cluster structs.
// Edge regions are automatically sized to fit remaining map bounds.
func (ag *AbstractGrid) BuildClusters(mapWidth, mapHeight int) {
	// Calculate the number of clusters needed
	cols := (mapWidth + ag.RegionWidth - 1) / ag.RegionWidth
	rows := (mapHeight + ag.RegionHeight - 1) / ag.RegionHeight
	totalClusters := cols * rows

	// Pre-allocate the exact size to avoid reallocation and maintain contiguous memory
	ag.Clusters = make([]Cluster, 0, totalClusters)

	var currentID uint16 = 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			startX := x * ag.RegionWidth
			startY := y * ag.RegionHeight

			// Handle edge cases where the map isn't perfectly divisible by regionSize
			width := ag.RegionWidth
			if startX+width > mapWidth {
				width = mapWidth - startX
			}

			height := ag.RegionHeight
			if startY+height > mapHeight {
				height = mapHeight - startY
			}

			ag.Clusters = append(ag.Clusters, Cluster{
				ID:     currentID,
				X:      startX,
				Y:      startY,
				Width:  width,
				Height: height,
			})
			currentID++
		}
	}
}
