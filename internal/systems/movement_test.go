package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 01.3: ECS Core Setup - MovementSystem Tests
// Must use standard Go 'testing' package for E2E tests, verifying determinism.

// TestMovementSystem_E2E validates "Test First" logic:
// If an entity is spawned with Position{0,0} and Velocity{1,1}, after one tick its Position is {1,1}.
func TestMovementSystem_E2E(t *testing.T) {
	// 1. Initialize world and components
	world := ecs.NewWorld()
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	// 2. Add System
	movementSystem := systems.NewMovementSystem(&world)

	// 3. Spawn Test Entity
	entity := world.NewEntity(posID, velID)

	pos := (*components.Position)(world.Get(entity, posID))
	vel := (*components.Velocity)(world.Get(entity, velID))

	// Initial State
	pos.X, pos.Y = 0, 0
	vel.X, vel.Y = 1, 1

	// 4. Tick the System once
	movementSystem.Update(&world)

	// 5. Verification
	newPos := (*components.Position)(world.Get(entity, posID))
	if newPos.X != 1 || newPos.Y != 1 {
		t.Fatalf("Expected Position{1, 1}, got Position{%f, %f}", newPos.X, newPos.Y)
	}
}

// Deterministic Check: Runs simulation multiple times, expecting the exact same final state.
func TestMovementSystem_Deterministic(t *testing.T) {
	runSim := func() float32 {
		world := ecs.NewWorld()
		posID := ecs.ComponentID[components.Position](&world)
		velID := ecs.ComponentID[components.Velocity](&world)

		movementSystem := systems.NewMovementSystem(&world)

		// Spawn identical entities
		for i := 0; i < 1000; i++ {
			entity := world.NewEntity(posID, velID)
			pos := (*components.Position)(world.Get(entity, posID))
			vel := (*components.Velocity)(world.Get(entity, velID))

			// Deterministic start state based on 'i'
			pos.X, pos.Y = float32(i), float32(i)
			vel.X, vel.Y = 0.5, 0.5
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
