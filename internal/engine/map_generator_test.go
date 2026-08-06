package engine

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// Phase 02.2: Map Generator E2E & Deterministic Tests
// Worldgen v2: planet-scale sanity invariants (spec P1.1-P1.4).

// gameSeed mirrors the seed construction in cmd/game/main.go.
func gameSeed(v byte) [32]byte {
	return [32]byte{v, v + 1, v + 2, v + 3, v + 4}
}

// landComponents flood-fills 4-connected land regions (BiomeID != BiomeOcean)
// and returns their sizes in tiles.
func landComponents(grid *MapGrid) []int {
	n := grid.Width * grid.Height
	visited := make([]bool, n)
	queue := make([]int, 0, 1024)
	var sizes []int
	for start := 0; start < n; start++ {
		if visited[start] || grid.Tiles[start].BiomeID == BiomeOcean {
			continue
		}
		size := 0
		visited[start] = true
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			i := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			size++
			x, y := i%grid.Width, i/grid.Width
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || nx >= grid.Width || ny < 0 || ny >= grid.Height {
					continue
				}
				j := ny*grid.Width + nx
				if !visited[j] && grid.Tiles[j].BiomeID != BiomeOcean {
					visited[j] = true
					queue = append(queue, j)
				}
			}
		}
		sizes = append(sizes, size)
	}
	return sizes
}

// enclosedWaterTiles counts water tiles NOT reachable from any border water
// tile — i.e. lakes fully enclosed by land.
func enclosedWaterTiles(grid *MapGrid) int {
	w, h := grid.Width, grid.Height
	n := w * h
	visited := make([]bool, n)
	queue := make([]int, 0, 1024)
	push := func(i int) {
		if !visited[i] && grid.Tiles[i].BiomeID == BiomeOcean {
			visited[i] = true
			queue = append(queue, i)
		}
	}
	for x := 0; x < w; x++ {
		push(x)
		push((h-1)*w + x)
	}
	for y := 0; y < h; y++ {
		push(y * w)
		push(y*w + w - 1)
	}
	for len(queue) > 0 {
		i := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		x, y := i%w, i/w
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			push(ny*w + nx)
		}
	}
	enclosed := 0
	for i := 0; i < n; i++ {
		if grid.Tiles[i].BiomeID == BiomeOcean && !visited[i] {
			enclosed++
		}
	}
	return enclosed
}

// TestGenerateMap_PlanetInvariants verifies the macro geography reads like a
// planet at game scale for several seeds: ocean fraction, multiple continents,
// islands, enclosed lakes, mountains and coastal beaches.
func TestGenerateMap_PlanetInvariants(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1024x1024 generation in -short mode")
	}
	const size = 1024
	for _, sv := range []byte{42, 7, 200} {
		grid := NewMapGrid(size, size)
		GenerateMap(grid, gameSeed(sv))
		n := size * size

		ocean := 0
		mountains := 0
		beaches := 0
		for i := 0; i < n; i++ {
			switch grid.Tiles[i].BiomeID {
			case BiomeOcean:
				ocean++
			case BiomeMountain:
				mountains++
			case BiomeBeach:
				beaches++
			}
		}
		oceanFrac := float64(ocean) / float64(n)
		if oceanFrac < 0.25 || oceanFrac > 0.60 {
			t.Errorf("seed %d: ocean fraction %.3f outside [0.25, 0.60]", sv, oceanFrac)
		}
		if mountains < 10 {
			t.Errorf("seed %d: expected mountain ranges, got only %d BiomeMountain tiles", sv, mountains)
		}
		if beaches == 0 {
			t.Errorf("seed %d: expected coastal beaches, got none", sv)
		}

		sizes := landComponents(grid)
		continents := 0
		islands := 0
		for _, s := range sizes {
			if s >= n*3/100 {
				continents++
			}
			if s < n/200 { // < 0.5% of tiles
				islands++
			}
		}
		if continents < 2 {
			t.Errorf("seed %d: expected >= 2 disjoint landmasses of >= 3%% of tiles, got %d (component sizes >= 1%%: %v)",
				sv, continents, filterSizes(sizes, n/100))
		}
		if islands < 1 {
			t.Errorf("seed %d: expected at least one island (< 0.5%% of tiles), got none", sv)
		}

		if lakes := enclosedWaterTiles(grid); lakes == 0 {
			t.Errorf("seed %d: expected at least one lake (water enclosed by land), got none", sv)
		}
	}
}

func filterSizes(sizes []int, minSize int) []int {
	var out []int
	for _, s := range sizes {
		if s >= minSize {
			out = append(out, s)
		}
	}
	return out
}

// TestGenerateMap_FullScaleDeterminism regenerates the full game-sized map and
// requires byte-identical output for the same seed.
func TestGenerateMap_FullScaleDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1024x1024 generation in -short mode")
	}
	const size = 1024
	grid1 := NewMapGrid(size, size)
	grid2 := NewMapGrid(size, size)
	GenerateMap(grid1, gameSeed(42))
	GenerateMap(grid2, gameSeed(42))
	for i := range grid1.Tiles {
		if grid1.Tiles[i] != grid2.Tiles[i] {
			t.Fatalf("tile mismatch at index %d: %+v vs %+v", i, grid1.Tiles[i], grid2.Tiles[i])
		}
		if grid1.Resources[i] != grid2.Resources[i] {
			t.Fatalf("resource mismatch at index %d: %+v vs %+v", i, grid1.Resources[i], grid2.Resources[i])
		}
	}
}

// TestGenerateMap_Performance keeps full-scale generation inside the ~4s budget.
func TestGenerateMap_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1024x1024 generation in -short mode")
	}
	grid := NewMapGrid(1024, 1024)
	start := time.Now()
	GenerateMap(grid, gameSeed(42))
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("GenerateMap(1024x1024) took %v, budget is 4s", elapsed)
	}
}

// TestGenerateMap_SmallGridSafe guards the degenerate small-map path used by
// system tests (no panics, still deterministic).
func TestGenerateMap_SmallGridSafe(t *testing.T) {
	for _, size := range []int{10, 16, 100} {
		g1 := NewMapGrid(size, size)
		g2 := NewMapGrid(size, size)
		GenerateMap(g1, gameSeed(9))
		GenerateMap(g2, gameSeed(9))
		for i := range g1.Tiles {
			if g1.Tiles[i] != g2.Tiles[i] {
				t.Fatalf("size %d: tile mismatch at %d", size, i)
			}
		}
	}
}

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
