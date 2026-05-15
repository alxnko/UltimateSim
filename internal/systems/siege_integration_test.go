package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewSiegeSystem(&world)

	// Register components
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	// Create Attacker Capital (Country 1) at War with Defender (Country 2)
	attackerCapital := world.NewEntity(warCompID, affID)
	attWar := (*components.WarTrackerComponent)(world.Get(attackerCapital, warCompID))
	attWar.Active = true
	attWar.TargetCountryID = 2
	attAff := (*components.Affiliation)(world.Get(attackerCapital, affID))
	attAff.CountryID = 1

	// Create Defender Village (Country 2)
	defenderVillage := world.NewEntity(villageID, affID, posID, marketID, loyaltyID)
	defAff := (*components.Affiliation)(world.Get(defenderVillage, affID))
	defAff.CountryID = 2
	defPos := (*components.Position)(world.Get(defenderVillage, posID))
	defPos.X = 10.0
	defPos.Y = 10.0
	defMarket := (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	defMarket.FoodPrice = 5.0
	defLoyalty := (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))
	defLoyalty.Value = 100

	// Run system before NPCs arrive - no siege should happen
	sys.Update(&world)
	if world.Has(defenderVillage, siegeID) {
		t.Fatalf("Village should not be under siege with no hostile NPCs")
	}

	// Create 1 friendly NPC (Defender) at the village
	friendlyNPC := world.NewEntity(npcID, affID, posID, vitalsID)
	fAff := (*components.Affiliation)(world.Get(friendlyNPC, affID))
	fAff.CountryID = 2
	fPos := (*components.Position)(world.Get(friendlyNPC, posID))
	fPos.X, fPos.Y = 11.0, 11.0
	fVitals := (*components.VitalsComponent)(world.Get(friendlyNPC, vitalsID))
	fVitals.Blood = 100.0

	// Create 2 hostile NPCs (Attacker) near the village
	for i := 0; i < 2; i++ {
		hNPC := world.NewEntity(npcID, affID, posID, vitalsID)
		hAff := (*components.Affiliation)(world.Get(hNPC, affID))
		hAff.CountryID = 1
		hPos := (*components.Position)(world.Get(hNPC, posID))
		hPos.X, hPos.Y = 12.0, 10.0 // Within radius 5
		hVitals := (*components.VitalsComponent)(world.Get(hNPC, vitalsID))
		hVitals.Blood = 100.0
	}

	// Hostiles outnumber friendly (2 vs 1). Update system to trigger siege
	sys.Update(&world)

	// Assert SiegeMarker added
	if !world.Has(defenderVillage, siegeID) {
		t.Fatalf("Village should have SiegeMarker applied")
	}
	siege := (*components.SiegeMarker)(world.Get(defenderVillage, siegeID))
	if siege.BesiegerCountryID != 1 {
		t.Fatalf("Expected BesiegerCountryID 1, got %d", siege.BesiegerCountryID)
	}

	// Re-fetch pointers as structural changes happened
	defMarket = (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	defLoyalty = (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))

	// Assert immediate economic/psychological impact
	if defMarket.FoodPrice != 15.0 {
		t.Fatalf("Expected FoodPrice to spike to 15.0, got %f", defMarket.FoodPrice)
	}
	if defLoyalty.Value != 99 {
		t.Fatalf("Expected Loyalty to drain to 99, got %d", defLoyalty.Value)
	}

	// Run another tick to test continuous effect
	sys.Update(&world)

	// Re-fetch pointers
	defMarket = (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	defLoyalty = (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))

	if defMarket.FoodPrice != 25.0 {
		t.Fatalf("Expected FoodPrice to spike to 25.0, got %f", defMarket.FoodPrice)
	}
	if defLoyalty.Value != 98 {
		t.Fatalf("Expected Loyalty to drain to 98, got %d", defLoyalty.Value)
	}

	// Remove hostility (e.g. they walk away or die)
	sys.Update(&world) // Need to modify NPCs or kill them

	// Kill hostile NPCs by fetching them
	q := world.Query(filter.All(npcID, affID, vitalsID))
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(affID))
		if aff.CountryID == 1 {
			v := (*components.VitalsComponent)(q.Get(vitalsID))
			v.Blood = 0 // Dead
		}
	}

	sys.Update(&world)

	// Assert siege removed
	if world.Has(defenderVillage, siegeID) {
		t.Fatalf("Village should no longer have SiegeMarker after hostiles die")
	}
}
