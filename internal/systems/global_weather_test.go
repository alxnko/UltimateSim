package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 19.2: Global Weather System Deterministic E2E Test
func TestGlobalWeatherSystem_Determinism(t *testing.T) {
	world1 := ecs.NewWorld()
	world2 := ecs.NewWorld()

	// Create identical deterministic grids
	grid1 := &engine.MapGrid{
		Width:      10,
		Height:     10,
		Tiles:      make([]engine.TileData, 100),
		Resources:  make([]engine.ResourceDepot, 100),
		TileStates: make([]engine.TileState, 100),
	}
	engine.GenerateMap(grid1, [32]byte{1})

	grid2 := &engine.MapGrid{
		Width:      10,
		Height:     10,
		Tiles:      make([]engine.TileData, 100),
		Resources:  make([]engine.ResourceDepot, 100),
		TileStates: make([]engine.TileState, 100),
	}
	engine.GenerateMap(grid2, [32]byte{1})

	sys1 := NewGlobalWeatherSystem(&world1, grid1)
	sys2 := NewGlobalWeatherSystem(&world2, grid2)

	// Ensure there is at least one forest tile that we explicitly override to test mutation
	// We'll set tile 0 to TemperateDeciduousForest, very hot (165), with 10 WoodValue.
	grid1.Tiles[0].BiomeID = engine.BiomeTemperateDeciduousForest
	grid1.Tiles[0].Temperature = 165
	grid1.Resources[0].WoodValue = 10

	grid2.Tiles[0].BiomeID = engine.BiomeTemperateDeciduousForest
	grid2.Tiles[0].Temperature = 165
	grid2.Resources[0].WoodValue = 10

	// Tick systems
	for i := 0; i < 100000; i++ {
		sys1.Update(&world1)
		sys2.Update(&world2)
	}

	// At tick 100,000, temperature increases by 5 (165 -> 170).
	// Because 170 > 165, the forest must mutate into Grassland and WoodValue = 0.

	if grid1.Tiles[0].BiomeID != engine.BiomeGrassland {
		t.Errorf("Expected biome to mutate to BiomeGrassland due to climate shift, got %d", grid1.Tiles[0].BiomeID)
	}
	if grid1.Resources[0].WoodValue != 0 {
		t.Errorf("Expected wood value to drop to 0 due to climate shift, got %d", grid1.Resources[0].WoodValue)
	}

	// Verify both simulated worlds produced identically deterministic outputs
	if grid1.Tiles[0].Temperature != grid2.Tiles[0].Temperature ||
		grid1.Tiles[0].BiomeID != grid2.Tiles[0].BiomeID ||
		grid1.Resources[0].WoodValue != grid2.Resources[0].WoodValue {
		t.Fatal("GlobalWeatherSystem broke determinism rules: identical seeds produced diverging map grids.")
	}
}
