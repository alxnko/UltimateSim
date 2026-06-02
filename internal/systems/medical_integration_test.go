package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestMedicalSystem_Integration(t *testing.T) {
	// Initialize World
	world := ecs.NewWorld()

	// Register components explicitly
	_ = ecs.ComponentID[components.NPC](&world)
	_ = ecs.ComponentID[components.JobComponent](&world)
	_ = ecs.ComponentID[components.Position](&world)
	_ = ecs.ComponentID[components.Path](&world)
	_ = ecs.ComponentID[components.VitalsComponent](&world)
	_ = ecs.ComponentID[components.Identity](&world)
	_ = ecs.ComponentID[components.TreasuryComponent](&world)
	_ = ecs.ComponentID[components.CombatMarker](&world)

	// Initialize Engine
	pathQueue := engine.NewPathRequestQueue(100, 0) // No workers, sync resolution
	hookGraph := engine.NewSparseHookGraph()

	// Initialize Systems
	medicalSys := NewMedicalSystem(&world, pathQueue, hookGraph)

	// Build Entities
	// Patient
	patient := world.NewEntity()
	world.Add(patient,
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.TreasuryComponent](&world),
	)

	pPos := (*components.Position)(world.Get(patient, ecs.ComponentID[components.Position](&world)))
	pPos.X = 10.0
	pPos.Y = 10.0

	pVitals := (*components.VitalsComponent)(world.Get(patient, ecs.ComponentID[components.VitalsComponent](&world)))
	pVitals.Blood = 30.0 // Injured (< 50)
	pVitals.Pain = 50.0  // In pain (> 20)

	pIdent := (*components.Identity)(world.Get(patient, ecs.ComponentID[components.Identity](&world)))
	pIdent.ID = 1

	pTreas := (*components.TreasuryComponent)(world.Get(patient, ecs.ComponentID[components.TreasuryComponent](&world)))
	pTreas.Wealth = 5.0 // Too poor to afford the 10.0 treatment cost

	// Doctor
	doctor := world.NewEntity()
	world.Add(doctor,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Path](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.TreasuryComponent](&world),
	)

	dJob := (*components.JobComponent)(world.Get(doctor, ecs.ComponentID[components.JobComponent](&world)))
	dJob.JobID = components.JobDoctor

	dPos := (*components.Position)(world.Get(doctor, ecs.ComponentID[components.Position](&world)))
	dPos.X = 0.0 // Far away (distSq = 200)
	dPos.Y = 0.0

	dPath := (*components.Path)(world.Get(doctor, ecs.ComponentID[components.Path](&world)))
	dPath.HasPath = false

	dIdent := (*components.Identity)(world.Get(doctor, ecs.ComponentID[components.Identity](&world)))
	dIdent.ID = 2

	dTreas := (*components.TreasuryComponent)(world.Get(doctor, ecs.ComponentID[components.TreasuryComponent](&world)))
	dTreas.Wealth = 100.0


	// Tick 1: Doctor detects patient and queues path
	medicalSys.Update(&world)

	// Verify path queue was hit
	if !dPath.HasPath {
		t.Fatalf("Expected Doctor to detect patient and enqueue path, HasPath is false")
	}
	if dPath.TargetX != 10.0 || dPath.TargetY != 10.0 {
		t.Fatalf("Expected path target to be 10,10. Got %f,%f", dPath.TargetX, dPath.TargetY)
	}

	// Consume the path request (simulation of Path Queue worker)


	// Teleport Doctor to be adjacent to patient
	dPos.X = 10.5
	dPos.Y = 10.0

	// Tick 2: Doctor heals patient
	medicalSys.Update(&world)

	// Verify healing
	if pVitals.Blood != 55.0 {
		t.Errorf("Expected patient Blood to increase by 25 to 55.0. Got %f", pVitals.Blood)
	}
	if pVitals.Pain != 25.0 {
		t.Errorf("Expected patient Pain to decrease by 25 to 25.0. Got %f", pVitals.Pain)
	}

	// Verify Economics / Justice hook
	if hookGraph.GetHook(dIdent.ID, pIdent.ID) != 50 {
		t.Errorf("Expected Doctor to gain 50 Hooks on patient due to patient lacking wealth. Got %d", hookGraph.GetHook(dIdent.ID, pIdent.ID))
	}
	if dTreas.Wealth != 100.0 {
		t.Errorf("Expected Doctor Wealth to remain unchanged due to poor patient. Got %f", dTreas.Wealth)
	}

	// Make patient rich
	pTreas.Wealth = 50.0

	// Tick 3: Doctor continues healing
	medicalSys.Update(&world)

	// Verify healing 2
	if pVitals.Blood != 80.0 {
		t.Errorf("Expected patient Blood to increase by 25 to 80.0. Got %f", pVitals.Blood)
	}
	if pVitals.Pain != 0.0 {
		t.Errorf("Expected patient Pain to decrease by 25 to 0.0. Got %f", pVitals.Pain)
	}

	// Verify payment
	if pTreas.Wealth != 40.0 {
		t.Errorf("Expected patient Wealth to decrease by 10 to 40.0. Got %f", pTreas.Wealth)
	}
	if dTreas.Wealth != 110.0 {
		t.Errorf("Expected Doctor Wealth to increase by 10 to 110.0. Got %f", dTreas.Wealth)
	}
}
