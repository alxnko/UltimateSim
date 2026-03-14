package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 24.1: The Labor Union Engine (Systemic Emergence) Deterministic Tests
func TestLaborUnionSystem_Integration(t *testing.T) {
	world1, hook1 := setupLaborUnionTestWorld()
	sys1 := NewLaborUnionSystem(&world1, hook1)
	sys1.Update(&world1)

	world2, hook2 := setupLaborUnionTestWorld()
	sys2 := NewLaborUnionSystem(&world2, hook2)
	sys2.Update(&world2)

	// Validate deterministic execution of negative hook insertion
	strikerID := uint64(10)
	scabID := uint64(20)
	employerID := uint64(100)

	hookVal1 := hook1.GetHook(strikerID, scabID)
	if hookVal1 != -50 {
		t.Fatalf("Expected Scab Hook to be -50, got %d", hookVal1)
	}

	empHookVal1 := hook1.GetHook(strikerID, employerID)
	if empHookVal1 != -10 {
		t.Fatalf("Expected Employer Hook to be -10, got %d", empHookVal1)
	}

	hookVal2 := hook2.GetHook(strikerID, scabID)
	if hookVal1 != hookVal2 {
		t.Fatalf("Determinism Failure: Run 1 (%d) != Run 2 (%d)", hookVal1, hookVal2)
	}
}

func setupLaborUnionTestWorld() (ecs.World, *engine.SparseHookGraph) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// Component IDs
	idID := ecs.ComponentID[components.Identity](&world)
	strikeID := ecs.ComponentID[components.StrikeMarker](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// 1. Create a Striker Entity
	strikerEnt := world.NewEntity(idID, strikeID)
	sIdent := (*components.Identity)(world.Get(strikerEnt, idID))
	sIdent.ID = 10
	marker := (*components.StrikeMarker)(world.Get(strikerEnt, strikeID))
	marker.TargetEmployerID = 100

	// 2. Create an Employer Entity (Identity only, for ID tracking)
	empEnt := world.NewEntity(idID)
	eIdent := (*components.Identity)(world.Get(empEnt, idID))
	eIdent.ID = 100

	// 3. Create a "Scab" Entity (Employed by struck business)
	scabEnt := world.NewEntity(idID, npcID, jobID)
	scabIdent := (*components.Identity)(world.Get(scabEnt, idID))
	scabIdent.ID = 20
	scabJob := (*components.JobComponent)(world.Get(scabEnt, jobID))
	scabJob.EmployerID = 100

	return world, hookGraph
}
