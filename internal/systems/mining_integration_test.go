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
	engine.InitializeRNG([32]byte{1, 2, 3}) // Deterministic seed

	// 1. Initialize MapGrid with Resources
	mapGrid := engine.NewMapGrid(10, 10)

	idx := 5*10 + 5 // Map coordinates (5, 5)
	mapGrid.Resources[idx].IronValue = 10
	mapGrid.Resources[idx].StoneValue = 10

	miningSys := NewMiningSystem(&world, mapGrid)

	// 2. Setup Employer (Village)
	villageEnt := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	villageID := (*components.Identity)(world.Get(villageEnt, ecs.ComponentID[components.Identity](&world)))
	villageID.ID = 100

	villageStorage := (*components.StorageComponent)(world.Get(villageEnt, ecs.ComponentID[components.StorageComponent](&world)))
	villageStorage.Iron = 0
	villageStorage.Stone = 0

	villageMarket := (*components.MarketComponent)(world.Get(villageEnt, ecs.ComponentID[components.MarketComponent](&world)))
	villageMarket.IronPrice = 5.0
	villageMarket.StonePrice = 5.0

	// 3. Setup Miner NPC
	minerEnt := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
	)

	minerPos := (*components.Position)(world.Get(minerEnt, ecs.ComponentID[components.Position](&world)))
	minerPos.X = 5.0
	minerPos.Y = 5.0

	minerJob := (*components.JobComponent)(world.Get(minerEnt, ecs.ComponentID[components.JobComponent](&world)))
	minerJob.JobID = components.JobMiner
	minerJob.EmployerID = 100 // Match village

	minerVitals := (*components.VitalsComponent)(world.Get(minerEnt, ecs.ComponentID[components.VitalsComponent](&world)))
	minerVitals.Stamina = 100.0

	// 4. Tick simulation enough to trigger miner extraction (every 30 ticks)
	for i := 0; i < 30; i++ {
		miningSys.Update(&world)
	}

	// 5. Assert Geography -> Biology -> Economy -> Entropy Butterfly Effect

	// Biology: Stamina should be drained by 5
	vitalsAfter := (*components.VitalsComponent)(world.Get(minerEnt, ecs.ComponentID[components.VitalsComponent](&world)))
	if vitalsAfter.Stamina != 95.0 {
		t.Errorf("Expected Stamina to be 95.0 after mining, got %f", vitalsAfter.Stamina)
	}

	// Geography: MapGrid resources should be depleted by 1
	if mapGrid.Resources[idx].IronValue != 9 {
		t.Errorf("Expected MapGrid IronValue to be 9, got %d", mapGrid.Resources[idx].IronValue)
	}
	if mapGrid.Resources[idx].StoneValue != 9 {
		t.Errorf("Expected MapGrid StoneValue to be 9, got %d", mapGrid.Resources[idx].StoneValue)
	}

	// Economy: Village Storage should increase by 1
	storageAfter := (*components.StorageComponent)(world.Get(villageEnt, ecs.ComponentID[components.StorageComponent](&world)))
	if storageAfter.Iron != 1 {
		t.Errorf("Expected Village Storage Iron to be 1, got %d", storageAfter.Iron)
	}
	if storageAfter.Stone != 1 {
		t.Errorf("Expected Village Storage Stone to be 1, got %d", storageAfter.Stone)
	}

	// Economy: Market prices should decrease by 0.1
	marketAfter := (*components.MarketComponent)(world.Get(villageEnt, ecs.ComponentID[components.MarketComponent](&world)))
	if marketAfter.IronPrice > 4.901 || marketAfter.IronPrice < 4.899 {
		t.Errorf("Expected Village Market IronPrice to be 4.9, got %f", marketAfter.IronPrice)
	}
	if marketAfter.StonePrice > 4.901 || marketAfter.StonePrice < 4.899 {
		t.Errorf("Expected Village Market StonePrice to be 4.9, got %f", marketAfter.StonePrice)
	}
}
