package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// This test simulates the "Butterfly Effect": Geography + Combat -> Logistics (Starvation) & Loyalty (Rebellion)
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewSiegeSystem(&world)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	warTrackerID := ecs.ComponentID[components.WarTrackerComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)

	// 1. Create defending village (Country 2)
	vEnt := world.NewEntity(posID, affID, villageID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(vEnt, posID))
	vPos.X, vPos.Y = 10, 10

	vAff := (*components.Affiliation)(world.Get(vEnt, affID))
	vAff.CountryID = 2

	vMarket := (*components.MarketComponent)(world.Get(vEnt, marketID))
	vMarket.FoodPrice = 2.0 // Base price

	vLoyalty := (*components.LoyaltyComponent)(world.Get(vEnt, loyaltyID))
	vLoyalty.Value = 100 // Full loyalty

	// 2. Simulate WarTracker (Country 1 attacks Country 2)
	capEnt := world.NewEntity(affID, warTrackerID)
	cAff := (*components.Affiliation)(world.Get(capEnt, affID))
	cAff.CountryID = 1

	warTracker := (*components.WarTrackerComponent)(world.Get(capEnt, warTrackerID))
	warTracker.Active = true
	warTracker.TargetCountryID = 2

	// 3. Create NPCs around the village
	// Defender (Country 2)
	dNPC := world.NewEntity(npcID, posID, affID)
	dPos := (*components.Position)(world.Get(dNPC, posID))
	dPos.X, dPos.Y = 11, 10 // distSq = 1 (within 25)
	dAff := (*components.Affiliation)(world.Get(dNPC, affID))
	dAff.CountryID = 2

	// Attacker 1 (Country 1)
	aNPC1 := world.NewEntity(npcID, posID, affID)
	aPos1 := (*components.Position)(world.Get(aNPC1, posID))
	aPos1.X, aPos1.Y = 9, 10 // distSq = 1 (within 25)
	aAff1 := (*components.Affiliation)(world.Get(aNPC1, affID))
	aAff1.CountryID = 1

	// Attacker 2 (Country 1) -> Outnumbers defenders 2 to 1
	aNPC2 := world.NewEntity(npcID, posID, affID)
	aPos2 := (*components.Position)(world.Get(aNPC2, posID))
	aPos2.X, aPos2.Y = 10, 11 // distSq = 1 (within 25)
	aAff2 := (*components.Affiliation)(world.Get(aNPC2, affID))
	aAff2.CountryID = 1

	// Fast forward 100 ticks to trigger the system
	sys.tickCounter = 99
	sys.Update(&world)

	// 4. Assertions: Siege Application
	if !world.Has(vEnt, siegeID) {
		t.Fatalf("Expected SiegeMarker to be applied to Village (attackers outnumber defenders)")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(vEnt, siegeID))
	if siegeMarker.BesiegerCountryID != 1 {
		t.Fatalf("Expected BesiegerCountryID to be 1, got %d", siegeMarker.BesiegerCountryID)
	}

	// 5. Assertions: Economic & Loyalty impact (The Butterfly Effect)
	vMarketAfter := (*components.MarketComponent)(world.Get(vEnt, marketID))
	if vMarketAfter.FoodPrice <= 2.0 {
		t.Fatalf("Expected FoodPrice to skyrocket during siege, but remained %f", vMarketAfter.FoodPrice)
	}

	vLoyaltyAfter := (*components.LoyaltyComponent)(world.Get(vEnt, loyaltyID))
	if vLoyaltyAfter.Value >= 100 {
		t.Fatalf("Expected Loyalty to drain during siege, but remained %d", vLoyaltyAfter.Value)
	}

	// 6. Assertions: Siege Removal
	// Move attackers away (distSq > 25)
	aPos1.X, aPos1.Y = 100, 100
	aPos2.X, aPos2.Y = 100, 100

	// Tick again
	sys.tickCounter = 199
	sys.Update(&world)

	if world.Has(vEnt, siegeID) {
		t.Fatalf("Expected SiegeMarker to be removed when attackers leave")
	}
}
