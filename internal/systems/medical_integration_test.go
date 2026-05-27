package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestMedicalSystem_Integration(t *testing.T) {
	// Initialize Engine and World
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	pathQueue := engine.NewPathRequestQueue(100, 1)

	// Register Components
	jID := ecs.ComponentID[components.JobComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	sys := NewMedicalSystem(&world, pathQueue, hooks)

	// Create a Doctor
	docEnt := world.NewEntity(npcID, idID, jID, posID, pathID)

	docId := (*components.Identity)(world.Get(docEnt, idID))
	docId.ID = 1

	docJob := (*components.JobComponent)(world.Get(docEnt, jID))
	docJob.JobID = components.JobDoctor

	docPos := (*components.Position)(world.Get(docEnt, posID))
	docPos.X = 10.0
	docPos.Y = 10.0

	// Create a wealthy patient who is injured
	patient1Ent := world.NewEntity(npcID, idID, posID, vitalsID, treasuryID)

	p1Id := (*components.Identity)(world.Get(patient1Ent, idID))
	p1Id.ID = 2

	p1Pos := (*components.Position)(world.Get(patient1Ent, posID))
	p1Pos.X = 11.0
	p1Pos.Y = 11.0

	p1Vitals := (*components.VitalsComponent)(world.Get(patient1Ent, vitalsID))
	p1Vitals.Blood = 30.0
	p1Vitals.Pain = 50.0

	p1Treasury := (*components.TreasuryComponent)(world.Get(patient1Ent, treasuryID))
	p1Treasury.Wealth = 100.0


	// Create a poor patient who is in pain, further away
	patient2Ent := world.NewEntity(npcID, idID, posID, vitalsID, treasuryID)

	p2Id := (*components.Identity)(world.Get(patient2Ent, idID))
	p2Id.ID = 3

	p2Pos := (*components.Position)(world.Get(patient2Ent, posID))
	p2Pos.X = 20.0
	p2Pos.Y = 20.0

	p2Vitals := (*components.VitalsComponent)(world.Get(patient2Ent, vitalsID))
	p2Vitals.Blood = 100.0
	p2Vitals.Pain = 80.0

	p2Treasury := (*components.TreasuryComponent)(world.Get(patient2Ent, treasuryID))
	p2Treasury.Wealth = 0.0

	// Tick 1: Doctor should heal patient 1 since they are adjacent
	sys.Update(&world)

	if p1Vitals.Blood != 100.0 || p1Vitals.Pain != 0.0 {
		t.Errorf("Expected Patient 1 to be fully healed, got Blood: %f, Pain: %f", p1Vitals.Blood, p1Vitals.Pain)
	}

	if p1Treasury.Wealth != 80.0 {
		t.Errorf("Expected Patient 1 to have paid 20 wealth, remaining: %f", p1Treasury.Wealth)
	}

	// Verify no hook was generated for patient 1
	hookValue := hooks.GetHook(1, 2)
	if hookValue != 0 {
		t.Errorf("Expected no hook generated for paying patient, got: %d", hookValue)
	}

	// Tick 2: Doctor should now target patient 2 (since patient 1 is healed)
	// But doctor is at 10,10 and patient 2 is at 20,20 -> pathfind enqueue
	sys.Update(&world)

	// Simulate Pathfinding completing and moving the doctor adjacent to patient 2
	docPos.X = 21.0
	docPos.Y = 21.0

	// Tick 3: Doctor heals patient 2
	sys.Update(&world)

	if p2Vitals.Blood != 100.0 || p2Vitals.Pain != 0.0 {
		t.Errorf("Expected Patient 2 to be fully healed, got Blood: %f, Pain: %f", p2Vitals.Blood, p2Vitals.Pain)
	}

	if p2Treasury.Wealth != 0.0 {
		t.Errorf("Expected Patient 2 wealth to remain 0, got: %f", p2Treasury.Wealth)
	}

	// Verify hook was generated for patient 2 since they couldn't pay
	hookValue2 := hooks.GetHook(1, 3)
	if hookValue2 != -50 {
		t.Errorf("Expected negative hook of -50 for defaulting on medical debt, got: %d", hookValue2)
	}
}
