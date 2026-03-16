package systems_test

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 33: The Refugee Crisis (Systemic Emergence) Deterministic Tests
func TestRefugeeMigration_ButterflyEffect(t *testing.T) {
	world1, hook1, target1 := setupRefugeeTestWorld()
	world2, hook2, target2 := setupRefugeeTestWorld()

	sys1 := systems.NewRefugeeMigrationSystem(hook1)
	sys2 := systems.NewRefugeeMigrationSystem(hook2)

	move1 := systems.NewMovementSystem(&world1, engine.NewMapGrid(10, 10), engine.NewCalendar())
	move2 := systems.NewMovementSystem(&world2, engine.NewMapGrid(10, 10), engine.NewCalendar())

	// Tick until they arrive and integrate/reject (takes ~2-3 ticks based on velocity)
	for i := 0; i < 5; i++ {
		move1.Update(&world1)
		sys1.Update(&world1)

		move2.Update(&world2)
		sys2.Update(&world2)
	}

	// Because the village is traumatized and they speak a different language, they should be rejected
	// and a -50 hook should be seeded against target1.

	incoming1 := hook1.GetAllIncomingHooks(target1)
	incoming2 := hook2.GetAllIncomingHooks(target2)

	if len(incoming1) == 0 {
		t.Fatalf("Expected at least one incoming hook from rejected refugees")
	}

	hookVal1 := 0
	for _, v := range incoming1 {
		hookVal1 = v
		break
	}

	if hookVal1 != -50 {
		t.Errorf("Expected hook value to be -50, got %d", hookVal1)
	}

	// Determinism check
	if len(incoming1) != len(incoming2) {
		t.Errorf("Determinism failure: World1 spawned %d hooks, World2 spawned %d hooks", len(incoming1), len(incoming2))
	}
}

func setupRefugeeTestWorld() (ecs.World, *engine.SparseHookGraph, uint64) {
	// Re-initialize RNG for test isolation/determinism
	var seed [32]byte
	seed[0] = 1
	engine.InitializeRNG(seed)
	world := ecs.NewWorld()
	hook := engine.NewSparseHookGraph()

	// IDs
	posID := ecs.ComponentID[components.Position](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)

	velID := ecs.ComponentID[components.Velocity](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	refClusterID := ecs.ComponentID[components.RefugeeCluster](&world)
	refDataID := ecs.ComponentID[components.RefugeeData](&world)

	// Create Target Traumatized Village
	targetEnt := world.NewEntity()
	world.Add(targetEnt, villageID, posID, popID, storageID, jurID, idID, cultureID)

	vPos := (*components.Position)(world.Get(targetEnt, posID))
	vPos.X = 5.0
	vPos.Y = 5.0

	vID := (*components.Identity)(world.Get(targetEnt, idID))
	vID.ID = 100 // Target ID for hooks

	vJur := (*components.JurisdictionComponent)(world.Get(targetEnt, jurID))
	vJur.Trauma = 100 // Highly traumatized

	vCult := (*components.CultureComponent)(world.Get(targetEnt, cultureID))
	vCult.LanguageID = 1 // Local language

	vStore := (*components.StorageComponent)(world.Get(targetEnt, storageID))
	vStore.Food = 1000 // Plenty of food, but rejected due to trauma

	vPop := (*components.PopulationComponent)(world.Get(targetEnt, popID))
	vPop.Count = 10
	vPop.Citizens = make([]components.CitizenData, 10)

	// Create Refugee Cluster
	refEnt := world.NewEntity()
	world.Add(refEnt, posID, velID, pathID, refClusterID, refDataID)

	rPos := (*components.Position)(world.Get(refEnt, posID))
	rPos.X = 4.9 // start them right on top of it so they arrive on tick 1
	rPos.Y = 4.9

	rVel := (*components.Velocity)(world.Get(refEnt, velID))
	rVel.X = 0.0
	rVel.Y = 0.0

	rPath := (*components.Path)(world.Get(refEnt, pathID))
	rPath.TargetX = 5.0
	rPath.TargetY = 5.0

	rData := (*components.RefugeeData)(world.Get(refEnt, refDataID))
	rData.Count = 5
	rData.Culture.LanguageID = 2 // Foreign language
	rData.Citizens = make([]components.CitizenData, 5)

	return world, hook, vID.ID
}
