package engine

import (
	"testing"
	"unsafe"
)

// Phase 02.1: Geography - The Map Data Array Tests

// TestMapGridIntegrity verifies that 1D array math maps perfectly to 2D coordinates.
func TestMapGridIntegrity(t *testing.T) {
	width := 100
	height := 100
	grid := NewMapGrid(width, height)

	// Set a tile with extreme values at a specific coordinate
	targetX, targetY := 99, 99
	expectedTile := TileData{
		Elevation:   255,
		Moisture:    128,
		Temperature: 64,
	}

	grid.SetTile(targetX, targetY, expectedTile)

	// Retrieve the tile and verify it matches
	retrievedTile := grid.GetTile(targetX, targetY)

	if retrievedTile.Elevation != expectedTile.Elevation {
		t.Errorf("Expected Elevation %d, got %d", expectedTile.Elevation, retrievedTile.Elevation)
	}
	if retrievedTile.Moisture != expectedTile.Moisture {
		t.Errorf("Expected Moisture %d, got %d", expectedTile.Moisture, retrievedTile.Moisture)
	}
	if retrievedTile.Temperature != expectedTile.Temperature {
		t.Errorf("Expected Temperature %d, got %d", expectedTile.Temperature, retrievedTile.Temperature)
	}

	// Verify that an adjacent tile was NOT affected (testing for cross-contamination)
	adjacentTile := grid.GetTile(98, 99)
	if adjacentTile.Elevation != 0 || adjacentTile.Moisture != 0 || adjacentTile.Temperature != 0 {
		t.Errorf("Adjacent tile was modified unexpectedly: %+v", adjacentTile)
	}
}

// TestTileDataDOD verifies that TileData strictly follows Data-Oriented Design constraints.
func TestTileDataDOD(t *testing.T) {
	// A struct with three uint8 fields should ideally take exactly 3 bytes.
	// We use unsafe.Sizeof to verify the compiler is packing it tightly without padding.
	var tile TileData
	expectedSize := uintptr(3)
	actualSize := unsafe.Sizeof(tile)

	if actualSize != expectedSize {
		t.Errorf("DOD Violation: Expected TileData size to be %d bytes, got %d bytes. Check for compiler padding.", expectedSize, actualSize)
	}
}

// TestMapGridOutOfBounds verifies that out-of-bounds access returns a zero-value tile safely.
func TestMapGridOutOfBounds(t *testing.T) {
	grid := NewMapGrid(10, 10)

	outOfBoundsTile := grid.GetTile(-1, -1)
	if outOfBoundsTile.Elevation != 0 || outOfBoundsTile.Moisture != 0 || outOfBoundsTile.Temperature != 0 {
		t.Errorf("Expected zero-value tile for out-of-bounds access, got: %+v", outOfBoundsTile)
	}

	outOfBoundsTile = grid.GetTile(10, 10)
	if outOfBoundsTile.Elevation != 0 || outOfBoundsTile.Moisture != 0 || outOfBoundsTile.Temperature != 0 {
		t.Errorf("Expected zero-value tile for out-of-bounds access, got: %+v", outOfBoundsTile)
	}

	// Ensure setting out-of-bounds doesn't panic
	grid.SetTile(10, 10, TileData{Elevation: 100})
}
