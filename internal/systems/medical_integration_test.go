package systems

import (
	"testing"

	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 68: The Physical Medical Engine Integration Test
func TestMedicalSystem_Integration(t *testing.T) {
	// 1. Setup World & Managers
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(10, 2)
	pathQueue.StartWorkers()
	defer pathQueue.Close()

	hookGraph := engine.NewSparseHookGraph()

	medicalSys := NewMedicalSystem(&world, pathQueue, hookGraph)

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	// 2. Create Injured Patient (No Wealth)
	patientEntity := world.NewEntity(npcID, posID, vitalsID, treasuryID, identID)

	pPos := (*components.Position)(world.Get(patientEntity, posID))
	pPos.X, pPos.Y = 10.0, 10.0

	pVitals := (*components.VitalsComponent)(world.Get(patientEntity, vitalsID))
	pVitals.Blood = 20.0
	pVitals.Pain = 50.0

	pTreasury := (*components.TreasuryComponent)(world.Get(patientEntity, treasuryID))
	pTreasury.Wealth = 0.0

	pIdent := (*components.Identity)(world.Get(patientEntity, identID))
	pIdent.ID = 100

	// 3. Create Doctor
	doctorEntity := world.NewEntity(npcID, posID, jobID, pathID, identID)

	dPos := (*components.Position)(world.Get(doctorEntity, posID))
	dPos.X, dPos.Y = 10.0, 11.0

	dJob := (*components.JobComponent)(world.Get(doctorEntity, jobID))
	dJob.JobID = components.JobDoctor

	dIdent := (*components.Identity)(world.Get(doctorEntity, identID))
	dIdent.ID = 200

	// 4. Tick Medical System
	medicalSys.Update()

	// 5. Verify biological healing
	vitalsMapper := ecs.ComponentID[components.VitalsComponent](&world)
	patientVitals := (*components.VitalsComponent)(world.Get(patientEntity, vitalsMapper))

	if patientVitals.Blood != 100.0 {
		t.Errorf("Expected Blood to be restored to 100, got %f", patientVitals.Blood)
	}
	if patientVitals.Pain != 0.0 {
		t.Errorf("Expected Pain to be reduced to 0, got %f", patientVitals.Pain)
	}

	// 6. Verify economic/social hook (The Butterfly Effect)
	// The fee is 50.0. The patient had 0.0. Therefore, unpaid debt is 50.0.
	// The doctor should log a negative hook against the patient equal to -50.
	hookValue := hookGraph.GetHook(200, 100)
	if hookValue != -50 {
		t.Errorf("Expected hook value of -50, got %d", hookValue)
	}
}
