package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine E2E Butterfly Effect Test
// Proves that Geopolitical War triggers a local physical siege, which directly
// impacts the local Economy (food prices spike) and Politics (loyalty drops).
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	// Create defending Country 1
	country1 := world.NewEntity(affID, warID)
	c1Aff := (*components.Affiliation)(world.Get(country1, affID))
	c1Aff.CountryID = 1

	// Create attacking Country 2
	country2 := world.NewEntity(affID, warID)
	c2Aff := (*components.Affiliation)(world.Get(country2, affID))
	c2Aff.CountryID = 2
	c2War := (*components.WarTrackerComponent)(world.Get(country2, warID))
	c2War.Active = true
	c2War.TargetCountryID = 1

	// Create defending Village in Country 1
	village := world.NewEntity(villageID, posID, affID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X = 10.0
	vPos.Y = 10.0
	vAff := (*components.Affiliation)(world.Get(village, affID))
	vAff.CountryID = 1
	vMarket := (*components.MarketComponent)(world.Get(village, marketID))
	vMarket.FoodPrice = 5.0
	vLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	vLoyalty.Value = 100

	// Create 2 Hostile NPCs from Country 2 at the village
	for i := 0; i < 2; i++ {
		e := world.NewEntity(npcID, posID, affID, identID)
		pos := (*components.Position)(world.Get(e, posID))
		pos.X = 11.0
		pos.Y = 11.0 // distSq = 2.0
		aff := (*components.Affiliation)(world.Get(e, affID))
		aff.CountryID = 2
	}

	// Initialize system
	siegeSys := NewSiegeSystem(&world)

	// Tick 1: Hostiles outnumber defenders (2 to 0). Siege should apply.
	siegeSys.Update(&world)

	if !world.Has(village, siegeID) {
		t.Fatalf("Village should have a SiegeMarker applied")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(village, siegeID))
	if siegeMarker.BesiegerCountryID != 2 {
		t.Errorf("Expected BesiegerCountryID 2, got %d", siegeMarker.BesiegerCountryID)
	}

	// Verify economic and political fallout (Butterfly Effect)
	vMarket = (*components.MarketComponent)(world.Get(village, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))

	if vMarket.FoodPrice != 15.0 {
		t.Errorf("Expected FoodPrice to spike to 15.0, got %f", vMarket.FoodPrice)
	}

	if vLoyalty.Value != 95 {
		t.Errorf("Expected Loyalty to drop to 95, got %d", vLoyalty.Value)
	}

	// Create 3 Defending NPCs from Country 1 to break the siege
	for i := 0; i < 3; i++ {
		e := world.NewEntity(npcID, posID, affID, identID)
		pos := (*components.Position)(world.Get(e, posID))
		pos.X = 10.0
		pos.Y = 10.0 // At the village
		aff := (*components.Affiliation)(world.Get(e, affID))
		aff.CountryID = 1
	}

	// Tick 2: Defenders outnumber hostiles (3 to 2). Siege should be broken.
	siegeSys.Update(&world)

	if world.Has(village, siegeID) {
		t.Fatalf("Village should no longer have a SiegeMarker applied after defenders arrive")
	}

	// Ensure determinism check doesn't fail
}
