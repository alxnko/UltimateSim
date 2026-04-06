package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 54 - The Radicalization Engine (Echo Chamber)

func TestEchoChamber_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Setup MapGrid with an isolated tile
	mapGrid := engine.NewMapGrid(10, 10)
	isolatedIdx := 1*10 + 1 // x=1, y=1
	mapGrid.TileStates[isolatedIdx].FootTraffic = 0 // Extreme isolation

	// 2. Setup the Overlord Capital (Cosmopolitan, Belief 100)
	capEnt := world.NewEntity()
	world.Add(capEnt,
		ecs.ComponentID[components.CapitalComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.CultureComponent](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
	)

	capAff := (*components.Affiliation)(world.Get(capEnt, ecs.ComponentID[components.Affiliation](&world)))
	capAff.CountryID = 1

	capCult := (*components.CultureComponent)(world.Get(capEnt, ecs.ComponentID[components.CultureComponent](&world)))
	capCult.LanguageID = 1

	capBelief := (*components.BeliefComponent)(world.Get(capEnt, ecs.ComponentID[components.BeliefComponent](&world)))
	capBelief.Beliefs = []components.Belief{
		{BeliefID: 100, Weight: 50}, // Main religion
	}

	// 3. Setup the Isolated Vassal Village
	villEnt := world.NewEntity()
	world.Add(villEnt,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.LoyaltyComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.CultureComponent](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
	)

	villPos := (*components.Position)(world.Get(villEnt, ecs.ComponentID[components.Position](&world)))
	villPos.X = 1.0
	villPos.Y = 1.0

	villAff := (*components.Affiliation)(world.Get(villEnt, ecs.ComponentID[components.Affiliation](&world)))
	villAff.CityID = 2
	villAff.CountryID = 1 // Vassal of Capital 1

	villLoyalty := (*components.LoyaltyComponent)(world.Get(villEnt, ecs.ComponentID[components.LoyaltyComponent](&world)))
	villLoyalty.Value = 10 // Start with some loyalty

	villCult := (*components.CultureComponent)(world.Get(villEnt, ecs.ComponentID[components.CultureComponent](&world)))
	villCult.LanguageID = 1

	// Village holds the state religion but also a minority divergent belief
	villBelief := (*components.BeliefComponent)(world.Get(villEnt, ecs.ComponentID[components.BeliefComponent](&world)))
	villBelief.Beliefs = []components.Belief{
		{BeliefID: 100, Weight: 20}, // State Religion
		{BeliefID: 200, Weight: 30}, // Divergent Belief (Currently slightly dominant)
	}

	// 4. Setup NPCs in the village
	for i := 0; i < 5; i++ {
		npcEnt := world.NewEntity()
		world.Add(npcEnt,
			ecs.ComponentID[components.NPC](&world),
			ecs.ComponentID[components.Affiliation](&world),
			ecs.ComponentID[components.BeliefComponent](&world),
		)

		npcAff := (*components.Affiliation)(world.Get(npcEnt, ecs.ComponentID[components.Affiliation](&world)))
		npcAff.CityID = 2
		npcAff.CountryID = 1

		npcBelief := (*components.BeliefComponent)(world.Get(npcEnt, ecs.ComponentID[components.BeliefComponent](&world)))
		npcBelief.Beliefs = []components.Belief{
			{BeliefID: 100, Weight: 20},
			{BeliefID: 200, Weight: 30},
		}
	}

	// 5. Initialize Systems
	echoSystem := NewEchoChamberSystem(mapGrid)
	frictionSystem := NewCulturalFrictionSystem()

	// 6. Run the Simulation (Echo Chamber takes 100 ticks per eval, Friction 50 ticks)
	for tick := 1; tick <= 500; tick++ {
		echoSystem.Update(&world)
		frictionSystem.Update(&world)

		if tick % 50 == 0 {
			// Update the village's dominant belief based on NPCs for FrictionSystem to detect
			// (Normally another system or shared pointer would aggregate this, but we simulate it here)
			npcQuery := world.Query(ecs.All(ecs.ComponentID[components.NPC](&world), ecs.ComponentID[components.BeliefComponent](&world)))
			var domBelief uint32 = 0
			var maxWeight int32 = -1
			for npcQuery.Next() {
				bComp := (*components.BeliefComponent)(npcQuery.Get(ecs.ComponentID[components.BeliefComponent](&world)))
				for _, b := range bComp.Beliefs {
					if b.Weight > maxWeight {
						maxWeight = b.Weight
						domBelief = b.BeliefID
					}
				}
				break // Just sample one for the village aggregation in test
			}
			villBelief.Beliefs = []components.Belief{{BeliefID: domBelief, Weight: maxWeight}}
		}

	}

	// 7. Verify the Butterfly Effect

	// A. Did the Echo Chamber radicalize the divergent belief?
	npcQuery := world.Query(ecs.All(ecs.ComponentID[components.NPC](&world), ecs.ComponentID[components.BeliefComponent](&world)))
	if !npcQuery.Next() {
		t.Fatalf("Failed to find NPCs")
	}
	npcBeliefs := (*components.BeliefComponent)(npcQuery.Get(ecs.ComponentID[components.BeliefComponent](&world)))

	var divergentWeight int32
	var stateReligionWeight int32
	for _, b := range npcBeliefs.Beliefs {
		if b.BeliefID == 200 {
			divergentWeight = b.Weight
		} else if b.BeliefID == 100 {
			stateReligionWeight = b.Weight
		}
	}

	if divergentWeight <= 30 {
		t.Errorf("Expected divergent belief to be radicalized (>30), got %d", divergentWeight)
	}
	if stateReligionWeight >= 20 {
		t.Errorf("Expected state religion to decay (<20), got %d", stateReligionWeight)
	}

	// B. Did Cultural Friction drain loyalty due to the radicalized divergent belief?
	if villLoyalty.Value != 0 {
		t.Errorf("Expected Loyalty to drop to 0 due to Cultural Friction, got %d", villLoyalty.Value)
	}
}
