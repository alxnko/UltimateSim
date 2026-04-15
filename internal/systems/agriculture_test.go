package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 58 - The Agricultural Engine (AgricultureSystem)
func TestAgricultureSystem_Deterministic(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Set up a fertile tile
	tileIdx := 5*10 + 5
	mapGrid.Tiles[tileIdx] = engine.TileData{
		Elevation:   100,
		Moisture:    200, // Very wet initially
		Temperature: 150, // Temperate
		BiomeID:     engine.BiomeTemperateRainForest,
	}
	mapGrid.Resources[tileIdx].FoodValue = 50

	agSystem := NewAgricultureSystem(&world, mapGrid)

	// 1. Create Employer (Village)
	employer := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	employerIdent := (*components.Identity)(world.Get(employer, ecs.ComponentID[components.Identity](&world)))
	employerIdent.ID = 101

	employerStorage := (*components.StorageComponent)(world.Get(employer, ecs.ComponentID[components.StorageComponent](&world)))
	employerStorage.Food = 0

	// 2. Create Farmer NPC
	farmer := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
	)

	farmerPos := (*components.Position)(world.Get(farmer, ecs.ComponentID[components.Position](&world)))
	farmerPos.X = 5.0
	farmerPos.Y = 5.0

	farmerJob := (*components.JobComponent)(world.Get(farmer, ecs.ComponentID[components.JobComponent](&world)))
	farmerJob.JobID = components.JobFarmer
	farmerJob.EmployerID = 101

	// Fast forward 60 ticks to trigger a harvest
	agSystem.tickStamp = 59
	agSystem.Update(&world)

	// Assertions
	if employerStorage.Food != 1 {
		t.Fatalf("Expected 1 Food in storage, got %d", employerStorage.Food)
	}

	if mapGrid.Resources[tileIdx].FoodValue != 49 {
		t.Fatalf("Expected FoodValue to decrease to 49, got %d", mapGrid.Resources[tileIdx].FoodValue)
	}

	if mapGrid.Tiles[tileIdx].Moisture != 199 {
		t.Fatalf("Expected Moisture to decrease to 199, got %d", mapGrid.Tiles[tileIdx].Moisture)
	}

	// Trigger many harvests to simulate desertification
	for i := 0; i < 150*60; i++ {
		agSystem.Update(&world)
	}

	// Because food only had 49 left, it will harvest 49 more times, dropping moisture by 49.
	// Initial moisture 200 - 50 = 150.
	// 150 moisture with 100 elevation and 150 temp -> BiomeTemperateDeciduousForest (moisture < 200).
	// To test desertification better, let's reset food to a very high amount so we cross the < 85 threshold (TemperateDesert).

	mapGrid.Resources[tileIdx].FoodValue = 200
	mapGrid.Tiles[tileIdx].Moisture = 150

	for i := 0; i < 100*60; i++ {
		agSystem.Update(&world)
	}

	// Now Moisture should have dropped by another 100, down to 50.
	// Temp 150, Moisture 50 -> BiomeTemperateDesert (moisture < 85).

	if mapGrid.Tiles[tileIdx].Moisture != 50 {
		t.Fatalf("Expected Moisture to decrease to 50, got %d", mapGrid.Tiles[tileIdx].Moisture)
	}

	if mapGrid.Tiles[tileIdx].BiomeID != engine.BiomeTemperateDesert {
		t.Fatalf("Expected biome to turn into TemperateDesert (%d), got %d", engine.BiomeTemperateDesert, mapGrid.Tiles[tileIdx].BiomeID)
	}
}
