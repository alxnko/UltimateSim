package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestMiningSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.StorageComponent](&world)
	ecs.ComponentID[components.MarketComponent](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.BusinessComponent](&world)
	ecs.ComponentID[components.NPC](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.VitalsComponent](&world)

	// Create MapGrid
	mapGrid := engine.NewMapGrid(10, 10)
	idx := 5*10 + 5 // x=5, y=5
	mapGrid.Resources[idx].IronValue = 2
	mapGrid.Resources[idx].StoneValue = 1
	mapGrid.Tiles[idx].Elevation = 100

	// Create Employer (Village)
	employerEntity := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	employerIdent := (*components.Identity)(world.Get(employerEntity, ecs.ComponentID[components.Identity](&world)))
	employerIdent.ID = 12345

	employerMarket := (*components.MarketComponent)(world.Get(employerEntity, ecs.ComponentID[components.MarketComponent](&world)))
	employerMarket.IronPrice = 10.0
	employerMarket.StonePrice = 5.0

	employerStorage := (*components.StorageComponent)(world.Get(employerEntity, ecs.ComponentID[components.StorageComponent](&world)))
	employerStorage.Iron = 0
	employerStorage.Stone = 0

	// Create Miner (NPC)
	minerEntity := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
	)

	minerJob := (*components.JobComponent)(world.Get(minerEntity, ecs.ComponentID[components.JobComponent](&world)))
	minerJob.JobID = components.JobMiner
	minerJob.EmployerID = 12345

	minerPos := (*components.Position)(world.Get(minerEntity, ecs.ComponentID[components.Position](&world)))
	minerPos.X = 5.0
	minerPos.Y = 5.0

	minerVitals := (*components.VitalsComponent)(world.Get(minerEntity, ecs.ComponentID[components.VitalsComponent](&world)))
	minerVitals.Stamina = 100.0

	system := NewMiningSystem(&world, mapGrid)

	// Tick 1: Mine Iron
	system.Update(&world)

	if mapGrid.Resources[idx].IronValue != 1 {
		t.Errorf("Expected IronValue to be 1, got %d", mapGrid.Resources[idx].IronValue)
	}
	if employerStorage.Iron != 1 {
		t.Errorf("Expected Storage Iron to be 1, got %d", employerStorage.Iron)
	}
	if employerMarket.IronPrice >= 10.0 {
		t.Errorf("Expected IronPrice to drop, got %f", employerMarket.IronPrice)
	}
	if minerVitals.Stamina != 95.0 {
		t.Errorf("Expected Stamina to be 95.0, got %f", minerVitals.Stamina)
	}
	if mapGrid.Tiles[idx].Elevation != 99 {
		t.Errorf("Expected Elevation to be 99, got %d", mapGrid.Tiles[idx].Elevation)
	}

	// Tick 2: Mine Iron again
	system.Update(&world)

	if mapGrid.Resources[idx].IronValue != 0 {
		t.Errorf("Expected IronValue to be 0, got %d", mapGrid.Resources[idx].IronValue)
	}
	if employerStorage.Iron != 2 {
		t.Errorf("Expected Storage Iron to be 2, got %d", employerStorage.Iron)
	}
	if mapGrid.Tiles[idx].Elevation != 98 {
		t.Errorf("Expected Elevation to be 98, got %d", mapGrid.Tiles[idx].Elevation)
	}

	// Tick 3: Mine Stone
	system.Update(&world)

	if mapGrid.Resources[idx].StoneValue != 0 {
		t.Errorf("Expected StoneValue to be 0, got %d", mapGrid.Resources[idx].StoneValue)
	}
	if employerStorage.Stone != 1 {
		t.Errorf("Expected Storage Stone to be 1, got %d", employerStorage.Stone)
	}
	if employerMarket.StonePrice >= 5.0 {
		t.Errorf("Expected StonePrice to drop, got %f", employerMarket.StonePrice)
	}
	if mapGrid.Tiles[idx].Elevation != 97 {
		t.Errorf("Expected Elevation to be 97, got %d", mapGrid.Tiles[idx].Elevation)
	}

	// Tick 4: Depleted, no mining
	system.Update(&world)

	if minerVitals.Stamina != 85.0 {
		t.Errorf("Expected Stamina to remain 85.0, got %f", minerVitals.Stamina)
	}
	if mapGrid.Tiles[idx].Elevation != 97 {
		t.Errorf("Expected Elevation to remain 97, got %d", mapGrid.Tiles[idx].Elevation)
	}
}
