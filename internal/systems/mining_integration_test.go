package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// TestMiningSystem_Integration verifies the Phase 67 Subterranean Mining Engine loop:
// JobMiner -> MapGrid (Elevation/Resources decay) -> Vitals (Stamina drain) -> Market (Prices drop).
func TestMiningSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := &engine.MapGrid{
		Width:  10,
		Height: 10,
		Tiles:  make([]engine.TileData, 100),
		Resources: make([]engine.ResourceDepot, 100),
	}

	// Set up the target tile
	targetIdx := 5*10 + 5
	mapGrid.Tiles[targetIdx].Elevation = 100
	mapGrid.Resources[targetIdx].StoneValue = 10
	mapGrid.Resources[targetIdx].IronValue = 5

	// Create Employer (Village)
	employerID := uint64(1001)
	employer := world.NewEntity()
	world.Add(employer,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
	)

	idComp := (*components.Identity)(world.Get(employer, ecs.ComponentID[components.Identity](&world)))
	idComp.ID = employerID

	storage := (*components.StorageComponent)(world.Get(employer, ecs.ComponentID[components.StorageComponent](&world)))
	market := (*components.MarketComponent)(world.Get(employer, ecs.ComponentID[components.MarketComponent](&world)))
	market.StonePrice = 20.0
	market.IronPrice = 50.0

	// Create Miner (NPC)
	miner := world.NewEntity()
	world.Add(miner,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
	)

	pos := (*components.Position)(world.Get(miner, ecs.ComponentID[components.Position](&world)))
	pos.X = 5.0
	pos.Y = 5.0

	job := (*components.JobComponent)(world.Get(miner, ecs.ComponentID[components.JobComponent](&world)))
	job.JobID = components.JobMiner
	job.EmployerID = employerID

	vitals := (*components.VitalsComponent)(world.Get(miner, ecs.ComponentID[components.VitalsComponent](&world)))
	vitals.Stamina = 100.0
	vitals.Consciousness = 100.0

	system := NewMiningSystem(&world, mapGrid)

	// Tick until the mining system triggers
	for i := 0; i < 50; i++ {
		system.Update()
	}

	// Verify Geography Drain
	if mapGrid.Resources[targetIdx].StoneValue != 8 {
		t.Errorf("Expected Map StoneValue to drop to 8, got %d", mapGrid.Resources[targetIdx].StoneValue)
	}
	if mapGrid.Resources[targetIdx].IronValue != 4 {
		t.Errorf("Expected Map IronValue to drop to 4, got %d", mapGrid.Resources[targetIdx].IronValue)
	}

	// Verify Elevation Tunneling
	if mapGrid.Tiles[targetIdx].Elevation != 99 {
		t.Errorf("Expected Elevation to drop to 99, got %d", mapGrid.Tiles[targetIdx].Elevation)
	}

	// Verify Storage Gains
	if storage.Stone != 2 {
		t.Errorf("Expected Storage Stone to be 2, got %d", storage.Stone)
	}
	if storage.Iron != 1 {
		t.Errorf("Expected Storage Iron to be 1, got %d", storage.Iron)
	}

	// Verify Market Price drops
	if market.StonePrice >= 20.0 {
		t.Errorf("Expected Market StonePrice to drop below 20.0, got %f", market.StonePrice)
	}
	if market.IronPrice >= 50.0 {
		t.Errorf("Expected Market IronPrice to drop below 50.0, got %f", market.IronPrice)
	}

	// Verify Biological Drain
	if vitals.Stamina >= 100.0 {
		t.Errorf("Expected Vitals Stamina to drop below 100.0, got %f", vitals.Stamina)
	}
}
