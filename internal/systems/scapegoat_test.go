package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 36.1: The Scapegoat & Witch Hunt Engine Testing
// E2E Butterfly Effect Test proving:
// Trauma -> Scapegoat System -> Justice System -> Criminal Flagging

func TestScapegoatSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.ScapegoatComponent](&world)
	ecs.ComponentID[components.NPC](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.BeliefComponent](&world)
	ecs.ComponentID[components.Memory](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.CrimeMarker](&world)
	ecs.ComponentID[components.Needs](&world)

	// Systems
	scapegoatSys := NewScapegoatSystem()
	hooks := engine.NewSparseHookGraph()
	justiceSys := NewJusticeSystem(&world, hooks)

	// 1. Create a traumatized Jurisdiction
	jurEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
		ecs.ComponentID[components.ScapegoatComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)

	jurPos := (*components.Position)(world.Get(jurEnt, ecs.ComponentID[components.Position](&world)))
	jurPos.X = 10.0
	jurPos.Y = 10.0

	jur := (*components.JurisdictionComponent)(world.Get(jurEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	jur.RadiusSquared = 100.0 // Radius 10
	jur.Trauma = 20           // Highly traumatized

	scape := (*components.ScapegoatComponent)(world.Get(jurEnt, ecs.ComponentID[components.ScapegoatComponent](&world)))
	scape.Active = false

	// 2. Create Majority NPCs (BeliefID 1)
	for i := 0; i < 5; i++ {
		ent := world.NewEntity(
			ecs.ComponentID[components.NPC](&world),
			ecs.ComponentID[components.Position](&world),
			ecs.ComponentID[components.BeliefComponent](&world),
			ecs.ComponentID[components.Memory](&world),
			ecs.ComponentID[components.Affiliation](&world),
			ecs.ComponentID[components.Identity](&world),
			ecs.ComponentID[components.Needs](&world),
		)
		pos := (*components.Position)(world.Get(ent, ecs.ComponentID[components.Position](&world)))
		pos.X = 11.0
		pos.Y = 11.0

		bel := (*components.BeliefComponent)(world.Get(ent, ecs.ComponentID[components.BeliefComponent](&world)))
		bel.Beliefs = append(bel.Beliefs, components.Belief{BeliefID: 1, Weight: 100})
	}

	// 3. Create Minority NPC (BeliefID 2)
	minorityEnt := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Needs](&world),
	)
	mPos := (*components.Position)(world.Get(minorityEnt, ecs.ComponentID[components.Position](&world)))
	mPos.X = 12.0
	mPos.Y = 12.0

	mBel := (*components.BeliefComponent)(world.Get(minorityEnt, ecs.ComponentID[components.BeliefComponent](&world)))
	mBel.Beliefs = append(mBel.Beliefs, components.Belief{BeliefID: 2, Weight: 100})

	// 4. Run ScapegoatSystem
	// Simulate 60 ticks so modulo 50 matches 10
	for i := 0; i < 60; i++ {
		scapegoatSys.Update(&world)
	}

	// Validate Scapegoat selection
	if !scape.Active {
		t.Fatalf("Expected ScapegoatComponent to be Active after evaluation")
	}

	if scape.TargetBeliefID != 2 {
		t.Errorf("Expected ScapegoatComponent to target Minority Belief (2), got %d", scape.TargetBeliefID)
	}

	if jur.Trauma != 10 { // 20 - 10 catharsis
		t.Errorf("Expected Trauma to reduce to 10 (catharsis), got %d", jur.Trauma)
	}

	// 5. Run JusticeSystem
	// Since JusticeSystem looks for Affiliation and Memory, make sure the entities have them properly initialized
	// and that they actually fall within the Jurisdiction radius. (10,10 to 12,12 is distSq 8, which is <= 100)

	// Tick 1 to trigger detection
	justiceSys.Update(&world)

	// Validate Justice Integration
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	if !world.Has(minorityEnt, crimeID) {
		t.Errorf("Expected minority NPC to be flagged with CrimeMarker due to Scapegoat state")
	}

	// Make sure majority is safe
	majoritySafe := true
	npcQuery := world.Query(ecs.All(ecs.ComponentID[components.NPC](&world)))
	for npcQuery.Next() {
		ent := npcQuery.Entity()
		if ent != minorityEnt && world.Has(ent, crimeID) {
			majoritySafe = false
		}
	}

	if !majoritySafe {
		t.Errorf("Majority NPC was incorrectly flagged with CrimeMarker")
	}
}
