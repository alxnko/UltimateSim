package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 67 - The Subterranean Mining Engine
// This integration test verifies the Butterfly Effect:
// 1. Geography: Resources are physically extracted from the MapGrid.
// 2. Biology: The miner's stamina is reduced (grueling physical labor).
// 3. Logistics: The extracted resources are deposited in the employer's storage.
// 4. Macroeconomics: The influx of new materials natively crashes the MarketComponent prices.
func TestMiningSystem_Integration(t *testing.T) {
	// 1. Setup World and Seed RNG
	engine.InitializeRNG([32]byte{1})
	world := ecs.NewWorld()

	// 2. Setup MapGrid with rich subterranean resources
	mapGrid := engine.NewMapGrid(10, 10)
	targetIdx := 5*10 + 5
	mapGrid.Resources[targetIdx].StoneValue = 100
	mapGrid.Resources[targetIdx].IronValue = 50

	// 3. Register components
	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	// 4. Setup Employer (Village)
	employerEnt := world.NewEntity(villageID, storageID, popID, marketID, identID)
	employerIdent := (*components.Identity)(world.Get(employerEnt, identID))
	employerIdent.ID = 1001

	employerStorage := (*components.StorageComponent)(world.Get(employerEnt, storageID))
	employerStorage.Stone = 0
	employerStorage.Iron = 0
	employerStorage.Food = 100 // Prevent divide by zero in price discovery

	employerPop := (*components.PopulationComponent)(world.Get(employerEnt, popID))
	employerPop.Count = 100 // Baseline demand

	employerMarket := (*components.MarketComponent)(world.Get(employerEnt, marketID))

	// 5. Setup Initial Prices (High prices due to 0 supply)
	pdSystem := NewPriceDiscoverySystem()
	pdSystem.Update(&world)

	initialStonePrice := employerMarket.StonePrice
	initialIronPrice := employerMarket.IronPrice

	if initialStonePrice <= 0 || initialIronPrice <= 0 {
		t.Fatalf("Expected initial prices to be > 0, got Stone: %f, Iron: %f", initialStonePrice, initialIronPrice)
	}

	// 6. Setup Miner NPC
	minerEnt := world.NewEntity(npcID, posID, jobID, vitalsID)
	minerPos := (*components.Position)(world.Get(minerEnt, posID))
	minerPos.X = 5
	minerPos.Y = 5

	minerJob := (*components.JobComponent)(world.Get(minerEnt, jobID))
	minerJob.JobID = components.JobMiner
	minerJob.EmployerID = 1001

	minerVitals := (*components.VitalsComponent)(world.Get(minerEnt, vitalsID))
	minerVitals.Stamina = 100.0

	// 7. Initialize MiningSystem
	miningSys := NewMiningSystem(&world, mapGrid)

	// 8. Tick the system to trigger harvest (requires % 60 == 0)
	for i := 0; i < 60; i++ {
		miningSys.Update()
	}

	// 9. Assert Geography (MapGrid extraction)
	if mapGrid.Resources[targetIdx].StoneValue >= 100 {
		t.Errorf("Expected StoneValue to decrease, got %d", mapGrid.Resources[targetIdx].StoneValue)
	}
	if mapGrid.Resources[targetIdx].IronValue >= 50 {
		t.Errorf("Expected IronValue to decrease, got %d", mapGrid.Resources[targetIdx].IronValue)
	}

	// 10. Assert Logistics (Employer Storage)
	if employerStorage.Stone <= 0 {
		t.Errorf("Expected Employer Storage to have Stone, got %d", employerStorage.Stone)
	}
	if employerStorage.Iron <= 0 {
		t.Errorf("Expected Employer Storage to have Iron, got %d", employerStorage.Iron)
	}

	// 11. Assert Biology (Stamina drain)
	if minerVitals.Stamina >= 100.0 {
		t.Errorf("Expected Miner Stamina to decrease, got %f", minerVitals.Stamina)
	}

	// 12. Assert Macroeconomics (Price Crash)
	pdSystem.Update(&world)

	newStonePrice := employerMarket.StonePrice
	newIronPrice := employerMarket.IronPrice

	if newStonePrice >= initialStonePrice {
		t.Errorf("Expected StonePrice to crash (initial: %f, new: %f)", initialStonePrice, newStonePrice)
	}
	if newIronPrice >= initialIronPrice {
		t.Errorf("Expected IronPrice to crash (initial: %f, new: %f)", initialIronPrice, newIronPrice)
	}
}
