package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)

	// Create an active war between Attacker (CountryID 1) and Defender (CountryID 2)
	attackerCapital := world.NewEntity()
	world.Add(attackerCapital, affID, warID)
	affAttacker := (*components.Affiliation)(world.Get(attackerCapital, affID))
	affAttacker.CountryID = 1
	warTracker := (*components.WarTrackerComponent)(world.Get(attackerCapital, warID))
	warTracker.Active = true
	warTracker.TargetCountryID = 2

	// Create a target village for Country 2
	village := world.NewEntity()
	world.Add(village, posID, affID, villageID, marketID, loyaltyID, popID)

	villagePos := (*components.Position)(world.Get(village, posID))
	villagePos.X = 100.0
	villagePos.Y = 100.0

	villageAff := (*components.Affiliation)(world.Get(village, affID))
	villageAff.CountryID = 2

	villageMarket := (*components.MarketComponent)(world.Get(village, marketID))
	villageMarket.FoodPrice = 10.0

	villageLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	villageLoyalty.Value = 100

	villagePop := (*components.PopulationComponent)(world.Get(village, popID))
	villagePop.Count = 10 // Need at least 10% * 10 = 1, and our threshold is >= 3 hostiles.

	// Spawn 3 hostile NPCs from Country 1 surrounding the village
	for i := 0; i < 3; i++ {
		npc := world.NewEntity()
		world.Add(npc, posID, affID)

		npcPos := (*components.Position)(world.Get(npc, posID))
		npcPos.X = 100.0 + float32(i) // Within distSq < 25
		npcPos.Y = 100.0

		npcAff := (*components.Affiliation)(world.Get(npc, affID))
		npcAff.CountryID = 1
	}

	// Create System
	siegeSys := NewSiegeSystem(&world)

	// Tick 1: System should detect the 3 hostile NPCs, outnumbering the village, and add SiegeMarker
	siegeSys.Update(&world)

	if !world.Has(village, siegeID) {
		t.Fatalf("Expected village to have SiegeMarker after tick 1")
	}

	// Verify market price and loyalty changed
	villageMarket = (*components.MarketComponent)(world.Get(village, marketID))
	if villageMarket.FoodPrice != 10.5 {
		t.Errorf("Expected FoodPrice to be 10.5, got %f", villageMarket.FoodPrice)
	}

	villageLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	if villageLoyalty.Value != 99 {
		t.Errorf("Expected Loyalty to be 99, got %d", villageLoyalty.Value)
	}

	// Tick 2: Effect continues
	siegeSys.Update(&world)

	villageMarket = (*components.MarketComponent)(world.Get(village, marketID))
	if villageMarket.FoodPrice != 11.0 {
		t.Errorf("Expected FoodPrice to be 11.0, got %f", villageMarket.FoodPrice)
	}

	villageLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	if villageLoyalty.Value != 98 {
		t.Errorf("Expected Loyalty to be 98, got %d", villageLoyalty.Value)
	}

	// Now move the hostiles away
	npcQuery := world.Query(siegeSys.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(posID))
		pos.X = 0.0
		pos.Y = 0.0 // Far away
	}

	// Tick 3: System should remove SiegeMarker because hostiles are gone
	siegeSys.Update(&world)

	if world.Has(village, siegeID) {
		t.Fatalf("Expected village to lose SiegeMarker after hostiles left")
	}

	// Price and loyalty should NOT have changed on Tick 3 because the marker was removed
	// (or at most changed one last time during the remove tick depending on exact logic flow,
	// but based on our Update loop, processing happens AFTER marker addition, and existing sieges
	// get processed. Wait, the processExistingSieges happens at the end. Since the marker was
	// removed in step 3, processExistingSieges won't see it.)
	villageMarket = (*components.MarketComponent)(world.Get(village, marketID))
	if villageMarket.FoodPrice != 11.0 {
		t.Errorf("Expected FoodPrice to remain 11.0 after siege ended, got %f", villageMarket.FoodPrice)
	}
}
