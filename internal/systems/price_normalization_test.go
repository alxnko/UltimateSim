package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestPriceNormalizationSystem_Deterministic verifies Phase 16.2 DOD constraints
func TestPriceNormalizationSystem_Deterministic(t *testing.T) {
	world := ecs.NewWorld()

	sys := NewPriceNormalizationSystem(&world)

	// Component IDs
	unionEntityID := ecs.ComponentID[components.UnionEntity](&world)
	unionCompID := ecs.ComponentID[components.UnionComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)

	// 1. Create two Village entities that will belong to the same union
	village1 := world.NewEntity(villageID, affilID, marketID)
	v1Affil := (*components.Affiliation)(world.Get(village1, affilID))
	v1Affil.CityID = 1 // Acts as City 1
	v1Market := (*components.MarketComponent)(world.Get(village1, marketID))
	v1Market.FoodPrice = 2.0
	v1Market.WoodPrice = 4.0

	village2 := world.NewEntity(villageID, affilID, marketID)
	v2Affil := (*components.Affiliation)(world.Get(village2, affilID))
	v2Affil.CityID = 2 // Acts as City 2
	v2Market := (*components.MarketComponent)(world.Get(village2, marketID))
	v2Market.FoodPrice = 6.0
	v2Market.WoodPrice = 8.0

	// 2. Create a Currency Union connecting the two cities
	unionEntity := world.NewEntity(unionEntityID, unionCompID)
	unionComp := (*components.UnionComponent)(world.Get(unionEntity, unionCompID))
	unionComp.UnionType = components.UnionCurrency
	unionComp.MemberIDs = []uint32{1, 2}

	// 3. Run the system
	sys.Update()

	// 4. Verify mathematical average is applied
	// Food: (2.0 + 6.0) / 2 = 4.0
	// Wood: (4.0 + 8.0) / 2 = 6.0

	if v1Market.FoodPrice != 4.0 {
		t.Fatalf("Expected Village 1 FoodPrice to be 4.0, got %v", v1Market.FoodPrice)
	}

	if v2Market.FoodPrice != 4.0 {
		t.Fatalf("Expected Village 2 FoodPrice to be 4.0, got %v", v2Market.FoodPrice)
	}

	if v1Market.WoodPrice != 6.0 {
		t.Fatalf("Expected Village 1 WoodPrice to be 6.0, got %v", v1Market.WoodPrice)
	}

	if v2Market.WoodPrice != 6.0 {
		t.Fatalf("Expected Village 2 WoodPrice to be 6.0, got %v", v2Market.WoodPrice)
	}

	// 5. Deterministic verification loop check
	sys.Update()

	if v1Market.FoodPrice != 4.0 || v2Market.FoodPrice != 4.0 {
		t.Fatalf("Expected stable prices across repeated ticks, got %v and %v", v1Market.FoodPrice, v2Market.FoodPrice)
	}
}
