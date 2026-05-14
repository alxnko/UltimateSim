package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)

	// Create Attacker Capital (Country 1)
	attackerCap := world.NewEntity(capID, warCompID, affilID)
	warComp := (*components.WarTrackerComponent)(world.Get(attackerCap, warCompID))
	warComp.TargetCountryID = 2 // Attacking Country 2
	warComp.Active = true
	attAffil := (*components.Affiliation)(world.Get(attackerCap, affilID))
	attAffil.CountryID = 1

	// Create Defender Village (Country 2)
	defVillage := world.NewEntity(villageID, posID, popID, affilID, marketID, loyaltyID)
	defPos := (*components.Position)(world.Get(defVillage, posID))
	defPos.X, defPos.Y = 100.0, 100.0
	defPop := (*components.PopulationComponent)(world.Get(defVillage, popID))
	defPop.Count = 2
	defAffil := (*components.Affiliation)(world.Get(defVillage, affilID))
	defAffil.CountryID = 2
	defMarket := (*components.MarketComponent)(world.Get(defVillage, marketID))
	defMarket.FoodPrice = 10.0
	defLoyalty := (*components.LoyaltyComponent)(world.Get(defVillage, loyaltyID))
	defLoyalty.Value = 50

	// Create 3 Hostile NPCs (Country 1) surrounding the village
	for i := 0; i < 3; i++ {
		npc := world.NewEntity(npcID, posID, affilID)
		npcPos := (*components.Position)(world.Get(npc, posID))
		npcPos.X, npcPos.Y = 101.0, 101.0 // Within radius 5.0 (distSq <= 25.0)
		npcAffil := (*components.Affiliation)(world.Get(npc, affilID))
		npcAffil.CountryID = 1
	}

	// Create 1 irrelevant NPC (Country 3) near the village
	npc3 := world.NewEntity(npcID, posID, affilID)
	npcPos3 := (*components.Position)(world.Get(npc3, posID))
	npcPos3.X, npcPos3.Y = 101.0, 101.0
	npcAffil3 := (*components.Affiliation)(world.Get(npc3, affilID))
	npcAffil3.CountryID = 3

	siegeSys := systems.NewSiegeSystem(&world)

	// Update loop for 30 ticks (since it processes every 10 ticks)
	for i := 1; i <= 30; i++ {
		siegeSys.Update(&world)
	}

	// Verify the SiegeMarker was applied
	if !world.Has(defVillage, siegeID) {
		t.Fatalf("Expected Defender Village to have SiegeMarker applied")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(defVillage, siegeID))
	if siegeMarker.BesiegerCountryID != 1 {
		t.Errorf("Expected BesiegerCountryID to be 1, got %d", siegeMarker.BesiegerCountryID)
	}

	// Verify consequences: Food Price spiked, Loyalty drained
	// At tick 10: marker added
	// At tick 20: marker processes (+5 FoodPrice, -1 Loyalty)
	// At tick 30: marker processes (+5 FoodPrice, -1 Loyalty)
	defMarketAfter := (*components.MarketComponent)(world.Get(defVillage, marketID))
	if defMarketAfter.FoodPrice <= 10.0 {
		t.Errorf("Expected FoodPrice to spike above 10.0, got %f", defMarketAfter.FoodPrice)
	}

	defLoyaltyAfter := (*components.LoyaltyComponent)(world.Get(defVillage, loyaltyID))
	if defLoyaltyAfter.Value >= 50 {
		t.Errorf("Expected Loyalty to drain below 50, got %d", defLoyaltyAfter.Value)
	}

	// Now move attackers away and verify the siege breaks
	npcQuery := world.Query(ecs.All(npcID, posID, affilID))
	for npcQuery.Next() {
		affil := (*components.Affiliation)(npcQuery.Get(affilID))
		if affil.CountryID == 1 {
			pos := (*components.Position)(npcQuery.Get(posID))
			pos.X, pos.Y = 0.0, 0.0 // Far away
		}
	}

	// Update another 10 ticks
	for i := 1; i <= 10; i++ {
		siegeSys.Update(&world)
	}

	if world.Has(defVillage, siegeID) {
		t.Fatalf("Expected Defender Village to have SiegeMarker removed after attackers left")
	}
}
