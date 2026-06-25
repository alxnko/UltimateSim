package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 67: The Subterranean Mining Engine Integration Test

func TestMiningSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Add resources to map grid at (2, 2)
	idx := 2*mapGrid.Width + 2
	mapGrid.Resources[idx].IronValue = 10
	mapGrid.Resources[idx].StoneValue = 10
	mapGrid.Tiles[idx].Elevation = 5
	mapGrid.Tiles[idx].Moisture = 5
	mapGrid.Tiles[idx].Temperature = 5

	// Create Employer (Village)
	employerEntity := world.NewEntity()
	employerID := uint64(100)
	idID := ecs.ComponentID[components.Identity](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)

	world.Add(employerEntity, idID, storageID, marketID, villageID)

	// Set Employer components
	ident := (*components.Identity)(world.Get(employerEntity, idID))
	ident.ID = employerID

	storage := (*components.StorageComponent)(world.Get(employerEntity, storageID))
	storage.Iron = 0
	storage.Stone = 0

	market := (*components.MarketComponent)(world.Get(employerEntity, marketID))
	market.IronPrice = 5.0
	market.StonePrice = 3.0

	// Create Miner (NPC)
	minerEntity := world.NewEntity()
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	world.Add(minerEntity, npcID, jobID, posID, vitalsID)

	// Set Miner components
	job := (*components.JobComponent)(world.Get(minerEntity, jobID))
	job.JobID = components.JobMiner
	job.EmployerID = employerID

	pos := (*components.Position)(world.Get(minerEntity, posID))
	pos.X = 2.0
	pos.Y = 2.0

	vitals := (*components.VitalsComponent)(world.Get(minerEntity, vitalsID))
	vitals.Stamina = 100.0

	// Initialize System
	miningSystem := NewMiningSystem(&world, mapGrid)

	// Run system for 10 ticks
	for i := 0; i < 10; i++ {
		miningSystem.Update(&world)
	}

	// Verify the Butterfly Effect

	// 1. Geography: Resources should be extracted, Elevation should be reduced
	// Extracted 10 Iron
	if mapGrid.Resources[idx].IronValue != 0 {
		t.Errorf("Expected IronValue to be 0, got %d", mapGrid.Resources[idx].IronValue)
	}
	// After Iron is 0, we'll continue extracting but test doesn't check since we stop at 10 ticks.

	// Since tickCounter increments up to 10, elevation drops once at tick 10
	if mapGrid.Tiles[idx].Elevation != 4 {
		t.Errorf("Expected Elevation to be 4, got %d", mapGrid.Tiles[idx].Elevation)
	}

	// 2. Biology: Stamina should be drained (10 extractions * 5.0 = 50.0)
	if vitals.Stamina != 50.0 {
		t.Errorf("Expected Stamina to be 50.0, got %f", vitals.Stamina)
	}

	// 3. Economy: Employer Storage should increase, Market Prices should lower
	if storage.Iron != 10 {
		t.Errorf("Expected Employer Storage Iron to be 10, got %d", storage.Iron)
	}

	expectedIronPrice := float32(5.0) - float32(0.01*10)
	// Compare with tolerance due to float32
	if market.IronPrice < expectedIronPrice-0.001 || market.IronPrice > expectedIronPrice+0.001 {
		t.Errorf("Expected Market IronPrice to be %f, got %f", expectedIronPrice, market.IronPrice)
	}
}
