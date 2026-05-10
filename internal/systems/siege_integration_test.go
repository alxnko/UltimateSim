package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// This E2E test verifies that an overwhelming physical presence of hostile NPCs
// organically triggers a siege, raising food prices and draining loyalty over time.

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize System
	sys := NewSiegeSystem(&world)

	// Component IDs
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeCompID := ecs.ComponentID[components.SiegeMarker](&world)

	// 1. Create a War condition
	// Attacker Country = 10, Defender Country = 20
	warEntity := world.NewEntity(warCompID, affID)
	warComp := (*components.WarTrackerComponent)(world.Get(warEntity, warCompID))
	warComp.TargetCountryID = 20
	warComp.Active = true
	warAff := (*components.Affiliation)(world.Get(warEntity, affID))
	warAff.CountryID = 10

	// 2. Create the Defending Village
	village := world.NewEntity(villageID, posID, affID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X = 50.0
	vPos.Y = 50.0

	vAff := (*components.Affiliation)(world.Get(village, affID))
	vAff.CountryID = 20

	vMarket := (*components.MarketComponent)(world.Get(village, marketID))
	vMarket.FoodPrice = 10.0

	vLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	vLoyalty.Value = 100

	// 3. Create Defending NPCs (1 NPC)
	defNPC := world.NewEntity(npcID, posID, affID)
	defPos := (*components.Position)(world.Get(defNPC, posID))
	defPos.X = 51.0
	defPos.Y = 51.0
	defAff := (*components.Affiliation)(world.Get(defNPC, affID))
	defAff.CountryID = 20

	// 4. Create Attacking NPCs (3 NPCs)
	for i := 0; i < 3; i++ {
		atkNPC := world.NewEntity(npcID, posID, affID)
		atkPos := (*components.Position)(world.Get(atkNPC, posID))
		atkPos.X = 49.0
		atkPos.Y = 49.0 // Within radius
		atkAff := (*components.Affiliation)(world.Get(atkNPC, affID))
		atkAff.CountryID = 10
	}

	// 5. Tick 1 - Should evaluate and apply SiegeMarker (No price change yet)
	sys.Update(&world)

	if !world.Has(village, siegeCompID) {
		t.Fatalf("Expected Village to be besieged by overwhelming hostile forces")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(village, siegeCompID))
	if siegeMarker.BesiegerCountryID != 10 {
		t.Errorf("Expected BesiegerCountryID to be 10, got %d", siegeMarker.BesiegerCountryID)
	}

	// Market and Loyalty should not change on the very first tick the marker is applied
	vMarket = (*components.MarketComponent)(world.Get(village, marketID))
	if vMarket.FoodPrice != 10.0 {
		t.Errorf("Expected FoodPrice to remain 10.0 on tick 1, got %f", vMarket.FoodPrice)
	}

	// 6. Tick 2 - SiegeMarker is present, apply organic economic/psychological effects
	sys.Update(&world)

	vMarket = (*components.MarketComponent)(world.Get(village, marketID))
	if vMarket.FoodPrice <= 10.0 {
		t.Errorf("Expected FoodPrice to increase due to siege, got %f", vMarket.FoodPrice)
	}

	vLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	if vLoyalty.Value >= 100 {
		t.Errorf("Expected Loyalty to decrease due to siege, got %d", vLoyalty.Value)
	}

	// 7. Break the Siege - Move the attackers away
	atkQuery := world.Query(ecs.All(npcID, affID, posID))
	for atkQuery.Next() {
		aff := (*components.Affiliation)(atkQuery.Get(affID))
		if aff.CountryID == 10 {
			pos := (*components.Position)(atkQuery.Get(posID))
			pos.X = 999.0 // Move far away
		}
	}

	// 8. Tick 3 - Siege should be lifted
	sys.Update(&world)

	if world.Has(village, siegeCompID) {
		t.Fatalf("Expected SiegeMarker to be removed after attackers retreated")
	}
}
