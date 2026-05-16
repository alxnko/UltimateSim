package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine Integration Test
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.MarketComponent](&world)
	ecs.ComponentID[components.LoyaltyComponent](&world)
	ecs.ComponentID[components.NPC](&world)
	ecs.ComponentID[components.CountryComponent](&world)
	ecs.ComponentID[components.CapitalComponent](&world)
	ecs.ComponentID[components.WarTrackerComponent](&world)
	ecs.ComponentID[components.SiegeMarker](&world)

	// Set up Country A (Defender)
	countryA := world.NewEntity(
		ecs.ComponentID[components.CapitalComponent](&world),
		ecs.ComponentID[components.CountryComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.WarTrackerComponent](&world),
	)
	affilA := (*components.Affiliation)(world.Get(countryA, ecs.ComponentID[components.Affiliation](&world)))
	affilA.CountryID = 1

	warA := (*components.WarTrackerComponent)(world.Get(countryA, ecs.ComponentID[components.WarTrackerComponent](&world)))
	warA.Active = false

	// Set up Country B (Attacker)
	countryB := world.NewEntity(
		ecs.ComponentID[components.CapitalComponent](&world),
		ecs.ComponentID[components.CountryComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.WarTrackerComponent](&world),
	)
	affilB := (*components.Affiliation)(world.Get(countryB, ecs.ComponentID[components.Affiliation](&world)))
	affilB.CountryID = 2

	warB := (*components.WarTrackerComponent)(world.Get(countryB, ecs.ComponentID[components.WarTrackerComponent](&world)))
	warB.Active = true
	warB.TargetCountryID = 1

	// Setup Village for Country A
	villageA := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.LoyaltyComponent](&world),
	)
	vPos := (*components.Position)(world.Get(villageA, ecs.ComponentID[components.Position](&world)))
	vPos.X, vPos.Y = 10.0, 10.0

	vAffil := (*components.Affiliation)(world.Get(villageA, ecs.ComponentID[components.Affiliation](&world)))
	vAffil.CountryID = 1

	vMarket := (*components.MarketComponent)(world.Get(villageA, ecs.ComponentID[components.MarketComponent](&world)))
	vMarket.FoodPrice = 1.0

	vLoyalty := (*components.LoyaltyComponent)(world.Get(villageA, ecs.ComponentID[components.LoyaltyComponent](&world)))
	vLoyalty.Value = 100

	// Set up defenders (1 NPC for Country A)
	defender := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)
	dPos := (*components.Position)(world.Get(defender, ecs.ComponentID[components.Position](&world)))
	dPos.X, dPos.Y = 10.0, 10.0
	dAffil := (*components.Affiliation)(world.Get(defender, ecs.ComponentID[components.Affiliation](&world)))
	dAffil.CountryID = 1

	// Set up hostiles (3 NPCs for Country B)
	var attackers []ecs.Entity
	for i := 0; i < 3; i++ {
		attacker := world.NewEntity(
			ecs.ComponentID[components.NPC](&world),
			ecs.ComponentID[components.Position](&world),
			ecs.ComponentID[components.Affiliation](&world),
		)
		aPos := (*components.Position)(world.Get(attacker, ecs.ComponentID[components.Position](&world)))
		aPos.X, aPos.Y = 11.0, 11.0 // Within 5 units distance squared (1^2 + 1^2 = 2 <= 25)
		aAffil := (*components.Affiliation)(world.Get(attacker, ecs.ComponentID[components.Affiliation](&world)))
		aAffil.CountryID = 2
		attackers = append(attackers, attacker)
	}

	sys := &SiegeSystem{}
	sys.Initialize(&world)

	// Tick 1: Outnumbered, so a siege should begin.
	sys.Update(&world)

	if !world.Has(villageA, ecs.ComponentID[components.SiegeMarker](&world)) {
		t.Fatalf("Expected SiegeMarker to be added to Village A because it is outnumbered.")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(villageA, ecs.ComponentID[components.SiegeMarker](&world)))
	if siegeMarker.BesiegerCountryID != 2 {
		t.Errorf("Expected BesiegerCountryID to be 2, got %d", siegeMarker.BesiegerCountryID)
	}

	// Re-fetch component values to verify they were updated structurally
	vMarketAfter := (*components.MarketComponent)(world.Get(villageA, ecs.ComponentID[components.MarketComponent](&world)))
	vLoyaltyAfter := (*components.LoyaltyComponent)(world.Get(villageA, ecs.ComponentID[components.LoyaltyComponent](&world)))

	if vMarketAfter.FoodPrice <= 1.0 {
		t.Errorf("Expected FoodPrice to spike above 1.0, got %f", vMarketAfter.FoodPrice)
	}
	if vLoyaltyAfter.Value >= 100 {
		t.Errorf("Expected Loyalty to drain below 100, got %d", vLoyaltyAfter.Value)
	}

	// Move attackers away so siege ends
	for _, attacker := range attackers {
		aPos := (*components.Position)(world.Get(attacker, ecs.ComponentID[components.Position](&world)))
		aPos.X, aPos.Y = 100.0, 100.0 // Move far away
	}

	// Tick 2: Attackers are gone, siege should end.
	sys.Update(&world)

	if world.Has(villageA, ecs.ComponentID[components.SiegeMarker](&world)) {
		t.Fatalf("Expected SiegeMarker to be removed from Village A because attackers left.")
	}
}
