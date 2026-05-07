package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine Integration Test
// Demonstrates the "Butterfly Effect": War Declared -> Hostiles Outnumber Defenders ->
// SiegeMarker Applied -> FoodPrice Spikes & Loyalty Drains.

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// 2. Initialize System
	sys := NewSiegeSystem(&world)

	// 3. Setup Entities

	// Attacker Capital (Country 1) at War with Defender (Country 2)
	attackerCap := world.NewEntity(posID, affID, capID, warID)
	world.Get(attackerCap, posID) // Just initialize
	attAff := (*components.Affiliation)(world.Get(attackerCap, affID))
	attAff.CountryID = 1
	war := (*components.WarTrackerComponent)(world.Get(attackerCap, warID))
	war.TargetCountryID = 2
	war.Active = true

	// Defender Village (Country 2)
	defenderVillage := world.NewEntity(posID, affID, villageID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(defenderVillage, posID))
	vPos.X, vPos.Y = 10.0, 10.0
	vAff := (*components.Affiliation)(world.Get(defenderVillage, affID))
	vAff.CountryID = 2
	vMarket := (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	vMarket.FoodPrice = 5.0
	vLoyalty := (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))
	vLoyalty.Value = 100

	// 1 Defender NPC
	defenderNPC := world.NewEntity(posID, affID, npcID)
	dPos := (*components.Position)(world.Get(defenderNPC, posID))
	dPos.X, dPos.Y = 10.0, 10.0
	dAff := (*components.Affiliation)(world.Get(defenderNPC, affID))
	dAff.CountryID = 2

	// 3 Hostile NPCs (Attackers) within radius
	for i := 0; i < 3; i++ {
		hostileNPC := world.NewEntity(posID, affID, npcID)
		hPos := (*components.Position)(world.Get(hostileNPC, posID))
		hPos.X, hPos.Y = 11.0, 11.0 // distSq = 2.0 (<= 25.0)
		hAff := (*components.Affiliation)(world.Get(hostileNPC, affID))
		hAff.CountryID = 1
	}

	// ---------------------------------------------------------
	// TICK 1: Tick counter increments, but system only runs every 50 ticks.
	// So we fast forward.
	// ---------------------------------------------------------
	for i := 0; i < 50; i++ {
		sys.Update(&world)
	}

	// After 50 ticks, the siege should be applied
	if !world.Has(defenderVillage, siegeID) {
		t.Logf("Village pos: X=%v, Y=%v", vPos.X, vPos.Y)
		t.Logf("Active wars: %v", sys)
		t.Fatalf("Tick 50: Expected SiegeMarker to be applied, but it was not")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(defenderVillage, siegeID))
	if siegeMarker.BesiegerCountryID != 1 {
		t.Fatalf("Tick 50: Expected BesiegerCountryID to be 1, got %d", siegeMarker.BesiegerCountryID)
	}

	// Re-fetch pointers after structural change (world.Add)
	vMarket = (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))

	if vMarket.FoodPrice != 10.0 {
		t.Fatalf("Tick 50: Expected FoodPrice to spike to 10.0, got %f", vMarket.FoodPrice)
	}

	if vLoyalty.Value != 98 {
		t.Fatalf("Tick 50: Expected Loyalty to drain to 98, got %d", vLoyalty.Value)
	}

	// ---------------------------------------------------------
	// Fast forward another 50 ticks
	// ---------------------------------------------------------
	for i := 0; i < 50; i++ {
		sys.Update(&world)
	}

	if vMarket.FoodPrice != 15.0 {
		t.Fatalf("Tick 100: Expected FoodPrice to spike to 15.0, got %f", vMarket.FoodPrice)
	}

	if vLoyalty.Value != 96 {
		t.Fatalf("Tick 100: Expected Loyalty to drain to 96, got %d", vLoyalty.Value)
	}

	// ---------------------------------------------------------
	// Now, war ends. The siege should be lifted.
	// ---------------------------------------------------------
	war.Active = false

	for i := 0; i < 50; i++ {
		sys.Update(&world)
	}

	if world.Has(defenderVillage, siegeID) {
		t.Fatalf("Tick 150: Expected SiegeMarker to be removed after war ended")
	}
}
