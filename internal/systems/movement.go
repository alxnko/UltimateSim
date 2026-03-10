package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 01.3: ECS Core Setup - MovementSystem
// Implementing a deterministic, cache-friendly iteration system over Position and Velocity components.

// MovementSystem updates the Position of entities based on their Velocity.
type MovementSystem struct {
	filter ecs.Filter
}

// NewMovementSystem creates a new MovementSystem.
func NewMovementSystem(world *ecs.World) *MovementSystem {
	// Enforce strict 'arche-go' filter usage to query specific components and prevent 'Zombie Entity' processing.
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// ecs.All returns an ecs.Mask which implements ecs.Filter
	mask := ecs.All(posID, velID)

	return &MovementSystem{
		filter: &mask,
	}
}

// Update executes the system logic per tick.
func (s *MovementSystem) Update(world *ecs.World) {
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// Iterate over all entities matching the filter
	query := world.Query(s.filter)
	for query.Next() {
		// Access components via flat memory pointers (arche-go handles the layout)
		pos := (*components.Position)(query.Get(posID))
		vel := (*components.Velocity)(query.Get(velID))

		// Apply velocity to position
		pos.X += vel.X
		pos.Y += vel.Y
	}
}
