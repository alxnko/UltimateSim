package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestEconomicBlocSystem_Deterministic verifies Phase 16.2 DOD constraints
func TestEconomicBlocSystem_Deterministic(t *testing.T) {
	world := ecs.NewWorld()

	sys := NewEconomicBlocSystem(&world)

	// Component IDs
	unionEntityID := ecs.ComponentID[components.UnionEntity](&world)
	unionCompID := ecs.ComponentID[components.UnionComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)

	// 1. Create Starving Village
	village1 := world.NewEntity(villageID, affilID, marketID, storageID)
	v1Affil := (*components.Affiliation)(world.Get(village1, affilID))
	v1Affil.CityID = 1 // Acts as City 1
	v1Market := (*components.MarketComponent)(world.Get(village1, marketID))
	v1Market.FoodPrice = 15.0 // Famine threshold
	v1Storage := (*components.StorageComponent)(world.Get(village1, storageID))
	v1Storage.Food = 0

	// 2. Create Surplus Village
	village2 := world.NewEntity(villageID, affilID, marketID, storageID)
	v2Affil := (*components.Affiliation)(world.Get(village2, affilID))
	v2Affil.CityID = 2 // Acts as City 2
	v2Market := (*components.MarketComponent)(world.Get(village2, marketID))
	v2Market.FoodPrice = 2.0 // Normal price
	v2Storage := (*components.StorageComponent)(world.Get(village2, storageID))
	v2Storage.Food = 200

	// 3. Create Economic Bloc connecting the two cities
	unionEntity := world.NewEntity(unionEntityID, unionCompID)
	unionComp := (*components.UnionComponent)(world.Get(unionEntity, unionCompID))
	unionComp.UnionType = components.UnionEconomicBloc
	unionComp.MemberIDs = []uint32{1, 2}

	// 4. Run the system
	sys.Update()

	// 5. Verify mathematical transfer of surplus
	// Village 2 has 200 Food. Transfer should be 100 (half the surplus).

	if v1Storage.Food != 100 {
		t.Fatalf("Expected Village 1 Food to be 100, got %v", v1Storage.Food)
	}

	if v2Storage.Food != 100 {
		t.Fatalf("Expected Village 2 Food to be 100, got %v", v2Storage.Food)
	}

	// 6. Deterministic verification loop check
	// Should not transfer again because Food is now balanced and highest surplus is 100.
	sys.Update()

	if v1Storage.Food != 100 || v2Storage.Food != 100 {
		t.Fatalf("Expected stable storage after famine resolution, got %v and %v", v1Storage.Food, v2Storage.Food)
	}
}
