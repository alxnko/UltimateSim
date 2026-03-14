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

// Phase 09.5: Item Inheritance Tests
func TestDeathSystem_ItemInheritance(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	posID := ecs.ComponentID[components.Position](&world)

	deathSystem := systems.NewDeathSystem(&world)

	// Entity with High Prestige (should spawn item)
	e1 := world.NewEntity(needsID, legacyID, posID)
	n1 := (*components.Needs)(world.Get(e1, needsID))
	n1.Food = 0.0 // Starving
	l1 := (*components.Legacy)(world.Get(e1, legacyID))
	l1.Prestige = components.ExtremePrestigeThreshold + 50
	p1 := (*components.Position)(world.Get(e1, posID))
	p1.X = 10.0
	p1.Y = 20.0

	// Entity with Low Prestige (should not spawn item)
	e2 := world.NewEntity(needsID, legacyID, posID)
	n2 := (*components.Needs)(world.Get(e2, needsID))
	n2.Food = 0.0 // Starving
	l2 := (*components.Legacy)(world.Get(e2, legacyID))
	l2.Prestige = 10 // Low prestige
	p2 := (*components.Position)(world.Get(e2, posID))
	p2.X = 5.0
	p2.Y = 5.0

	// Pre-update item count
	itemID := ecs.ComponentID[components.ItemEntity](&world)
	legendID := ecs.ComponentID[components.LegendComponent](&world)

	queryBefore := world.Query(ecs.All(itemID))
	countBefore := 0
	for queryBefore.Next() { countBefore++ }
	// In arche-go, queries fully iterated via for q.Next() automatically close and unlock the world.
	// Calling q.Close() causes an unbalanced unlock panic.

	if countBefore != 0 {
		t.Errorf("Expected 0 items before update, got %d", countBefore)
	}

	deathSystem.Update(&world)

	// Post-update verification
	if world.Alive(e1) || world.Alive(e2) {
		t.Errorf("Expected both starving entities to despawn")
	}

	queryAfter := world.Query(ecs.All(itemID, legendID, posID))
	countAfter := 0
	for queryAfter.Next() {
		countAfter++

		pos := (*components.Position)(queryAfter.Get(posID))
		legend := (*components.LegendComponent)(queryAfter.Get(legendID))

		if pos.X != 10.0 || pos.Y != 20.0 {
			t.Errorf("Item spawned at incorrect position: %v, %v", pos.X, pos.Y)
		}

		if legend.Prestige != components.ExtremePrestigeThreshold + 50 {
			t.Errorf("Item inherited incorrect prestige: %v", legend.Prestige)
		}
	}

	if countAfter != 1 {
		t.Errorf("Expected exactly 1 item spawned, got %d", countAfter)
	}
}

// Deterministic Check: Runs simulation multiple times, expecting exact same state
func TestMetabolismAndDeathSystem_Deterministic(t *testing.T) {
	runSim := func() int {
		world := ecs.NewWorld()
		needsID := ecs.ComponentID[components.Needs](&world)
		geneticsID := ecs.ComponentID[components.GenomeComponent](&world)

		metabolismSys := systems.NewMetabolismSystem(&world, nil)
		deathSys := systems.NewDeathSystem(&world)

		// Spawn identical entities
		for i := 0; i < 1000; i++ {
			entity := world.NewEntity(needsID, geneticsID)
			n := (*components.Needs)(world.Get(entity, needsID))
			g := (*components.GenomeComponent)(world.Get(entity, geneticsID))

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
