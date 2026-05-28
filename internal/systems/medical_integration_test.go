package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 68: The Physical Medical Engine
// Tests that a JobDoctor physically pathfinds to an injured NPC,
// heals them, and correctly handles wealth transfer and negative hook generation
// when the patient cannot afford the treatment (Butterfly Effect).
func TestMedicalSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	hooks := engine.NewSparseHookGraph()
	medSys := NewMedicalSystem(&world, hooks)

	// Create Doctor
	docEnt := world.NewEntity(npcID, identID, posID, jobID, pathID, needsID)
	docPos := (*components.Position)(world.Get(docEnt, posID))
	docPos.X, docPos.Y = 10.0, 10.0
	docIdent := (*components.Identity)(world.Get(docEnt, identID))
	docIdent.ID = 100
	docJob := (*components.JobComponent)(world.Get(docEnt, jobID))
	docJob.JobID = components.JobDoctor
	docNeeds := (*components.Needs)(world.Get(docEnt, needsID))
	docNeeds.Wealth = 100.0

	// Create wealthy patient (should pay)
	p1Ent := world.NewEntity(npcID, identID, posID, vitalsID, needsID)
	p1Pos := (*components.Position)(world.Get(p1Ent, posID))
	p1Pos.X, p1Pos.Y = 11.0, 10.0 // Adjacent to doctor
	p1Ident := (*components.Identity)(world.Get(p1Ent, identID))
	p1Ident.ID = 101
	p1Vitals := (*components.VitalsComponent)(world.Get(p1Ent, vitalsID))
	p1Vitals.Blood = 40.0 // Injured (threshold is < 50)
	p1Vitals.Pain = 0.0
	p1Needs := (*components.Needs)(world.Get(p1Ent, needsID))
	p1Needs.Wealth = 100.0

	// Create poor patient (should generate hook)
	p2Ent := world.NewEntity(npcID, identID, posID, vitalsID, needsID)
	p2Pos := (*components.Position)(world.Get(p2Ent, posID))
	p2Pos.X, p2Pos.Y = 20.0, 20.0 // Far away
	p2Ident := (*components.Identity)(world.Get(p2Ent, identID))
	p2Ident.ID = 102
	p2Vitals := (*components.VitalsComponent)(world.Get(p2Ent, vitalsID))
	p2Vitals.Blood = 100.0
	p2Vitals.Pain = 50.0 // Injured (threshold is > 20)
	p2Needs := (*components.Needs)(world.Get(p2Ent, needsID))
	p2Needs.Wealth = 10.0 // Cannot afford 50.0 cost

	// Tick 1: Doctor should heal adjacent wealthy patient
	medSys.Update(&world)

	if p1Vitals.Blood != 60.0 {
		t.Errorf("Expected wealthy patient Blood to be 60.0, got %f", p1Vitals.Blood)
	}
	if p1Needs.Wealth != 50.0 {
		t.Errorf("Expected wealthy patient Wealth to be 50.0, got %f", p1Needs.Wealth)
	}
	if docNeeds.Wealth != 150.0 {
		t.Errorf("Expected doctor Wealth to be 150.0, got %f", docNeeds.Wealth)
	}

	// Tick 2: Doctor should pathfind to poor patient
	medSys.Update(&world)

	docPath := (*components.Path)(world.Get(docEnt, pathID))
	if docPath.TargetX != 20.0 || docPath.TargetY != 20.0 {
		t.Errorf("Expected doctor to target poor patient at (20, 20), got (%f, %f)", docPath.TargetX, docPath.TargetY)
	}

	// Move doctor adjacent to poor patient
	docPos.X, docPos.Y = 21.0, 20.0

	// Tick 3: Doctor heals poor patient, extracts available wealth, and generates hook
	medSys.Update(&world)

	if p2Vitals.Pain != 30.0 {
		t.Errorf("Expected poor patient Pain to be 30.0, got %f", p2Vitals.Pain)
	}
	if p2Needs.Wealth != 0.0 {
		t.Errorf("Expected poor patient Wealth to be 0.0, got %f", p2Needs.Wealth)
	}
	if docNeeds.Wealth != 160.0 { // 150 + 10
		t.Errorf("Expected doctor Wealth to be 160.0, got %f", docNeeds.Wealth)
	}

	hookValue := hooks.GetHook(docIdent.ID, p2Ident.ID)
	if hookValue != -50 {
		t.Errorf("Expected hook value of -50 from doctor to poor patient, got %d", hookValue)
	}
}
