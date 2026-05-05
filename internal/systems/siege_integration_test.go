package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine Integration Test
// Proves that when a hostile NPC outnumbers friendly NPCs near a village during a war,
// the SiegeMarker is applied, FoodPrice spikes, and Loyalty is drained.

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	engine.InitializeRNG([32]byte{1})

	// Add component IDs
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.CapitalComponent](&world)
	ecs.ComponentID[components.WarTrackerComponent](&world)
	ecs.ComponentID[components.NPC](&world)
	siegeMarkerID := ecs.ComponentID[components.SiegeMarker](&world)

	siegeSys := systems.NewSiegeSystem(&world)

	// Create Country A (Defender)
	villageA := world.NewEntity()
	world.Add(villageA,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		loyaltyID,
		ecs.ComponentID[components.Position](&world),
	)
	affA := (*components.Affiliation)(world.Get(villageA, ecs.ComponentID[components.Affiliation](&world)))
	affA.CountryID = 1
	posA := (*components.Position)(world.Get(villageA, ecs.ComponentID[components.Position](&world)))
	posA.X = 10.0
	posA.Y = 10.0
	loyA := (*components.LoyaltyComponent)(world.Get(villageA, loyaltyID))
	loyA.Value = 100
	marketA := (*components.MarketComponent)(world.Get(villageA, ecs.ComponentID[components.MarketComponent](&world)))
	marketA.FoodPrice = 5.0

	// Create Country B (Attacker)
	capitalB := world.NewEntity()
	world.Add(capitalB,
		ecs.ComponentID[components.CapitalComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.WarTrackerComponent](&world),
	)
	affB := (*components.Affiliation)(world.Get(capitalB, ecs.ComponentID[components.Affiliation](&world)))
	affB.CountryID = 2
	warB := (*components.WarTrackerComponent)(world.Get(capitalB, ecs.ComponentID[components.WarTrackerComponent](&world)))
	warB.Active = true
	warB.TargetCountryID = 1 // Attacking Country A

	// Create Hostile NPCs (Country 2) near Village A
	for i := 0; i < 3; i++ {
		npc := world.NewEntity()
		world.Add(npc,
			ecs.ComponentID[components.NPC](&world),
			ecs.ComponentID[components.Affiliation](&world),
			ecs.ComponentID[components.Position](&world),
		)
		npcAff := (*components.Affiliation)(world.Get(npc, ecs.ComponentID[components.Affiliation](&world)))
		npcAff.CountryID = 2
		npcPos := (*components.Position)(world.Get(npc, ecs.ComponentID[components.Position](&world)))
		npcPos.X = 11.0 // Very close to village A
		npcPos.Y = 11.0
	}

	// Create Friendly NPC (Country 1) near Village A
	npcF := world.NewEntity()
	world.Add(npcF,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Position](&world),
	)
	npcFAff := (*components.Affiliation)(world.Get(npcF, ecs.ComponentID[components.Affiliation](&world)))
	npcFAff.CountryID = 1
	npcFPos := (*components.Position)(world.Get(npcF, ecs.ComponentID[components.Position](&world)))
	npcFPos.X = 12.0
	npcFPos.Y = 12.0

	// Tick 1
	siegeSys.Update(&world)

	// Assertions
	if !world.Has(villageA, siegeMarkerID) {
		t.Fatalf("Tick 1: Village A was outnumbered by hostile NPCs during war, but SiegeMarker was not applied")
	}

	// Stale pointer risk - re-fetch components after structural change
	loyA = (*components.LoyaltyComponent)(world.Get(villageA, loyaltyID))
	marketA = (*components.MarketComponent)(world.Get(villageA, ecs.ComponentID[components.MarketComponent](&world)))

	if loyA.Value != 99 {
		t.Errorf("Tick 1: Expected Loyalty to drain to 99, got %d", loyA.Value)
	}

	if marketA.FoodPrice != 7.0 {
		t.Errorf("Tick 1: Expected FoodPrice to spike to 7.0, got %f", marketA.FoodPrice)
	}

	// Now move hostiles away
	filterNPC := ecs.All(ecs.ComponentID[components.NPC](&world))
	query := world.Query(&filterNPC)
	for query.Next() {
		aff := (*components.Affiliation)(query.Get(ecs.ComponentID[components.Affiliation](&world)))
		if aff.CountryID == 2 {
			pos := (*components.Position)(query.Get(ecs.ComponentID[components.Position](&world)))
			pos.X = 100.0 // Move far away
		}
	}

	// Tick 2
	siegeSys.Update(&world)

	if world.Has(villageA, siegeMarkerID) {
		t.Fatalf("Tick 2: Hostile NPCs moved away, but SiegeMarker was not removed")
	}
}
