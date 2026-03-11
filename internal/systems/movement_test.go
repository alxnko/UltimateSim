package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 01.3: ECS Core Setup - MovementSystem Tests
// Phase 04.4: Resolving Kinematics - Tests
// Must use standard Go 'testing' package for E2E tests, verifying determinism.

// TestMovementSystem_E2E validates "Test First" logic:
// If an entity has an active Path with a node at {10, 10}, it must move towards it, update its Position, and pop the node once reached without exceeding bounds.
func TestMovementSystem_E2E(t *testing.T) {
	// 1. Initialize world and components
	world := ecs.NewWorld()
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	// Create dummy map for bounds
	mapGrid := engine.NewMapGrid(100, 100)

	// 2. Add System
	movementSystem := systems.NewMovementSystem(&world, mapGrid)

	// 3. Spawn Test Entity
	entity := world.NewEntity(posID, velID, pathID)

	pos := (*components.Position)(world.Get(entity, posID))
	vel := (*components.Velocity)(world.Get(entity, velID))
	path := (*components.Path)(world.Get(entity, pathID))

	// Initial State
	pos.X, pos.Y = 0, 0
	vel.X, vel.Y = 0, 0

	// Set Path
	path.Nodes = []components.Position{
		{X: 10, Y: 10},
	}
	path.HasPath = true

	// 4. Tick the System once (speed is 1.0, distance to 10,10 is ~14.14)
	movementSystem.Update(&world)

	// 5. Verification after 1 tick
	newPos := (*components.Position)(world.Get(entity, posID))
	// Velocity should point to 10, 10
	if newPos.X <= 0 || newPos.Y <= 0 {
		t.Fatalf("Expected movement towards {10, 10}, got Position{%f, %f}", newPos.X, newPos.Y)
	}

	// 6. Tick until reached
	for i := 0; i < 20; i++ {
		movementSystem.Update(&world)
	}

	// Should have reached 10, 10, popped node, and stopped
	finalPath := (*components.Path)(world.Get(entity, pathID))
	if finalPath.HasPath {
		t.Fatalf("Expected path to be finished, HasPath is true. Nodes remaining: %d", len(finalPath.Nodes))
	}
	if len(finalPath.Nodes) != 0 {
		t.Fatalf("Expected 0 nodes, got %d", len(finalPath.Nodes))
	}

	finalPos := (*components.Position)(world.Get(entity, posID))
	// Check if close to 10, 10
	if finalPos.X < 9.9 || finalPos.X > 10.1 || finalPos.Y < 9.9 || finalPos.Y > 10.1 {
		t.Fatalf("Expected Position near {10, 10}, got {%f, %f}", finalPos.X, finalPos.Y)
	}

	finalVel := (*components.Velocity)(world.Get(entity, velID))
	if finalVel.X != 0 || finalVel.Y != 0 {
		t.Fatalf("Expected Velocity{0, 0} after stopping, got {%f, %f}", finalVel.X, finalVel.Y)
	}
}

// Deterministic Check: Runs simulation multiple times, expecting the exact same final state.
func TestMovementSystem_Deterministic(t *testing.T) {
	runSim := func() float32 {
		world := ecs.NewWorld()
		posID := ecs.ComponentID[components.Position](&world)
		velID := ecs.ComponentID[components.Velocity](&world)
		pathID := ecs.ComponentID[components.Path](&world)

		mapGrid := engine.NewMapGrid(100, 100)
		movementSystem := systems.NewMovementSystem(&world, mapGrid)

		// Spawn identical entities
		for i := 0; i < 1000; i++ {
			entity := world.NewEntity(posID, velID, pathID)
			pos := (*components.Position)(world.Get(entity, posID))
			vel := (*components.Velocity)(world.Get(entity, velID))
			path := (*components.Path)(world.Get(entity, pathID))

			// Deterministic start state based on 'i'
			pos.X, pos.Y = float32(i), float32(i)
			vel.X, vel.Y = 0.5, 0.5

			// Some active paths
			if i%2 == 0 {
				path.HasPath = true
				path.Nodes = []components.Position{
					{X: float32(i) + 10, Y: float32(i) + 10},
				}
			}
		}

		// Run 10 ticks
		for i := 0; i < 10; i++ {
			movementSystem.Update(&world)
		}

		// Calculate total X sum as a fingerprint
		query := world.Query(ecs.All(posID))
		var totalX float32 = 0
		for query.Next() {
			p := (*components.Position)(query.Get(posID))
			totalX += p.X
		}
		return totalX
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 gave %f, Run 2 gave %f", result1, result2)
	}
}

// TestMovementSystem_NonPathingEntity verifies that an entity without a Path component
// still moves according to its Velocity component.
func TestMovementSystem_NonPathingEntity(t *testing.T) {
	world := ecs.NewWorld()
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	mapGrid := engine.NewMapGrid(100, 100)
	movementSystem := systems.NewMovementSystem(&world, mapGrid)

	// Spawn Test Entity WITHOUT Path component
	entity := world.NewEntity(posID, velID)

	pos := (*components.Position)(world.Get(entity, posID))
	vel := (*components.Velocity)(world.Get(entity, velID))

	// Initial State
	pos.X, pos.Y = 50, 50
	vel.X, vel.Y = 2, -2

	// Tick the System once
	movementSystem.Update(&world)

	// Verification
	newPos := (*components.Position)(world.Get(entity, posID))
	if newPos.X != 52 || newPos.Y != 48 {
		t.Fatalf("Expected Position{52, 48}, got Position{%f, %f}", newPos.X, newPos.Y)
	}

	// Tick again to verify continuous movement
	movementSystem.Update(&world)

	newPos2 := (*components.Position)(world.Get(entity, posID))
	if newPos2.X != 54 || newPos2.Y != 46 {
		t.Fatalf("Expected Position{54, 46}, got Position{%f, %f}", newPos2.X, newPos2.Y)
	}
}
