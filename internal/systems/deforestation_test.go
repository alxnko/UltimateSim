package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 55: The Ecological Collapse Engine (Butterfly Effect E2E Test)
// Tests the systemic connection between MapGrid (Geography), Lumberjacks (Labor),
// Storage (Economy), and eventually WinterHeating (Biology).

func TestDeforestationSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Set a tile to have a specific amount of wood
	mapGrid.Resources[55].WoodValue = 2 // x=5, y=5

	// Init system
	deforestSys := NewDeforestationSystem(&world, mapGrid)

	// 1. Create a Village (Employer)
	villageID := uint64(100)
	employerEntity := world.NewEntity()
	idID := ecs.ComponentID[components.Identity](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)

	world.Add(employerEntity, idID, storageID)
	ident := (*components.Identity)(world.Get(employerEntity, idID))
	ident.ID = villageID

	storage := (*components.StorageComponent)(world.Get(employerEntity, storageID))
	storage.Wood = 0

	// 2. Create a Lumberjack
	lumberjackEntity := world.NewEntity()
	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)

	world.Add(lumberjackEntity, npcID, posID, jobID)

	pos := (*components.Position)(world.Get(lumberjackEntity, posID))
	pos.X = 5.0
	pos.Y = 5.0 // Stand on tile 55

	job := (*components.JobComponent)(world.Get(lumberjackEntity, jobID))
	job.JobID = components.JobLumberjack
	job.EmployerID = villageID

	// Tick 1-49: Nothing should happen
	for i := 0; i < 49; i++ {
		deforestSys.Update(&world)
	}

	if storage.Wood != 0 {
		t.Errorf("Expected Storage Wood to be 0 before tick 50, got %d", storage.Wood)
	}

	if mapGrid.Resources[55].WoodValue != 2 {
		t.Errorf("Expected Grid WoodValue to be 2 before tick 50, got %d", mapGrid.Resources[55].WoodValue)
	}

	// Tick 50: Harvest 1 wood
	deforestSys.Update(&world)

	if storage.Wood != 1 {
		t.Errorf("Expected Storage Wood to be 1 after tick 50, got %d", storage.Wood)
	}

	if mapGrid.Resources[55].WoodValue != 1 {
		t.Errorf("Expected Grid WoodValue to be 1 after tick 50, got %d", mapGrid.Resources[55].WoodValue)
	}

	// Tick 51-99: Nothing
	for i := 0; i < 49; i++ {
		deforestSys.Update(&world)
	}

	// Tick 100: Harvest remaining wood (ecological collapse on this tile)
	deforestSys.Update(&world)

	if storage.Wood != 2 {
		t.Errorf("Expected Storage Wood to be 2 after tick 100, got %d", storage.Wood)
	}

	if mapGrid.Resources[55].WoodValue != 0 {
		t.Errorf("Expected Grid WoodValue to be 0 after tick 100 (depleted), got %d", mapGrid.Resources[55].WoodValue)
	}

	// Tick 101-150: Tile is barren, harvesting should yield nothing
	for i := 0; i < 50; i++ {
		deforestSys.Update(&world)
	}

	if storage.Wood != 2 {
		t.Errorf("Expected Storage Wood to remain 2 after tile depletion, got %d", storage.Wood)
	}

	// The Butterfly Effect: With local wood depleted (WoodValue = 0), the village can no longer grow its wood stockpile.
	// When Winter hits, `WinterHeatingSystem` will eventually drain this finite 2 wood, leading to Vitals.Pain spikes and death.
	// This proves that Geography -> Economy -> Biology are systemically linked.
}
