package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 19.4: Advanced Biology (Vitals Integration)
// Tests the Butterfly Effect: Starvation -> Pain -> Unconsciousness -> Paralysis.
func TestVitalsSystem_Integration(t *testing.T) {
	world1 := setupVitalsTestWorld()
	world2 := setupVitalsTestWorld()

	mapGrid := engine.NewMapGrid(10, 10)
	calendar := engine.NewCalendar()

	metabolism1 := NewMetabolismSystem(&world1, calendar)
	movement1 := NewMovementSystem(&world1, mapGrid, calendar)

	metabolism2 := NewMetabolismSystem(&world2, calendar)
	movement2 := NewMovementSystem(&world2, mapGrid, calendar)

	// Simulate 350 ticks (enough to drop consciousness to 0)
	for i := 0; i < 350; i++ {
		metabolism1.Update(&world1)
		movement1.Update(&world1)

		metabolism2.Update(&world2)
		movement2.Update(&world2)
	}

	// Verify the final state is identical
	verifyVitalsDeterministic(t, &world1, &world2)
}

func setupVitalsTestWorld() ecs.World {
	world := ecs.NewWorld()

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	geneticsID := ecs.ComponentID[components.GenomeComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)

	// Create starving entity
	entity := world.NewEntity(posID, velID, needsID, geneticsID, vitalsID)

	pos := (*components.Position)(world.Get(entity, posID))
	vel := (*components.Velocity)(world.Get(entity, velID))
	needs := (*components.Needs)(world.Get(entity, needsID))
	vitals := (*components.VitalsComponent)(world.Get(entity, vitalsID))

	pos.X = 5.0
	pos.Y = 5.0

	// Set velocity so it WOULD move if conscious
	vel.X = 1.0
	vel.Y = 1.0

	needs.Food = 0.0 // Starving

	// Initial vitals
	vitals.Stamina = 100.0
	vitals.Blood = 100.0
	vitals.Pain = 0.0
	vitals.Consciousness = 100.0

	return world
}

func verifyVitalsDeterministic(t *testing.T, w1, w2 *ecs.World) {
	posID1 := ecs.ComponentID[components.Position](w1)
	velID1 := ecs.ComponentID[components.Velocity](w1)
	vitalsID1 := ecs.ComponentID[components.VitalsComponent](w1)
	needsID1 := ecs.ComponentID[components.Needs](w1)

	query1 := w1.Query(ecs.All(posID1, velID1, vitalsID1, needsID1))
	if !query1.Next() {
		t.Fatal("Expected entity to exist in world 1")
	}

	pos1 := (*components.Position)(query1.Get(posID1))
	vel1 := (*components.Velocity)(query1.Get(velID1))
	vitals1 := (*components.VitalsComponent)(query1.Get(vitalsID1))
	needs1 := (*components.Needs)(query1.Get(needsID1))

	posID2 := ecs.ComponentID[components.Position](w2)
	velID2 := ecs.ComponentID[components.Velocity](w2)
	vitalsID2 := ecs.ComponentID[components.VitalsComponent](w2)
	needsID2 := ecs.ComponentID[components.Needs](w2)

	query2 := w2.Query(ecs.All(posID2, velID2, vitalsID2, needsID2))
	if !query2.Next() {
		t.Fatal("Expected entity to exist in world 2")
	}

	pos2 := (*components.Position)(query2.Get(posID2))
	vel2 := (*components.Velocity)(query2.Get(velID2))
	vitals2 := (*components.VitalsComponent)(query2.Get(vitalsID2))
	needs2 := (*components.Needs)(query2.Get(needsID2))

	// Assertions for determinism
	if vel1.X != vel2.X || vel1.Y != vel2.Y {
		t.Fatalf("Determinism failure: Vel1(%f, %f) != Vel2(%f, %f)", vel1.X, vel1.Y, vel2.X, vel2.Y)
	}
	if needs1.Food != needs2.Food {
		t.Fatalf("Determinism failure: Needs1(%f) != Needs2(%f)", needs1.Food, needs2.Food)
	}
	if pos1.X != pos2.X || pos1.Y != pos2.Y {
		t.Fatalf("Determinism failure: Pos1(%f, %f) != Pos2(%f, %f)", pos1.X, pos1.Y, pos2.X, pos2.Y)
	}

	if vitals1.Pain != vitals2.Pain {
		t.Fatalf("Determinism failure: Pain1(%f) != Pain2(%f)", vitals1.Pain, vitals2.Pain)
	}

	// Logic Checks
	// 200 ticks of food == 0 -> Pain should increase by 0.5 per tick = 100 Pain
	if vitals1.Pain < 90.0 {
		t.Fatalf("Expected high pain due to starvation, got %f", vitals1.Pain)
	}

	if vitals1.Consciousness > 0.0 {
		t.Fatalf("Expected <= 0 consciousness, got %f", vitals1.Consciousness)
	}

	// 0 consciousness -> Movement system should override velocity to 0.0
	if vel1.X != 0.0 || vel1.Y != 0.0 {
		t.Fatalf("Expected velocity to be forced to 0 due to unconsciousness, got (%f, %f)", vel1.X, vel1.Y)
	}

	// Velocity is 0 -> Should barely move from starting pos (5.0, 5.0) before passing out.
	// We'll just verify the Butterfly effect happened successfully (it stopped moving).
	if needs1.Food > 0.0 {
		t.Fatalf("Expected 0 food, got %f", needs1.Food)
	}
}
