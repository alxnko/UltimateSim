package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 13.1: Local Price Discovery (Market Logic) E2E Test

func TestPriceDiscoverySystem_Determinism(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	ecs.ComponentID[components.Village](&world)
	posID := ecs.ComponentID[components.Position](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)

	// Create test Village
	e := world.NewEntity(ecs.ComponentID[components.Village](&world), posID, storageID, popID, marketID)

	storage := (*components.StorageComponent)(world.Get(e, storageID))
	storage.Food = 50 // Supply
	storage.Wood = 100
	storage.Stone = 20
	storage.Iron = 0

	pop := (*components.PopulationComponent)(world.Get(e, popID))
	pop.Count = 10 // Demand multiplier

	// Run system
	system := NewPriceDiscoverySystem()
	system.Update(&world)

	market := (*components.MarketComponent)(world.Get(e, marketID))

	// Predict Math:
	// Food: Demand = 10 * 10 = 100. Supply = 50. Price = 100 / 51 = 1.9607843
	expectedFoodPrice := float32(100.0) / float32(51.0)
	if market.FoodPrice != expectedFoodPrice {
		t.Errorf("Expected FoodPrice %f, got %f", expectedFoodPrice, market.FoodPrice)
	}

	// Wood: Demand = 10 * 5 = 50. Supply = 100. Price = 50 / 101 = 0.4950495
	expectedWoodPrice := float32(50.0) / float32(101.0)
	if market.WoodPrice != expectedWoodPrice {
		t.Errorf("Expected WoodPrice %f, got %f", expectedWoodPrice, market.WoodPrice)
	}

	// Stone: Demand = 10 * 2 = 20. Supply = 20. Price = 20 / 21 = 0.95238096
	expectedStonePrice := float32(20.0) / float32(21.0)
	if market.StonePrice != expectedStonePrice {
		t.Errorf("Expected StonePrice %f, got %f", expectedStonePrice, market.StonePrice)
	}

	// Iron: Demand = 10 * 1 = 10. Supply = 0. Price = 10 / 1 = 10.0
	expectedIronPrice := float32(10.0) / float32(1.0)
	if market.IronPrice != expectedIronPrice {
		t.Errorf("Expected IronPrice %f, got %f", expectedIronPrice, market.IronPrice)
	}
}
