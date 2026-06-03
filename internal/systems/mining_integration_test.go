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

	// Set up a resource-rich tile
	tileIdx := 5*10 + 5
	mapGrid.Tiles[tileIdx] = engine.TileData{
		Elevation:   100,
		Moisture:    100,
		Temperature: 100,
		BiomeID:     engine.BiomeMountain,
	}
	mapGrid.Resources[tileIdx].IronValue = 50
	mapGrid.Resources[tileIdx].StoneValue = 50

	miningSystem := NewMiningSystem(&world, mapGrid)

	// 1. Create Employer (Village) with Market and Storage
	employer := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	employerIdent := (*components.Identity)(world.Get(employer, ecs.ComponentID[components.Identity](&world)))
	employerIdent.ID = 101

	employerStorage := (*components.StorageComponent)(world.Get(employer, ecs.ComponentID[components.StorageComponent](&world)))
	employerStorage.Iron = 0
	employerStorage.Stone = 0

	employerMarket := (*components.MarketComponent)(world.Get(employer, ecs.ComponentID[components.MarketComponent](&world)))
	employerMarket.IronPrice = 10.0
	employerMarket.StonePrice = 10.0

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
	minerJob.EmployerID = 101

	minerVitals := (*components.VitalsComponent)(world.Get(miner, ecs.ComponentID[components.VitalsComponent](&world)))
	minerVitals.Stamina = 100.0

	// Fast forward 30 ticks to trigger a mining extraction
	miningSystem.tickStamp = 29
	miningSystem.Update()

	// Assertions for step 1
	if employerStorage.Iron != 1 {
		t.Fatalf("Expected 1 Iron in storage, got %d", employerStorage.Iron)
	}

	if employerStorage.Stone != 1 {
		t.Fatalf("Expected 1 Stone in storage, got %d", employerStorage.Stone)
	}

	if mapGrid.Resources[tileIdx].IronValue != 49 {
		t.Fatalf("Expected IronValue to decrease to 49, got %d", mapGrid.Resources[tileIdx].IronValue)
	}

	if mapGrid.Tiles[tileIdx].Elevation != 99 {
		t.Fatalf("Expected Elevation to decrease to 99, got %d", mapGrid.Tiles[tileIdx].Elevation)
	}

	if minerVitals.Stamina != 98.0 {
		t.Fatalf("Expected Stamina to decrease to 98.0, got %f", minerVitals.Stamina)
	}

	if employerMarket.IronPrice >= 10.0 {
		t.Fatalf("Expected IronPrice to organically drop below 10.0, got %f", employerMarket.IronPrice)
	}

	// Trigger many extractions until stamina runs out or resources deplete
	for i := 0; i < 50*30; i++ {
		miningSystem.Update()
	}

	// With 100 Stamina, it drains 2 per extraction. That's 50 extractions.
	// We expect Stamina to reach 0 and stop mining.
	if minerVitals.Stamina > 5.0 {
		t.Fatalf("Expected Stamina to be drained below 5.0, got %f", minerVitals.Stamina)
	}

	// 48 extractions (including the first 1) = 48 iron and stone
	if employerStorage.Iron != 48 {
		t.Fatalf("Expected 48 Iron in storage after full stamina drain, got %d", employerStorage.Iron)
	}
	if employerStorage.Stone != 48 {
		t.Fatalf("Expected 48 Stone in storage after full stamina drain, got %d", employerStorage.Stone)
	}

	if mapGrid.Tiles[tileIdx].Elevation != 52 {
		t.Fatalf("Expected Elevation to drop to 52 due to extractions, got %d", mapGrid.Tiles[tileIdx].Elevation)
	}
}
