package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestMiningSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Set up grid tile at (5, 5) with resources and elevation
	idx := 5*mapGrid.Width + 5
	mapGrid.Resources[idx].IronValue = 10
	mapGrid.Tiles[idx].Elevation = 50

	sys := NewMiningSystem(&world, mapGrid)

	// Setup Employer (Village)
	employer := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.Village](&world),
	)

	empIdent := (*components.Identity)(world.Get(employer, ecs.ComponentID[components.Identity](&world)))
	empIdent.ID = 100

	empMarket := (*components.MarketComponent)(world.Get(employer, ecs.ComponentID[components.MarketComponent](&world)))
	empMarket.IronPrice = 50.0

	// Setup Miner (NPC)
	miner := world.NewEntity(
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
	)

	minerJob := (*components.JobComponent)(world.Get(miner, ecs.ComponentID[components.JobComponent](&world)))
	minerJob.JobID = components.JobMiner
	minerJob.EmployerID = 100

	minerPos := (*components.Position)(world.Get(miner, ecs.ComponentID[components.Position](&world)))
	minerPos.X = 5.0
	minerPos.Y = 5.0

	minerVitals := (*components.VitalsComponent)(world.Get(miner, ecs.ComponentID[components.VitalsComponent](&world)))
	minerVitals.Stamina = 100.0

	// Run system
	sys.Update(&world)

	// Verifications (Butterfly Effect check)

	// 1. Biology (Stamina drained)
	if minerVitals.Stamina >= 100.0 {
		t.Errorf("Expected Stamina to drop below 100.0, got %f", minerVitals.Stamina)
	}

	// 2. Geography (Resource extracted)
	if mapGrid.Resources[idx].IronValue != 9 {
		t.Errorf("Expected IronValue to be 9, got %d", mapGrid.Resources[idx].IronValue)
	}

	// 3. Economy (Resource stored & Price dropped)
	empStorage := (*components.StorageComponent)(world.Get(employer, ecs.ComponentID[components.StorageComponent](&world)))
	if empStorage.Iron != 1 {
		t.Errorf("Expected Employer Storage Iron to be 1, got %d", empStorage.Iron)
	}
	if empMarket.IronPrice >= 50.0 {
		t.Errorf("Expected IronPrice to drop below 50.0 due to saturation, got %f", empMarket.IronPrice)
	}

	// 4. Entropy (Elevation reduced)
	if mapGrid.Tiles[idx].Elevation != 49 {
		t.Errorf("Expected tile Elevation to be 49, got %d", mapGrid.Tiles[idx].Elevation)
	}
}
