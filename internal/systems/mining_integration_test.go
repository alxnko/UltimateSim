package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 67 - The Subterranean Mining Engine (Butterfly Effect E2E)
// Tests that a Miner successfully drains resources from the MapGrid, adds them to their Employer's Storage,
// drains their own Stamina (bridging to Pain), and dynamically lowers the MarketPrice of extracted goods.
func TestMiningSystem_Integration(t *testing.T) {
	// 1. Setup World and MapGrid
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)
	engine.InitializeRNG([32]byte{1})

	// Add Iron and Stone to MapGrid at (5, 5)
	mapGrid.Resources[5*10+5].IronValue = 10
	mapGrid.Resources[5*10+5].StoneValue = 10

	sys := NewMiningSystem(&world, mapGrid)

	// 2. Spawn Employer (Village)
	villageID := ecs.ComponentID[components.Village](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)

	empEntity := world.NewEntity(villageID, identID, storageID, marketID)
	empIdent := (*components.Identity)(world.Get(empEntity, identID))
	empIdent.ID = 100

	empStorage := (*components.StorageComponent)(world.Get(empEntity, storageID))
	empStorage.Iron = 0
	empStorage.Stone = 0

	empMarket := (*components.MarketComponent)(world.Get(empEntity, marketID))
	empMarket.IronPrice = 5.0
	empMarket.StonePrice = 3.0

	// 3. Spawn physical MineComponent for the Employer
	mineID := ecs.ComponentID[components.MineComponent](&world)
	mineEntity := world.NewEntity(mineID)
	mine := (*components.MineComponent)(world.Get(mineEntity, mineID))
	mine.EmployerID = 100
	mine.X = 5.0
	mine.Y = 5.0
	mine.Depth = 0 // No cave-ins for this test

	// 4. Spawn Miner NPC
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	minerEntity := world.NewEntity(npcID, jobID, posID, vitalsID)
	minerJob := (*components.JobComponent)(world.Get(minerEntity, jobID))
	minerJob.JobID = components.JobMiner
	minerJob.EmployerID = 100 // Works for our Village

	minerPos := (*components.Position)(world.Get(minerEntity, posID))
	minerPos.X = 5.0
	minerPos.Y = 5.0 // exactly at the mine

	minerVitals := (*components.VitalsComponent)(world.Get(minerEntity, vitalsID))
	minerVitals.Stamina = 5.0 // Will drain in 3 ticks (2 per tick)
	minerVitals.Pain = 0.0
	minerVitals.Blood = 100.0

	// 5. Run the simulation for a few ticks
	// Tick 1
	sys.Update(&world)

	// Stamina: 5.0 -> 3.0
	// MapGrid Iron: 10 -> 9
	// Employer Iron: 0 -> 1
	// Employer Market Iron Price: 5.0 -> 4.95

	if mapGrid.Resources[55].IronValue != 9 {
		t.Errorf("Expected MapGrid Iron to be 9, got %d", mapGrid.Resources[55].IronValue)
	}
	if empStorage.Iron != 1 {
		t.Errorf("Expected Employer Storage Iron to be 1, got %d", empStorage.Iron)
	}
	if minerVitals.Stamina != 3.0 {
		t.Errorf("Expected Miner Stamina to be 3.0, got %f", minerVitals.Stamina)
	}
	if empMarket.IronPrice > 4.96 { // floating point drift
		t.Errorf("Expected IronPrice to drop, got %f", empMarket.IronPrice)
	}

	// Tick 2
	sys.Update(&world)
	// Stamina: 3.0 -> 1.0

	// Tick 3
	sys.Update(&world)
	// Stamina: 1.0 -> 0.0, Pain -> 5.0 (Miner pushes through exhaustion)

	if minerVitals.Stamina != 0.0 {
		t.Errorf("Expected Miner Stamina to be exhausted (0.0), got %f", minerVitals.Stamina)
	}
	if minerVitals.Pain != 5.0 {
		t.Errorf("Expected Miner Pain to increase to 5.0 due to exhaustion, got %f", minerVitals.Pain)
	}

	// After 3 ticks, 3 Iron and 3 Stone should be extracted
	if empStorage.Iron != 3 {
		t.Errorf("Expected Employer Storage Iron to be 3, got %d", empStorage.Iron)
	}
}
