package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 49 - The Witch Hunt Engine
// E2E Butterfly Effect Test proving:
// Magic Casting -> High Trauma -> Scapegoat targets Esoteric -> Justice System flags Criminal

func TestWitchHuntSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	grid := engine.NewMapGrid(20, 20)

	// Pre-seed Mana to guarantee casting triggers
	for i := 0; i < len(grid.Mana); i++ {
		grid.Mana[i].Value = 100
	}

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.EsotericMarker](&world)
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
	castingSys := NewCastingSystem(&world, grid)
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
	scape.TargetEsoteric = false

	// 2. Create Normal NPCs (BeliefID 1)
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

	// 3. Create Caster NPC
	casterEnt := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Needs](&world),
	)
	cPos := (*components.Position)(world.Get(casterEnt, ecs.ComponentID[components.Position](&world)))
	cPos.X = 12.0
	cPos.Y = 12.0

	cJob := (*components.JobComponent)(world.Get(casterEnt, ecs.ComponentID[components.JobComponent](&world)))
	cJob.JobID = components.JobCaster

	cBel := (*components.BeliefComponent)(world.Get(casterEnt, ecs.ComponentID[components.BeliefComponent](&world)))
	cBel.Beliefs = append(cBel.Beliefs, components.Belief{BeliefID: 1, Weight: 100}) // Same belief as others

	// 4. Run CastingSystem
	castingSys.Update(&world)

	esotericID := ecs.ComponentID[components.EsotericMarker](&world)
	if !world.Has(casterEnt, esotericID) {
		t.Fatalf("Expected caster NPC to receive EsotericMarker")
	}

	// 5. Run ScapegoatSystem
	// Simulate 60 ticks so modulo 50 matches 10
	for i := 0; i < 60; i++ {
		scapegoatSys.Update(&world)
	}

	// Validate Scapegoat selection
	if !scape.Active {
		t.Fatalf("Expected ScapegoatComponent to be Active after evaluation")
	}

	if !scape.TargetEsoteric {
		t.Errorf("Expected ScapegoatComponent to target Esoteric casters, got TargetEsoteric = false")
	}

	if jur.Trauma != 10 { // 20 - 10 catharsis
		t.Errorf("Expected Trauma to reduce to 10 (catharsis), got %d", jur.Trauma)
	}

	// 6. Run JusticeSystem
	// Tick 1 to trigger detection
	justiceSys.Update(&world)

	// Validate Justice Integration
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	if !world.Has(casterEnt, crimeID) {
		t.Errorf("Expected caster NPC to be flagged with CrimeMarker due to Witch Hunt state")
	}

	// Make sure normal NPCs are safe
	majoritySafe := true
	npcQuery := world.Query(ecs.All(ecs.ComponentID[components.NPC](&world)))
	for npcQuery.Next() {
		ent := npcQuery.Entity()
		if ent != casterEnt && world.Has(ent, crimeID) {
			majoritySafe = false
		}
	}

	if !majoritySafe {
		t.Errorf("Normal NPC was incorrectly flagged with CrimeMarker during Witch Hunt")
	}
}
