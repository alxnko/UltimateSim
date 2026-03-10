package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: DeathSystem Tests

// TestDeathSystem_E2E validates "Test First" logic:
// If NPC Food reaches 0, the entity must be despawned
func TestDeathSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)

	deathSystem := systems.NewDeathSystem(&world)

	// Entity 1: Has Food
	e1 := world.NewEntity(needsID)
	n1 := (*components.Needs)(world.Get(e1, needsID))
	n1.Food = 10.0

	// Entity 2: Starving
	e2 := world.NewEntity(needsID)
	n2 := (*components.Needs)(world.Get(e2, needsID))
	n2.Food = 0.0

	// Entity 3: Already dead mathematically (negative)
	e3 := world.NewEntity(needsID)
	n3 := (*components.Needs)(world.Get(e3, needsID))
	n3.Food = -5.0

	deathSystem.Update(&world)

	// Verify
	if !world.Alive(e1) {
		t.Errorf("Expected Entity 1 (Food > 0) to be alive")
	}

	if world.Alive(e2) {
		t.Errorf("Expected Entity 2 (Food == 0) to be dead")
	}

	if world.Alive(e3) {
		t.Errorf("Expected Entity 3 (Food < 0) to be dead")
	}
}

// Deterministic Check: Runs simulation multiple times, expecting exact same state
func TestMetabolismAndDeathSystem_Deterministic(t *testing.T) {
	runSim := func() int {
		world := ecs.NewWorld()
		needsID := ecs.ComponentID[components.Needs](&world)
		geneticsID := ecs.ComponentID[components.Genetics](&world)

		metabolismSys := systems.NewMetabolismSystem(&world)
		deathSys := systems.NewDeathSystem(&world)

		// Spawn identical entities
		for i := 0; i < 1000; i++ {
			entity := world.NewEntity(needsID, geneticsID)
			n := (*components.Needs)(world.Get(entity, needsID))
			g := (*components.Genetics)(world.Get(entity, geneticsID))

			// deterministic state based on index
			n.Food = float32(i % 10) // Some will start with low food
			g.Health = uint8(i % 100)
		}

		// Run 50 ticks
		for i := 0; i < 50; i++ {
			metabolismSys.Update(&world)
			deathSys.Update(&world)
		}

		// Calculate total alive entities as fingerprint
		query := world.Query(ecs.All(needsID))
		count := 0
		for query.Next() {
			count++
		}
		return count
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 alive %d, Run 2 alive %d", result1, result2)
	}
}
