package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components
	villageID := ecs.ComponentID[components.Village](&world)
	posID := ecs.ComponentID[components.Position](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	warTrackerID := ecs.ComponentID[components.WarTrackerComponent](&world)

	// Create Country A (Attacker) and Country B (Defender)
	const CountryA uint32 = 100
	const CountryB uint32 = 200

	// Set up War: Country A is actively attacking Country B
	attackerCapital := world.NewEntity(affilID, warTrackerID)
	affilA := (*components.Affiliation)(world.Get(attackerCapital, affilID))
	affilA.CountryID = CountryA
	warA := (*components.WarTrackerComponent)(world.Get(attackerCapital, warTrackerID))
	warA.TargetCountryID = CountryB
	warA.Active = true

	// Create Defending Village (Country B)
	village := world.NewEntity(villageID, posID, affilID, marketID, loyaltyID)
	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X, vPos.Y = 10.0, 10.0
	vAffil := (*components.Affiliation)(world.Get(village, affilID))
	vAffil.CountryID = CountryB
	vMarket := (*components.MarketComponent)(world.Get(village, marketID))
	vMarket.FoodPrice = 10.0
	vLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	vLoyalty.Value = 100

	// Create Attacking NPCs (Country A) near the village (distSq <= 25.0)
	// We need 2 attackers to outnumber 1 allied NPC (or 0 allied NPCs)
	attacker1 := world.NewEntity(npcID, posID, affilID)
	a1Pos := (*components.Position)(world.Get(attacker1, posID))
	a1Pos.X, a1Pos.Y = 12.0, 10.0 // distSq = 4.0
	a1Affil := (*components.Affiliation)(world.Get(attacker1, affilID))
	a1Affil.CountryID = CountryA

	attacker2 := world.NewEntity(npcID, posID, affilID)
	a2Pos := (*components.Position)(world.Get(attacker2, posID))
	a2Pos.X, a2Pos.Y = 10.0, 13.0 // distSq = 9.0
	a2Affil := (*components.Affiliation)(world.Get(attacker2, affilID))
	a2Affil.CountryID = CountryA

	// Create 1 Defending NPC (Country B) to ensure hostileCount > alliedCount logic works
	defender1 := world.NewEntity(npcID, posID, affilID)
	d1Pos := (*components.Position)(world.Get(defender1, posID))
	d1Pos.X, d1Pos.Y = 10.0, 11.0 // distSq = 1.0
	d1Affil := (*components.Affiliation)(world.Get(defender1, affilID))
	d1Affil.CountryID = CountryB

	// Initialize System
	siegeSys := systems.NewSiegeSystem(&world)

	// --- Tick 1: Attackers are present and outnumber defenders (2 vs 1) ---
	siegeSys.Update(&world)

	// Re-fetch pointers after potential structural changes (SiegeMarker addition)
	vMarket = (*components.MarketComponent)(world.Get(village, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))

	if !world.Has(village, siegeID) {
		t.Fatalf("Expected village to have SiegeMarker applied")
	}
	if vMarket.FoodPrice <= 10.0 {
		t.Fatalf("Expected FoodPrice to spike due to siege, got %f", vMarket.FoodPrice)
	}
	if vLoyalty.Value >= 100 {
		t.Fatalf("Expected Loyalty to drop due to siege, got %d", vLoyalty.Value)
	}

	// --- Tick 2: Attackers move away (distSq > 25.0) ---
	a1Pos = (*components.Position)(world.Get(attacker1, posID))
	a2Pos = (*components.Position)(world.Get(attacker2, posID))
	a1Pos.X, a1Pos.Y = 100.0, 100.0
	a2Pos.X, a2Pos.Y = 100.0, 100.0

	siegeSys.Update(&world)

	if world.Has(village, siegeID) {
		t.Fatalf("Expected SiegeMarker to be removed when attackers leave")
	}
}
