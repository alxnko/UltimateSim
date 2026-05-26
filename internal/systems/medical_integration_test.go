package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 68 - The Physical Medical Engine Integration Test
func TestMedicalSystem_ButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewMedicalSystem(&world, hooks)

	identID := ecs.ComponentID[components.Identity](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	// Spawn Patient
	patientEnt := world.NewEntity(identID, vitalsID, posID, needsID)
	pIdent := (*components.Identity)(world.Get(patientEnt, identID))
	pIdent.ID = 100

	pVitals := (*components.VitalsComponent)(world.Get(patientEnt, vitalsID))
	pVitals.Blood = 20.0
	pVitals.Pain = 50.0

	pPos := (*components.Position)(world.Get(patientEnt, posID))
	pPos.X = 10.0
	pPos.Y = 10.0

	pNeeds := (*components.Needs)(world.Get(patientEnt, needsID))
	pNeeds.Wealth = 0.0 // Poor patient, will generate debt resent

	// Spawn Doctor
	docEnt := world.NewEntity(identID, posID, needsID, jobID, pathID)
	dIdent := (*components.Identity)(world.Get(docEnt, identID))
	dIdent.ID = 200

	dPos := (*components.Position)(world.Get(docEnt, posID))
	dPos.X = 15.0 // Slightly away
	dPos.Y = 10.0

	dNeeds := (*components.Needs)(world.Get(docEnt, needsID))
	dNeeds.Wealth = 100.0

	dJob := (*components.JobComponent)(world.Get(docEnt, jobID))
	dJob.JobID = components.JobDoctor

	// Tick 1: Doctor should pathfind to patient
	sys.Update(&world)

	dPath := (*components.Path)(world.Get(docEnt, pathID))
	if dPath.TargetX != 10.0 || dPath.TargetY != 10.0 {
		t.Errorf("Doctor did not target patient: expected (10, 10), got (%f, %f)", dPath.TargetX, dPath.TargetY)
	}

	// Move Doctor to adjacent
	dPos.X = 11.0

	// Tick 2: Doctor is adjacent, heals patient, creates debt resentment
	sys.Update(&world)

	// Verify healing
	if pVitals.Blood != 100.0 || pVitals.Pain != 0.0 {
		t.Errorf("Patient was not healed: Blood=%f, Pain=%f", pVitals.Blood, pVitals.Pain)
	}

	// Verify butterfly effect (Hook)
	hookVal := hooks.GetHook(dIdent.ID, pIdent.ID)
	if hookVal != -50 {
		t.Errorf("Doctor did not generate -50 hook against poor patient, got: %d", hookVal)
	}
}
