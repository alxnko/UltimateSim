package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 19.3: Biological Entropy (Aging)
// Test the butterfly effect: verify aging triggers Needs.Food depletion
func TestAgingSystem_ButterflyEffect(t *testing.T) {
	engine.InitializeRNG([32]byte{})
	world := ecs.NewWorld()

	// Component IDs
	idID := ecs.ComponentID[components.Identity](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	// We create an old NPC who is just about to start heavily rolling for death
	entity := world.NewEntity(idID, genID, needsID)

	id := (*components.Identity)(world.Get(entity, idID))
	id.Age = 79 // Next year they turn 80

	gen := (*components.GenomeComponent)(world.Get(entity, genID))
	gen.Health = 50 // Should degrade to 49

	needs := (*components.Needs)(world.Get(entity, needsID))
	needs.Food = 100.0

	// Create system
	sys := NewAgingSystem(&world, nil)

	// Run system for one "year" (360 ticks)
	for i := 0; i < 360; i++ {
		sys.Update(&world)
	}

	// Verify Age incremented
	if id.Age != 80 {
		t.Fatalf("Expected Age to be 80, got %v", id.Age)
	}

	// Verify Health decayed
	if gen.Health != 49 {
		t.Fatalf("Expected Health to decay to 49, got %v", gen.Health)
	}

	// To prove the butterfly effect, we need to run it enough years to trigger sudden death
	// Sudden death probability at age 81 is 6%, increases every year. Let's run for 50 more years.
	// We check if food is eventually set to 0.

	died := false
	for years := 0; years < 50; years++ {
		for ticks := 0; ticks < 360; ticks++ {
			sys.Update(&world)
		}

		if needs.Food == 0 {
			died = true
			break
		}
	}

	if !died {
		t.Fatalf("Expected NPC to eventually trigger sudden death (Needs.Food == 0) after aging past 80, but they survived indefinitely")
	}

	// At this point, running DeathSystem would despawn the entity, proving Integration
	hooks := engine.NewSparseHookGraph()
	deathSys := NewDeathSystem(&world, hooks)
	deathSys.Update(&world)

	if world.Alive(entity) {
		t.Fatalf("Expected DeathSystem to despawn the old entity, but it remained alive")
	}
}
