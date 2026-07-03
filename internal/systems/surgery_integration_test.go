package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestSurgerySystem_AmputationAndHooks(t *testing.T) {
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(100, 0) // synchronous resolution
	hooks := engine.NewSparseHookGraph()

	sys := NewSurgerySystem(&world, pathQueue, hooks)

	// Create Patient
	patient := world.NewEntity()
	posID := ecs.ComponentID[components.Position](&world)
	anatomyID := ecs.ComponentID[components.AnatomyComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	sanityID := ecs.ComponentID[components.SanityComponent](&world)

	world.Add(patient, posID, anatomyID, vitalsID, needsID, identID, sanityID)

	pPos := (*components.Position)(world.Get(patient, posID))
	pPos.X, pPos.Y = 10.0, 10.0

	pAnatomy := (*components.AnatomyComponent)(world.Get(patient, anatomyID))
	pAnatomy.InfectedLimbs = 1 // lowest bit set (e.g. left arm)
	pAnatomy.InfectionProg = 99.0

	pVitals := (*components.VitalsComponent)(world.Get(patient, vitalsID))
	pVitals.Blood = 100.0

	pNeeds := (*components.Needs)(world.Get(patient, needsID))
	pNeeds.Wealth = 20.0 // Bankrupt (cannot afford 50 fee)

	pIdent := (*components.Identity)(world.Get(patient, identID))
	pIdent.ID = 1

	pSanity := (*components.SanityComponent)(world.Get(patient, sanityID))
	pSanity.MaxStress = 100.0
	pSanity.Stress = 0.0

	// Create Doctor (far away)
	doctor := world.NewEntity()
	jobID := ecs.ComponentID[components.JobComponent](&world)

	world.Add(doctor, jobID, posID, needsID, identID)

	dPos := (*components.Position)(world.Get(doctor, posID))
	dPos.X, dPos.Y = 15.0, 15.0

	dJob := (*components.JobComponent)(world.Get(doctor, jobID))
	dJob.JobID = components.JobDoctor

	dNeeds := (*components.Needs)(world.Get(doctor, needsID))
	dNeeds.Wealth = 0.0

	dIdent := (*components.Identity)(world.Get(doctor, identID))
	dIdent.ID = 2

	// Tick 1: Patient infection progresses and crosses lethal threshold, blood drains. Doctor pathfinds.
	sys.Update(&world)

	if pAnatomy.InfectionProg != 99.5 {
		t.Errorf("Expected InfectionProg 99.5, got %f", pAnatomy.InfectionProg)
	}
	if pVitals.Blood != 100.0 {
		t.Errorf("Expected Blood 100.0, got %f", pVitals.Blood) // Threshold is 100.0, so at 99.5 no drain
	}

	// Note: engine.NewPathRequestQueue(100, 0) drops requests into requests channel. In synchronous mode (0 workers),
	// they stay there and don't make it to ResultsChannel.

	// Move Doctor adjacent
	dPos.X, dPos.Y = 11.0, 10.0

	// Tick 2: Patient crosses lethal threshold (99.5 + 0.5 = 100.0). Blood drains. Doctor is adjacent and amputates.
	sys.Update(&world)

	// Check Biology Bridge (Infection)
	if pVitals.Blood != 98.0 {
		t.Errorf("Expected Blood 98.0 due to infection, got %f", pVitals.Blood)
	}

	// Check Anatomy Bridge
	if pAnatomy.InfectedLimbs != 0 {
		t.Errorf("Expected 0 InfectedLimbs, got %d", pAnatomy.InfectedLimbs)
	}
	if pAnatomy.MissingLimbs != 1 {
		t.Errorf("Expected 1 MissingLimbs, got %d", pAnatomy.MissingLimbs)
	}
	if pAnatomy.InfectionProg != 0.0 {
		t.Errorf("Expected InfectionProg to reset to 0.0, got %f", pAnatomy.InfectionProg)
	}

	// Check Psychology Bridge
	if pSanity.Stress != 50.0 {
		t.Errorf("Expected Stress to jump to 50.0 after amputation, got %f", pSanity.Stress)
	}

	// Check Economy Bridge
	if pNeeds.Wealth != 0.0 {
		t.Errorf("Expected Patient Wealth 0.0, got %f", pNeeds.Wealth)
	}
	if dNeeds.Wealth != 20.0 {
		t.Errorf("Expected Doctor to extract 20.0 Wealth, got %f", dNeeds.Wealth)
	}

	// Check Hierarchy/Justice Bridge (Hooks)
	// Expect Doctor (2) to resent Patient (1) for -50
	hookVal := hooks.GetHook(2, 1)
	if hookVal != -50 {
		t.Errorf("Expected hook value -50 from Doctor to Patient, got %d", hookVal)
	}
}
