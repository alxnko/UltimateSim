package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.3: Infrastructure Wear System (Desire Paths) - Tests

// TestInfrastructureWearSystem_E2E validates that a moving entity increases foot traffic
// on the correct tile.
func TestInfrastructureWearSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	mapGrid := engine.NewMapGrid(10, 10)
	sys := systems.NewInfrastructureWearSystem(mapGrid)

	// Create a moving entity
	e1 := world.NewEntity(posID, velID)
	pos := (*components.Position)(world.Get(e1, posID))
	vel := (*components.Velocity)(world.Get(e1, velID))

	pos.X, pos.Y = 5.5, 5.5
	vel.X, vel.Y = 1.0, 0.0 // Moving

	// Create a stationary entity
	e2 := world.NewEntity(posID, velID)
	pos2 := (*components.Position)(world.Get(e2, posID))
	vel2 := (*components.Velocity)(world.Get(e2, velID))

	pos2.X, pos2.Y = 2.5, 2.5
	vel2.X, vel2.Y = 0.0, 0.0 // Stationary

	sys.Update(&world)

	index1 := int(5)*10 + int(5)
	if mapGrid.TileStates[index1].FootTraffic != 1 {
		t.Errorf("Expected FootTraffic at (5,5) to be 1, got %d", mapGrid.TileStates[index1].FootTraffic)
	}

	index2 := int(2)*10 + int(2)
	if mapGrid.TileStates[index2].FootTraffic != 0 {
		t.Errorf("Expected FootTraffic at (2,2) to be 0 for stationary entity, got %d", mapGrid.TileStates[index2].FootTraffic)
	}

	// Move the entity
	pos.X = 6.5
	sys.Update(&world)

	index3 := int(5)*10 + int(6)
	if mapGrid.TileStates[index3].FootTraffic != 1 {
		t.Errorf("Expected FootTraffic at (6,5) to be 1, got %d", mapGrid.TileStates[index3].FootTraffic)
	}
}

// TestInfrastructureWearSystem_Deterministic ensures multiple runs yield identical tile traffic states.
func TestInfrastructureWearSystem_Deterministic(t *testing.T) {
	runSim := func() uint32 {
		world := ecs.NewWorld()
		posID := ecs.ComponentID[components.Position](&world)
		velID := ecs.ComponentID[components.Velocity](&world)

		mapGrid := engine.NewMapGrid(100, 100)
		sys := systems.NewInfrastructureWearSystem(mapGrid)

		// Spawn 100 entities
		for i := 0; i < 100; i++ {
			e := world.NewEntity(posID, velID)
			pos := (*components.Position)(world.Get(e, posID))
			vel := (*components.Velocity)(world.Get(e, velID))

			// Deterministic start position and velocity based on i
			pos.X, pos.Y = float32(i%10)*10.0, float32(i/10)*10.0
			vel.X, vel.Y = 0.5, 0.5
		}

		// Run 10 ticks
		for i := 0; i < 10; i++ {
			// Update position for movement mapping (simulate movement)
			query := world.Query(ecs.All(posID, velID))
			for query.Next() {
				p := (*components.Position)(query.Get(posID))
				v := (*components.Velocity)(query.Get(velID))
				p.X += v.X
				p.Y += v.Y
			}
			sys.Update(&world)
		}

		var totalTraffic uint32 = 0
		for _, state := range mapGrid.TileStates {
			totalTraffic += state.FootTraffic
		}
		return totalTraffic
	}

	res1 := runSim()
	res2 := runSim()

	if res1 != res2 {
		t.Fatalf("Determinism check failed: Run 1 gave %d, Run 2 gave %d", res1, res2)
	}
}
