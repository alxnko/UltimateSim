package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: MetabolismSystem Tests

func TestMetabolismSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)
	geneticsID := ecs.ComponentID[components.Genetics](&world)

	metabolismSystem := systems.NewMetabolismSystem(&world, nil)

	// Entity 1: High Health (100)
	e1 := world.NewEntity(needsID, geneticsID)
	n1 := (*components.Needs)(world.Get(e1, needsID))
	g1 := (*components.Genetics)(world.Get(e1, geneticsID))
	n1.Food = 100.0
	g1.Health = 100 // Modifier = 1.0 -> Food loss = 0.05

	// Entity 2: Low Health (0)
	e2 := world.NewEntity(needsID, geneticsID)
	n2 := (*components.Needs)(world.Get(e2, needsID))
	g2 := (*components.Genetics)(world.Get(e2, geneticsID))
	n2.Food = 100.0
	g2.Health = 0 // Modifier = 2.0 -> Food loss = 0.10

	metabolismSystem.Update(&world)

	// Verify
	n1Updated := (*components.Needs)(world.Get(e1, needsID))
	n2Updated := (*components.Needs)(world.Get(e2, needsID))

	// floating point math check with a small epsilon
	if n1Updated.Food > 99.96 || n1Updated.Food < 99.94 {
		t.Fatalf("Expected n1.Food around 99.95, got %f", n1Updated.Food)
	}

	if n2Updated.Food > 99.91 || n2Updated.Food < 99.89 {
		t.Fatalf("Expected n2.Food around 99.90, got %f", n2Updated.Food)
	}
}
