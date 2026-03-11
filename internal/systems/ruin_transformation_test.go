package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.2: Ruin Transformation Tests

// TestRuinTransformationSystem_E2E validates "Test First" logic:
// If PopulationComponent.Count hits 0, the entity must be turned into a Ruin
func TestRuinTransformationSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	ruinID := ecs.ComponentID[components.RuinComponent](&world)

	ruinSys := systems.NewRuinTransformationSystem(&world)

	// Entity 1: Has Population
	e1 := world.NewEntity(popID, needsID, idID)
	p1 := (*components.PopulationComponent)(world.Get(e1, popID))
	p1.Count = 10
	n1 := (*components.Needs)(world.Get(e1, needsID))
	n1.Food = 100
	id1 := (*components.Identity)(world.Get(e1, idID))
	id1.Name = "LivingCity"

	// Entity 2: Population hit 0
	e2 := world.NewEntity(popID, needsID, idID)
	p2 := (*components.PopulationComponent)(world.Get(e2, popID))
	p2.Count = 0
	n2 := (*components.Needs)(world.Get(e2, needsID))
	n2.Food = 50
	id2 := (*components.Identity)(world.Get(e2, idID))
	id2.Name = "DeadCity"

	ruinSys.Update(&world)

	// Verify e1
	if !world.Has(e1, popID) {
		t.Errorf("Expected Entity 1 to keep PopulationComponent")
	}
	if !world.Has(e1, needsID) {
		t.Errorf("Expected Entity 1 to keep NeedsComponent")
	}
	if world.Has(e1, ruinID) {
		t.Errorf("Expected Entity 1 not to have RuinComponent")
	}

	// Verify e2
	if world.Has(e2, popID) {
		t.Errorf("Expected Entity 2 to lose PopulationComponent")
	}
	if world.Has(e2, needsID) {
		t.Errorf("Expected Entity 2 to lose NeedsComponent")
	}
	if !world.Has(e2, ruinID) {
		t.Fatalf("Expected Entity 2 to gain RuinComponent")
	}

	ruin := (*components.RuinComponent)(world.Get(e2, ruinID))
	if ruin.FormerName != "DeadCity" {
		t.Errorf("Expected Ruin FormerName to be 'DeadCity', got '%s'", ruin.FormerName)
	}
	if ruin.Decay != 0 {
		t.Errorf("Expected Ruin Decay to be 0, got %d", ruin.Decay)
	}
}

// Deterministic Check: Runs simulation multiple times, expecting exact same state
func TestRuinTransformationSystem_Deterministic(t *testing.T) {
	runSim := func() int {
		world := ecs.NewWorld()
		popID := ecs.ComponentID[components.PopulationComponent](&world)
		ruinID := ecs.ComponentID[components.RuinComponent](&world)

		ruinSys := systems.NewRuinTransformationSystem(&world)

		// Spawn identical entities
		for i := 0; i < 1000; i++ {
			entity := world.NewEntity(popID)
			p := (*components.PopulationComponent)(world.Get(entity, popID))

			// deterministic state based on index
			p.Count = uint32(i % 5) // Some will have 0 population
		}

		// Run update
		ruinSys.Update(&world)

		// Calculate total ruins as fingerprint
		query := world.Query(ecs.All(ruinID))
		count := 0
		for query.Next() {
			count++
		}
		return count
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 ruins %d, Run 2 ruins %d", result1, result2)
	}
}
