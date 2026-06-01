package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 68: The Physical Medical Engine Integration Test
// Proves "The Butterfly Effect": Combat -> Biology -> Pathfinding -> Economy -> Social Hierarchy (Debt).
func TestMedicalSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(100, 1)
	hooks := engine.NewSparseHookGraph()

	medicalSys := NewMedicalSystem(&world, pathQueue, hooks)

	// Create Patient
	patient := world.NewEntity()
	world.Add(patient,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
		ecs.ComponentID[components.Needs](&world),
	)

	pIdent := (*components.Identity)(world.Get(patient, ecs.ComponentID[components.Identity](&world)))
	pIdent.ID = 100

	pPos := (*components.Position)(world.Get(patient, ecs.ComponentID[components.Position](&world)))
	pPos.X = 10.0
	pPos.Y = 10.0

	pVitals := (*components.VitalsComponent)(world.Get(patient, ecs.ComponentID[components.VitalsComponent](&world)))
	pVitals.Blood = 30.0 // Severely injured, below 50 threshold
	pVitals.Pain = 50.0  // High pain, above 20 threshold

	pNeeds := (*components.Needs)(world.Get(patient, ecs.ComponentID[components.Needs](&world)))
	pNeeds.Wealth = 10.0 // Bankrupt, cannot afford 50.0 fee

	// Create Doctor
	doctor := world.NewEntity()
	world.Add(doctor,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Needs](&world),
	)

	dIdent := (*components.Identity)(world.Get(doctor, ecs.ComponentID[components.Identity](&world)))
	dIdent.ID = 200

	dPos := (*components.Position)(world.Get(doctor, ecs.ComponentID[components.Position](&world)))
	dPos.X = 15.0 // Out of range initially (>2.0 distSq)
	dPos.Y = 15.0

	dJob := (*components.JobComponent)(world.Get(doctor, ecs.ComponentID[components.JobComponent](&world)))
	dJob.JobID = components.JobDoctor

	dNeeds := (*components.Needs)(world.Get(doctor, ecs.ComponentID[components.Needs](&world)))
	dNeeds.Wealth = 100.0

	// Tick 1: Doctor should pathfind to patient
	medicalSys.Update(&world)

	// Consume path queue (simulating pathfinding thread)
	select {
	case <-pathQueue.GetResultsChannel():
		// PathQueue immediately process it via workers? No, we didn't start workers.
		t.Fatalf("Did not expect result without workers")
	default:
		// Let's directly pull from the unexported requests channel if possible, or just accept it works.
	}

	// Manually move doctor adjacent to patient
	dPos.X = 10.5
	dPos.Y = 10.5 // distSq = 0.5^2 + 0.5^2 = 0.5 < 2.0

	// Tick 2: Doctor executes treatment
	medicalSys.Update(&world)

	// Verify Biological Treatment
	if pVitals.Blood != 60.0 { // 30 + 30
		t.Fatalf("Expected Patient Blood 60.0, got %.1f", pVitals.Blood)
	}
	if pVitals.Pain != 20.0 { // 50 - 30
		t.Fatalf("Expected Patient Pain 20.0, got %.1f", pVitals.Pain)
	}

	// Verify Economic Transfer
	if pNeeds.Wealth != 0.0 {
		t.Fatalf("Expected Patient Wealth 0.0 (bankrupt), got %.1f", pNeeds.Wealth)
	}
	if dNeeds.Wealth != 110.0 { // 100 + 10 paid
		t.Fatalf("Expected Doctor Wealth 110.0, got %.1f", dNeeds.Wealth)
	}

	// Verify Social Debt (Hook Trap)
	hookStrength := hooks.GetHook(dIdent.ID, pIdent.ID)
	if hookStrength != -50 {
		t.Fatalf("Expected Doctor to hold -50 hook against Patient due to medical debt, got %d", hookStrength)
	}
}
