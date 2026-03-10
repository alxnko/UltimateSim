package components

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

func TestComponentSanity(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[Position](&world)
	velID := ecs.ComponentID[Velocity](&world)

	// Spawn 10 entities
	for i := 0; i < 10; i++ {
		entity := world.NewEntity()
		world.Add(entity, posID)
		world.Add(entity, velID)

		pos := (*Position)(world.Get(entity, posID))
		vel := (*Velocity)(world.Get(entity, velID))

		pos.X = float32(i)
		pos.Y = float32(i * 2)
		vel.X = 1.0
		vel.Y = 0.5
	}

	// Query and verify
	q := world.Query(filter.All(posID, velID))
	count := 0
	for q.Next() {
		pos := (*Position)(q.Get(posID))
		vel := (*Velocity)(q.Get(velID))

		if pos.X != float32(count) || pos.Y != float32(count * 2) {
			t.Errorf("Position mismatch at entity %d", count)
		}

		if vel.X != 1.0 || vel.Y != 0.5 {
			t.Errorf("Velocity mismatch at entity %d", count)
		}

		count++
	}

	if count != 10 {
		t.Errorf("Expected 10 entities, found %d", count)
	}
}
