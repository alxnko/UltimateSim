package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

func TestSiegeSystem_Integration(t *testing.T) {
	// Initialize world
	world := ecs.NewWorld()

	// 1. Setup the Siege System
	siegeSys := systems.NewSiegeSystem(&world)

	// Create Capitals for Country 1 and Country 2
	cap1 := world.NewEntity(
		ecs.ComponentID[components.CapitalComponent](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.WarTrackerComponent](&world),
	)

	aff1 := (*components.Affiliation)(world.Get(cap1, ecs.ComponentID[components.Affiliation](&world)))
	aff1.CountryID = 1

	war1 := (*components.WarTrackerComponent)(world.Get(cap1, ecs.ComponentID[components.WarTrackerComponent](&world)))
	war1.TargetCountryID = 2
	war1.Active = true

	// 2. Setup the target Village (belongs to Country 2)
	village := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.LoyaltyComponent](&world),
	)

	vPos := (*components.Position)(world.Get(village, ecs.ComponentID[components.Position](&world)))
	vPos.X = 100.0
	vPos.Y = 100.0

	vAff := (*components.Affiliation)(world.Get(village, ecs.ComponentID[components.Affiliation](&world)))
	vAff.CountryID = 2

	vMarket := (*components.MarketComponent)(world.Get(village, ecs.ComponentID[components.MarketComponent](&world)))
	vMarket.FoodPrice = 10.0

	vLoyalty := (*components.LoyaltyComponent)(world.Get(village, ecs.ComponentID[components.LoyaltyComponent](&world)))
	vLoyalty.Value = 100

	// 3. Setup NPCs
	// Defender NPC (belongs to Country 2)
	defender := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.NPC](&world),
	)
	dPos := (*components.Position)(world.Get(defender, ecs.ComponentID[components.Position](&world)))
	dPos.X = 101.0
	dPos.Y = 101.0
	dAff := (*components.Affiliation)(world.Get(defender, ecs.ComponentID[components.Affiliation](&world)))
	dAff.CountryID = 2

	// Attacker NPCs (belong to Country 1)
	for i := 0; i < 3; i++ {
		attacker := world.NewEntity(
			ecs.ComponentID[components.Position](&world),
			ecs.ComponentID[components.Affiliation](&world),
			ecs.ComponentID[components.NPC](&world),
		)
		aPos := (*components.Position)(world.Get(attacker, ecs.ComponentID[components.Position](&world)))
		aPos.X = 102.0
		aPos.Y = 102.0
		aAff := (*components.Affiliation)(world.Get(attacker, ecs.ComponentID[components.Affiliation](&world)))
		aAff.CountryID = 1
	}

	// 4. Run the first ticks to apply the siege marker (tickCounter must reach 10)
	for i := 0; i < 10; i++ {
		siegeSys.Update(&world)
	}

	// Check if siege marker was applied
	if !world.Has(village, ecs.ComponentID[components.SiegeMarker](&world)) {
		t.Fatalf("Expected village to have SiegeMarker applied, but it did not")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(village, ecs.ComponentID[components.SiegeMarker](&world)))
	if siegeMarker.BesiegerCountryID != 1 {
		t.Errorf("Expected BesiegerCountryID 1, got %d", siegeMarker.BesiegerCountryID)
	}

	// Wait for another siege update (10 ticks) to see the effects (spiking prices, draining loyalty)
	for i := 0; i < 10; i++ {
		siegeSys.Update(&world)
	}

	vMarket = (*components.MarketComponent)(world.Get(village, ecs.ComponentID[components.MarketComponent](&world)))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(village, ecs.ComponentID[components.LoyaltyComponent](&world)))

	if vMarket.FoodPrice <= 10.0 {
		t.Errorf("Expected FoodPrice to spike due to siege, got %f", vMarket.FoodPrice)
	}

	if vLoyalty.Value >= 100 {
		t.Errorf("Expected Loyalty to drain due to siege, got %d", vLoyalty.Value)
	}

	// Now move attackers away to end the siege
	npcFilter := filter.All(ecs.ComponentID[components.NPC](&world))
	query := world.Query(npcFilter)
	for query.Next() {
		aff := (*components.Affiliation)(query.Get(ecs.ComponentID[components.Affiliation](&world)))
		if aff.CountryID == 1 {
			pos := (*components.Position)(query.Get(ecs.ComponentID[components.Position](&world)))
			pos.X = 500.0 // Move far away
		}
	}

	// Wait for another 10 ticks to remove the siege
	for i := 0; i < 10; i++ {
		siegeSys.Update(&world)
	}

	if world.Has(village, ecs.ComponentID[components.SiegeMarker](&world)) {
		t.Fatalf("Expected SiegeMarker to be removed after attackers moved away")
	}
}
