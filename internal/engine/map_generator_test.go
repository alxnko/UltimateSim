package engine

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Phase 02.2: Map Generator E2E & Deterministic Tests

func TestMapGeneration_Determinism(t *testing.T) {
	width := 100
	height := 100
	grid1 := NewMapGrid(width, height)
	grid2 := NewMapGrid(width, height)

	seed := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// Generate map twice with same seed
	GenerateMap(grid1, seed)
	GenerateMap(grid2, seed)

	// Validate they are identical byte-for-byte
	buf1 := new(bytes.Buffer)
	err := binary.Write(buf1, binary.LittleEndian, grid1.Tiles)
	if err != nil {
		t.Fatalf("Failed to write grid1: %v", err)
	}

	buf2 := new(bytes.Buffer)
	err = binary.Write(buf2, binary.LittleEndian, grid2.Tiles)
	if err != nil {
		t.Fatalf("Failed to write grid2: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Errorf("Determinism failure: Grid1 and Grid2 outputs differ despite identical seed.")
	}

	// Spot check a specific tile to ensure it's not all zeros
	tile := grid1.GetTile(50, 50)
	if tile.Elevation == 0 && tile.Moisture == 0 && tile.Temperature == 0 {
		t.Logf("Warning: Tile(50,50) is all zeros, map generation might be broken or seed produces exact 0. e=%d m=%d t=%d", tile.Elevation, tile.Moisture, tile.Temperature)
	}

	for i := 0; i < len(grid1.Tiles); i++ {
		t1 := grid1.Tiles[i]
		t2 := grid2.Tiles[i]
		if t1.BiomeID != t2.BiomeID {
			t.Fatalf("BiomeID mismatch at index %d: run1=%d, run2=%d", i, t1.BiomeID, t2.BiomeID)
		}

		// Phase 02.4: Validate deterministic resource generation
		r1 := grid1.Resources[i]
		r2 := grid2.Resources[i]
		if r1.WoodValue != r2.WoodValue || r1.StoneValue != r2.StoneValue || r1.IronValue != r2.IronValue {
			t.Fatalf("Resource mismatch at index %d: run1=%+v, run2=%+v", i, r1, r2)
		}
	}
}

func TestGenerateMap_ResourceDepots(t *testing.T) {
	width := 100
	height := 100
	grid := NewMapGrid(width, height)
	seed := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	GenerateMap(grid, seed)

	for i, tile := range grid.Tiles {
		resource := grid.Resources[i]

		switch tile.BiomeID {
		case BiomeTemperateDeciduousForest, BiomeTemperateRainForest, BiomeTropicalSeasonalForest, BiomeTropicalRainForest:
			if resource.WoodValue < 50 || resource.WoodValue > 150 {
				t.Errorf("Forest Biome (ID %d) expected WoodValue between 50 and 150, got %d", tile.BiomeID, resource.WoodValue)
			}
			if resource.StoneValue != 0 || resource.IronValue != 0 {
				t.Errorf("Forest Biome (ID %d) expected no Stone or Iron, got Stone=%d, Iron=%d", tile.BiomeID, resource.StoneValue, resource.IronValue)
			}
		case BiomeMountain:
			if resource.StoneValue < 100 || resource.StoneValue > 255 {
				t.Errorf("Mountain Biome (ID %d) expected StoneValue between 100 and 255, got %d", tile.BiomeID, resource.StoneValue)
			}
			if resource.IronValue != 0 && (resource.IronValue < 20 || resource.IronValue > 100) {
				t.Errorf("Mountain Biome (ID %d) expected IronValue either 0 or between 20 and 100, got %d", tile.BiomeID, resource.IronValue)
			}
			if resource.WoodValue != 0 {
				t.Errorf("Mountain Biome (ID %d) expected no Wood, got Wood=%d", tile.BiomeID, resource.WoodValue)
			}
		}
	}
}

func TestMapGeneration_DifferentSeeds(t *testing.T) {
	width := 100
	height := 100
	grid1 := NewMapGrid(width, height)
	grid2 := NewMapGrid(width, height)

	seed1 := [32]byte{1, 2, 3}
	seed2 := [32]byte{4, 5, 6}

	GenerateMap(grid1, seed1)
	GenerateMap(grid2, seed2)

	// Validate they are different byte-for-byte
	buf1 := new(bytes.Buffer)
	err := binary.Write(buf1, binary.LittleEndian, grid1.Tiles)
	if err != nil {
		t.Fatalf("Failed to write grid1: %v", err)
	}

	buf2 := new(bytes.Buffer)
	err = binary.Write(buf2, binary.LittleEndian, grid2.Tiles)
	if err != nil {
		t.Fatalf("Failed to write grid2: %v", err)
	}

	if bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Errorf("Variation failure: Grid1 and Grid2 outputs are identical despite different seeds.")
	}
}
