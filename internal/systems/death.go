package systems

import (
	"log"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: DeathSystem
// Scans for any Entity where Needs.Food <= 0. If found, trigger the Despawn pipeline.

type DeathSystem struct {
	filter   ecs.Filter
	toRemove []ecs.Entity
}

// NewDeathSystem creates a new DeathSystem.
func NewDeathSystem(world *ecs.World) *DeathSystem {
	// Query entities that have Needs
	needsID := ecs.ComponentID[components.Needs](world)
	mask := ecs.All(needsID)

	return &DeathSystem{
		filter:   &mask,
		toRemove: make([]ecs.Entity, 0, 100),
	}
}

// Update executes the system logic per tick.
func (s *DeathSystem) Update(world *ecs.World) {
	needsID := ecs.ComponentID[components.Needs](world)

	// Collect entities to remove to avoid modifying the world while iterating
	// Reset the slice length to 0, retaining capacity to avoid GC pressure
	s.toRemove = s.toRemove[:0]

	query := world.Query(s.filter)
	for query.Next() {
		needs := (*components.Needs)(query.Get(needsID))

		if needs.Food <= 0 {
			s.toRemove = append(s.toRemove, query.Entity())
			// log root causes to standard output
			// To add proper Name logging, we'd need IdentityComponent, but we only strictly query Needs
			// ecs.Entity formats safely to string via %v
			log.Printf("Entity %v despawned due to starvation (Food <= 0)", query.Entity())
		}
	}

	// Remove dead entities
	for _, entity := range s.toRemove {
		world.RemoveEntity(entity)
	}
}
