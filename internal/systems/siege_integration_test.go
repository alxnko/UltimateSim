package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 66 - The Physical Siege Engine (SiegeSystem)
// Tests the Butterfly Effect from Geography/Combat -> Siege -> Economic collapse and Loyalty drain.
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	siegeSys := NewSiegeSystem(&world)

	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	warTrackerID := ecs.ComponentID[components.WarTrackerComponent](&world)
	siegeMarkerID := ecs.ComponentID[components.SiegeMarker](&world)

	// 1. Create Attacker Capital (Country 1) at war with Country 2
	attackerCap := world.NewEntity(affID, warTrackerID)
	attAff := (*components.Affiliation)(world.Get(attackerCap, affID))
	attAff.CountryID = 1

	war := (*components.WarTrackerComponent)(world.Get(attackerCap, warTrackerID))
	war.TargetCountryID = 2
	war.Active = true

	// 2. Create Target Village (Country 2)
	targetVillage := world.NewEntity(villageID, posID, affID, marketID, loyaltyID)
	vilPos := (*components.Position)(world.Get(targetVillage, posID))
	vilPos.X, vilPos.Y = 10.0, 10.0

	vilAff := (*components.Affiliation)(world.Get(targetVillage, affID))
	vilAff.CountryID = 2

	vilMarket := (*components.MarketComponent)(world.Get(targetVillage, marketID))
	vilMarket.FoodPrice = 5.0

	vilLoyalty := (*components.LoyaltyComponent)(world.Get(targetVillage, loyaltyID))
	vilLoyalty.Value = 100

	// 3. Create Hostile NPCs (Country 1) surrounding the village
	for i := 0; i < 5; i++ {
		hostileNPC := world.NewEntity(npcID, posID, affID)
		npcPos := (*components.Position)(world.Get(hostileNPC, posID))
		npcPos.X, npcPos.Y = 12.0, 12.0 // Very close (distSq = 8.0)

		npcAff := (*components.Affiliation)(world.Get(hostileNPC, affID))
		npcAff.CountryID = 1
	}

	// 4. Create Defending NPCs (Country 2) inside the village (fewer than hostiles)
	for i := 0; i < 2; i++ {
		defenderNPC := world.NewEntity(npcID, posID, affID)
		defPos := (*components.Position)(world.Get(defenderNPC, posID))
		defPos.X, defPos.Y = 10.0, 10.0

		defAff := (*components.Affiliation)(world.Get(defenderNPC, affID))
		defAff.CountryID = 2
	}

	// 5. Update SiegeSystem
	siegeSys.Update(&world)

	// 6. Verify Butterfly Effect
	if !world.Has(targetVillage, siegeMarkerID) {
		t.Fatalf("Expected Target Village to receive a SiegeMarker")
	}

	// Re-fetch component pointers (Arche-Go structural change invalidates previous pointers)
	vilMarket = (*components.MarketComponent)(world.Get(targetVillage, marketID))
	vilLoyalty = (*components.LoyaltyComponent)(world.Get(targetVillage, loyaltyID))

	if vilMarket.FoodPrice <= 5.0 {
		t.Errorf("Expected FoodPrice to spike due to siege, got %f", vilMarket.FoodPrice)
	}

	if vilLoyalty.Value >= 100 {
		t.Errorf("Expected Loyalty to drain due to siege, got %d", vilLoyalty.Value)
	}

	marker := (*components.SiegeMarker)(world.Get(targetVillage, siegeMarkerID))
	if marker.BesiegerCountryID != 1 {
		t.Errorf("Expected BesiegerCountryID to be 1, got %d", marker.BesiegerCountryID)
	}

	// 7. Verify Siege Broken
	// Move hostile NPCs away
	npcFilter := filter.All(npcID, posID, affID)
	npcQuery := world.Query(&npcFilter)
	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(affID))
		if aff.CountryID == 1 {
			pos := (*components.Position)(npcQuery.Get(posID))
			pos.X, pos.Y = 100.0, 100.0 // Move far away (distSq > 100.0)
		}
	}
	// Must close query manually if breaking early, but we exhausted it so auto-closed

	// Update system again
	siegeSys.Update(&world)

	if world.Has(targetVillage, siegeMarkerID) {
		t.Fatalf("Expected SiegeMarker to be removed after hostiles left")
	}
}
