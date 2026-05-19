package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestMedicalSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewMedicalSystem(&world, hooks)

	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	// Create a patient needing medical attention (cannot pay fee)
	patient := world.NewEntity(posID, vitalsID, needsID, idID)
	pPos := (*components.Position)(world.Get(patient, posID))
	pPos.X, pPos.Y = 10, 10

	pVitals := (*components.VitalsComponent)(world.Get(patient, vitalsID))
	pVitals.Blood = 10.0 // Dangerously low
	pVitals.Pain = 80.0

	pNeeds := (*components.Needs)(world.Get(patient, needsID))
	pNeeds.Wealth = 50.0 // Less than the 100.0 fee

	pIdent := (*components.Identity)(world.Get(patient, idID))
	pIdent.ID = 1001

	// Create a Doctor
	doctor := world.NewEntity(posID, jobID, idID, pathID, needsID)
	dPos := (*components.Position)(world.Get(doctor, posID))
	dPos.X, dPos.Y = 50, 50 // Far away

	dJob := (*components.JobComponent)(world.Get(doctor, jobID))
	dJob.JobID = components.JobDoctor

	dIdent := (*components.Identity)(world.Get(doctor, idID))
	dIdent.ID = 2001

	dNeeds := (*components.Needs)(world.Get(doctor, needsID))
	dNeeds.Wealth = 0.0

	// Step 1: Update sys - Doctor should detect patient and pathfind
	sys.Update(&world)

	dPath := (*components.Path)(world.Get(doctor, pathID))
	if dPath.TargetX != 10.0 || dPath.TargetY != 10.0 {
		t.Fatalf("Doctor failed to target the patient. Expected path target (10,10), got (%f,%f)", dPath.TargetX, dPath.TargetY)
	}

	// Step 2: Teleport doctor adjacent to patient and update again to trigger healing
	dPos.X, dPos.Y = 11, 10
	sys.Update(&world)

	// Verify healing
	pVitalsNew := (*components.VitalsComponent)(world.Get(patient, vitalsID))
	if pVitalsNew.Blood != 100.0 {
		t.Errorf("Patient Blood was not fully healed. Expected 100, got %f", pVitalsNew.Blood)
	}
	if pVitalsNew.Pain != 0.0 {
		t.Errorf("Patient Pain was not fully reduced. Expected 0, got %f", pVitalsNew.Pain)
	}

	// Verify Economy
	pNeedsNew := (*components.Needs)(world.Get(patient, needsID))
	if pNeedsNew.Wealth != 0.0 {
		t.Errorf("Patient Wealth was not drained. Expected 0, got %f", pNeedsNew.Wealth)
	}

	dNeedsNew := (*components.Needs)(world.Get(doctor, needsID))
	if dNeedsNew.Wealth != 50.0 {
		t.Errorf("Doctor did not collect the partial fee. Expected 50, got %f", dNeedsNew.Wealth)
	}

	// Verify Justice/Hook Bridge
	hookVal := hooks.GetHook(2001, 1001)
	if hookVal != -50 {
		t.Errorf("Expected doctor (2001) to hold a -50 hook against patient (1001) for unpaid debt, got %d", hookVal)
	}
}
