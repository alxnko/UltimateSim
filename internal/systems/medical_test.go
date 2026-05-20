package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestMedicalSystem_Integration(t *testing.T) {
	engine.InitializeRNG([32]byte{1, 2, 3})

	tm := engine.NewTickManager(60)

	hooks := engine.NewSparseHookGraph()
	medicalSys := NewMedicalSystem(tm.World, hooks)

	tm.AddSystem(medicalSys, engine.PhaseResolution)

	world := tm.World

	posID := ecs.ComponentID[components.Position](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	pathID := ecs.ComponentID[components.Path](world)
	identID := ecs.ComponentID[components.Identity](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)

	// Create Injured Patient (Poor, to trigger negative hook)
	patient := world.NewEntity(posID, vitalsID, needsID, identID)
	pPos := (*components.Position)(world.Get(patient, posID))
	pPos.X, pPos.Y = 10, 10
	pVitals := (*components.VitalsComponent)(world.Get(patient, vitalsID))
	pVitals.Blood = 20.0
	pVitals.Pain = 50.0
	pNeeds := (*components.Needs)(world.Get(patient, needsID))
	pNeeds.Wealth = 10.0 // Insufficient for the 50.0 fee
	pIdent := (*components.Identity)(world.Get(patient, identID))
	pIdent.ID = 101

	// Create JobDoctor NPC
	doctor := world.NewEntity(posID, jobID, pathID, identID, needsID)
	dPos := (*components.Position)(world.Get(doctor, posID))
	dPos.X, dPos.Y = 10, 11 // Within distSq <= 2.0 (1.0)
	dJob := (*components.JobComponent)(world.Get(doctor, jobID))
	dJob.JobID = components.JobDoctor
	dPath := (*components.Path)(world.Get(doctor, pathID))
	dPath.HasPath = false
	dIdent := (*components.Identity)(world.Get(doctor, identID))
	dIdent.ID = 102
	dNeeds := (*components.Needs)(world.Get(doctor, needsID))
	dNeeds.Wealth = 100.0

	// Tick the system
	tm.Tick()

	// Verify healing
	if pVitals.Blood != 100.0 || pVitals.Pain != 0.0 {
		t.Errorf("Patient was not healed correctly. Blood: %f, Pain: %f", pVitals.Blood, pVitals.Pain)
	}

	// Verify economic exchange
	if pNeeds.Wealth != 0.0 {
		t.Errorf("Patient wealth should be drained to 0.0, got %f", pNeeds.Wealth)
	}
	if dNeeds.Wealth != 110.0 { // 100 base + 10 collected
		t.Errorf("Doctor did not collect remaining wealth, got %f", dNeeds.Wealth)
	}

	// Verify negative hook generated against patient due to unpaid medical debt
	hook := hooks.GetHook(102, 101)
	if hook != -20 {
		t.Errorf("Expected hook of -20 for unpaid medical debt, got %d", hook)
	}
}
