package engine

// Phase 02.1: Geography - The Map Data Array
// Objective: Map numerical data arrays that construct the physical terrain of the game world independent of rendering.

// TileData represents the terrain data for a single grid coordinate.
// It is a tightly packed integer stack holding uint8 values to minimize memory overhead.
type TileData struct {
	Elevation   uint8
	Moisture    uint8
	Temperature uint8
	BiomeID     uint8
}

// Phase 02.4: Static Resource Depots
// ResourceDepot stores resource values. Kept parallel to TileData to avoid cache pollution.
type ResourceDepot struct {
	WoodValue  uint8
	StoneValue uint8
	IronValue  uint8
	FoodValue  uint8
}

// Phase 02.5: The Infrastructure Layer
// TileState tracks infrastructure logic like desire paths. Kept parallel to TileData.
type TileState struct {
	FootTraffic uint32
}

// MapGrid represents the game world map.
// It uses a contiguous 1D array masquerading as a 2D matrix (Grid[y * width + x]).
// This is dramatically faster for cache-lines than [][]Tile, adhering to Data-Oriented Design (DOD) principles.
type MapGrid struct {
	Width      int
	Height     int
	Tiles      []TileData
	Resources  []ResourceDepot
	TileStates []TileState
}

// NewMapGrid initializes a new MapGrid with the specified width and height.
func NewMapGrid(width, height int) *MapGrid {
	return &MapGrid{
		Width:      width,
		Height:     height,
		Tiles:      make([]TileData, width*height),
		Resources:  make([]ResourceDepot, width*height),
		TileStates: make([]TileState, width*height),
	}
}

// GetTile returns the TileData at the specified (x, y) coordinates.
// It uses 1D array indexing: index = y * width + x.
func (m *MapGrid) GetTile(x, y int) TileData {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		// Return default empty tile for out of bounds access,
		// alternatively could panic depending on desired engine strictness.
		// For robustness, returning zero-value TileData.
		return TileData{}
	}
	return m.Tiles[y*m.Width+x]
}

// SetTile updates the TileData at the specified (x, y) coordinates.
func (m *MapGrid) SetTile(x, y int, tile TileData) {
	if x >= 0 && x < m.Width && y >= 0 && y < m.Height {
		m.Tiles[y*m.Width+x] = tile
	}
}
