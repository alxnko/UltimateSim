package systems_test

import (
	"testing"
	"time"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

func TestWanderSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)
	// We use a small queue buffer since this is a limited test
	pathQueue := engine.NewPathRequestQueue(10, 1)
	pathQueue.StartWorkers()
	defer pathQueue.Close()

	// Seed some food
	mapGrid.Resources[5*10+5].FoodValue = 10 // (5, 5)

	// Phase 31: Update MapGrid Cache so WanderSystem finds it
	// `WanderSystem` uses `step := 8` to iterate over `FoodCache`. It checks index 0, 8, 16...
	// If `FoodCache` only has 1 element, it checks index 0, which is `5*10+5`. That's fine.
	mapGrid.FoodCache = append(mapGrid.FoodCache, 5*10+5)

	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	wanderSys := systems.NewWanderSystem(&world, mapGrid, pathQueue)

	// Entity 1: Needs food
	entity1 := world.NewEntity(posID, idID, needsID, pathID)
	pos1 := (*components.Position)(world.Get(entity1, posID))
	pos1.X, pos1.Y = 0, 0
	id1 := (*components.Identity)(world.Get(entity1, idID))
	id1.ID = 1
	id1.BaseTraits = 0
	needs1 := (*components.Needs)(world.Get(entity1, needsID))
	needs1.Food = 10.0 // Low food
	path1 := (*components.Path)(world.Get(entity1, pathID))
	path1.HasPath = false

	// Tick once to dispatch request
	for i := 0; i < 35; i++ {
		wanderSys.Update(&world)
		if path1.HasPath {
			break
		}
	}

	// Validate target selection occurred properly
	if !path1.HasPath {
		t.Fatalf("Entity 1 should have generated a path request to find food")
	}
	if path1.TargetX != 5 || path1.TargetY != 5 {
		t.Errorf("Entity 1 target should be (5, 5), got (%f, %f)", path1.TargetX, path1.TargetY)
	}

	// Wait briefly to allow async worker to process
	time.Sleep(100 * time.Millisecond)

	// Tick again to resolve queue
	wanderSys.Update(&world)

	// Verify path array populated
	if len(path1.Nodes) == 0 {
		t.Errorf("Entity 1 should have received path nodes from worker pool")
	}
}

func TestWanderSystem_Deterministic(t *testing.T) {
	runSim := func() float32 {
		world := ecs.NewWorld()
		mapGrid := engine.NewMapGrid(20, 20)
		pathQueue := engine.NewPathRequestQueue(100, 2)
		pathQueue.StartWorkers()
		defer pathQueue.Close()

		// Distribute resources
		for i := 0; i < 400; i++ {
			if i%7 == 0 {
				mapGrid.Resources[i].FoodValue = 5
			}
		}

		posID := ecs.ComponentID[components.Position](&world)
		idID := ecs.ComponentID[components.Identity](&world)
		needsID := ecs.ComponentID[components.Needs](&world)
		pathID := ecs.ComponentID[components.Path](&world)

		wanderSys := systems.NewWanderSystem(&world, mapGrid, pathQueue)

		// Spawn identical entities
		for i := uint64(1); i <= 10; i++ {
			entity := world.NewEntity(posID, idID, needsID, pathID)
			pos := (*components.Position)(world.Get(entity, posID))
			pos.X, pos.Y = float32(i), float32(i)

			id := (*components.Identity)(world.Get(entity, idID))
			id.ID = i
			// Every other entity is cautious
			if i%2 == 0 {
				id.BaseTraits = components.TraitCautious
			} else {
				id.BaseTraits = components.TraitRiskTaker
			}

			needs := (*components.Needs)(world.Get(entity, needsID))
			needs.Food = 20.0

			path := (*components.Path)(world.Get(entity, pathID))
			path.HasPath = false
		}

		wanderSys.Update(&world)
		// Wait for workers
		time.Sleep(200 * time.Millisecond)
		wanderSys.Update(&world)

		// Return sum of targets and path nodes
		var targetSum float32 = 0
		query := world.Query(ecs.All(pathID))
		for query.Next() {
			path := (*components.Path)(query.Get(pathID))
			targetSum += path.TargetX + path.TargetY
			// Include nodes to ensure queue drained properly
			targetSum += float32(len(path.Nodes))
			for _, node := range path.Nodes {
				targetSum += node.X + node.Y
			}
		}
		return targetSum
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 gave %f, Run 2 gave %f", result1, result2)
	}
}

func TestWanderSystem_PossessedBypass(t *testing.T) {
	world := ecs.NewWorld()
	grid := engine.NewMapGrid(10, 10)
	grid.Resources[5] = engine.ResourceDepot{FoodValue: 10} // Target

	queue := engine.NewPathRequestQueue(10, 1)
	sys := systems.NewWanderSystem(&world, grid, queue)

	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	possessedID := ecs.ComponentID[components.Possessed](&world)

	e := world.NewEntity(posID, idID, needsID, pathID, possessedID)
	pos := (*components.Position)(world.Get(e, posID))
	id := (*components.Identity)(world.Get(e, idID))
	needs := (*components.Needs)(world.Get(e, needsID))

	pos.X, pos.Y = 1.0, 1.0
	id.ID = 2
	needs.Food = 20.0 // Hungry

	sys.Update(&world)

	path := (*components.Path)(world.Get(e, pathID))
	if path.HasPath {
		t.Errorf("WanderSystem should not process paths for Possessed entities")
	}
}
