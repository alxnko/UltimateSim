package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 67 - The Subterranean Mining Engine (MiningSystem)
func TestMiningSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Set up a resource-rich mountain tile
	tileIdx := 5*10 + 5
	mapGrid.Tiles[tileIdx] = engine.TileData{
		Elevation:   200,
		Moisture:    100,
		Temperature: 100,
		BiomeID:     engine.BiomeMountain,
	}
	mapGrid.Resources[tileIdx].IronValue = 50
	mapGrid.Resources[tileIdx].StoneValue = 100

	miningSystem := NewMiningSystem(&world, mapGrid)

	// 1. Create Employer (Business)
	employer := world.NewEntity(
		ecs.ComponentID[components.BusinessComponent](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	employerIdent := (*components.Identity)(world.Get(employer, ecs.ComponentID[components.Identity](&world)))
	employerIdent.ID = 999

	employerStorage := (*components.StorageComponent)(world.Get(employer, ecs.ComponentID[components.StorageComponent](&world)))
	employerStorage.Iron = 0
	employerStorage.Stone = 0

	// 2. Create Miner NPC
	miner := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
	)

	minerPos := (*components.Position)(world.Get(miner, ecs.ComponentID[components.Position](&world)))
	minerPos.X = 5.0
	minerPos.Y = 5.0

	minerJob := (*components.JobComponent)(world.Get(miner, ecs.ComponentID[components.JobComponent](&world)))
	minerJob.JobID = components.JobMiner
	minerJob.EmployerID = 999

	minerVitals := (*components.VitalsComponent)(world.Get(miner, ecs.ComponentID[components.VitalsComponent](&world)))
	minerVitals.Stamina = 100.0

	// Fast forward 30 ticks to trigger a mining cycle
	miningSystem.tickStamp = 29
	miningSystem.Update(&world)

	// Assertions
	if employerStorage.Iron != 1 {
		t.Fatalf("Expected 1 Iron in storage, got %d", employerStorage.Iron)
	}

	if employerStorage.Stone != 1 {
		t.Fatalf("Expected 1 Stone in storage, got %d", employerStorage.Stone)
	}

	if mapGrid.Resources[tileIdx].IronValue != 49 {
		t.Fatalf("Expected IronValue to decrease to 49, got %d", mapGrid.Resources[tileIdx].IronValue)
	}

	if mapGrid.Resources[tileIdx].StoneValue != 99 {
		t.Fatalf("Expected StoneValue to decrease to 99, got %d", mapGrid.Resources[tileIdx].StoneValue)
	}

	if minerVitals.Stamina != 90.0 {
		t.Fatalf("Expected Stamina to drop to 90.0, got %f", minerVitals.Stamina)
	}

	if mapGrid.Tiles[tileIdx].Elevation != 199 {
		t.Fatalf("Expected Elevation to decrease to 199, got %d", mapGrid.Tiles[tileIdx].Elevation)
	}

	// Trigger many mining cycles to simulate exhaustion of stamina
	for i := 0; i < 9*30; i++ {
		miningSystem.Update(&world)
	}

	// At this point (1 + 9 = 10 total cycles), stamina should be 0 (100 - 10*10 = 0).
	if minerVitals.Stamina != 0.0 {
		t.Fatalf("Expected Stamina to drop to 0.0 after exhaustion, got %f", minerVitals.Stamina)
	}

	// Another cycle shouldn't mine anything due to 0 stamina
	ironBefore := employerStorage.Iron
	stoneBefore := employerStorage.Stone
	elevationBefore := mapGrid.Tiles[tileIdx].Elevation

	miningSystem.Update(&world)

	if employerStorage.Iron != ironBefore || employerStorage.Stone != stoneBefore {
		t.Fatalf("Expected storage to not change when exhausted. Iron: %d, Stone: %d", employerStorage.Iron, employerStorage.Stone)
	}

	if mapGrid.Tiles[tileIdx].Elevation != elevationBefore {
		t.Fatalf("Expected Elevation to not change when exhausted. Elevation: %d", mapGrid.Tiles[tileIdx].Elevation)
	}
}
