package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 49: The Witch Hunt Engine Integration Test
func TestWitchHunt_Integration(t *testing.T) {
	// Initialize deterministic PRNG
	engine.InitializeRNG([32]byte{1, 2, 3})

	world := ecs.NewWorld()
	grid := engine.NewMapGrid(10, 10)

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	scapeID := ecs.ComponentID[components.ScapegoatComponent](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	esoID := ecs.ComponentID[components.EsotericMarker](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	belID := ecs.ComponentID[components.BeliefComponent](&world)

	// Step 1: Create a traumatized jurisdiction
	jurEnt := world.NewEntity(posID, jurID, scapeID, affID)

	jurPos := (*components.Position)(world.Get(jurEnt, posID))
	jurPos.X = 5
	jurPos.Y = 5

	jur := (*components.JurisdictionComponent)(world.Get(jurEnt, jurID))
	jur.Trauma = 25 // High trauma to trigger scapegoating
	jur.RadiusSquared = 100.0

	// Step 2: Create a Caster NPC (The Witch)
	casterEnt := world.NewEntity(posID, jobID, memID, idID, affID, npcID, belID)

	casterPos := (*components.Position)(world.Get(casterEnt, posID))
	casterPos.X = 5
	casterPos.Y = 5

	job := (*components.JobComponent)(world.Get(casterEnt, jobID))
	job.JobID = components.JobCaster

	ident := (*components.Identity)(world.Get(casterEnt, idID))
	ident.BaseTraits |= components.TraitEsoteric // Explicitly give the trait

	// Add an esoteric marker directly to simulate a successful cast
	world.Add(casterEnt, esoID)
	eso := (*components.EsotericMarker)(world.Get(casterEnt, esoID))
	eso.Active = true

	// Provide mana to cast
	grid.Mana[55].Value = 100

	// Initialize systems
	scapeSystem := NewScapegoatSystem()
	scapeSystem.tickCounter = 9 // Will execute on next tick (10)

	justiceSystem := NewJusticeSystem(&world, nil) // Mock hook graph nil

	// Pre-check
	scapeComp := (*components.ScapegoatComponent)(world.Get(jurEnt, scapeID))
	if scapeComp.TargetEsoteric {
		t.Fatalf("TargetEsoteric should be false before execution")
	}

	// Execution Step 1: Scapegoat System triggers the Witch Hunt
	scapeSystem.Update(&world)

	if !scapeComp.TargetEsoteric {
		t.Fatalf("ScapegoatSystem failed to target esoterics. j.Comp.TargetEsoteric is false")
	}

	if jur.Trauma != 15 {
		t.Fatalf("ScapegoatSystem failed to apply catharsis. Expected Trauma 15, got %d", jur.Trauma)
	}

	// Execution Step 2: Justice System executes the Witch Hunt
	justiceSystem.Update(&world)

	// Verify CrimeMarker was appended
	if !world.Has(casterEnt, crimeID) {
		t.Fatalf("JusticeSystem failed to tag the Caster with a CrimeMarker")
	}

	// Verify Memory was appended correctly
	mem := (*components.Memory)(world.Get(casterEnt, memID))
	foundEsotericCrime := false
	for i := 0; i < len(mem.Events); i++ {
		if mem.Events[i].InteractionType == components.InteractionEsoteric {
			foundEsotericCrime = true
			break
		}
	}

	if !foundEsotericCrime {
		t.Fatalf("JusticeSystem failed to log InteractionEsoteric into the NPC's memory buffer")
	}
}
