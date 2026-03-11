package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.3: Infrastructure Wear System (Desire Paths)
// Evaluates moving entities. Every time an entity transitions Position vectors
// across a specific TileData grid index, increment TileStateComponent.FootTraffic.

type InfrastructureWearSystem struct {
	mapGrid *engine.MapGrid
}

// NewInfrastructureWearSystem creates a new InfrastructureWearSystem.
func NewInfrastructureWearSystem(mapGrid *engine.MapGrid) *InfrastructureWearSystem {
	return &InfrastructureWearSystem{
		mapGrid: mapGrid,
	}
}

// Update executes the system logic per tick.
func (s *InfrastructureWearSystem) Update(world *ecs.World) {
	if s.mapGrid == nil {
		return
	}

	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// We query for all entities with both Position and Velocity
	filter := ecs.All(posID, velID)
	query := world.Query(filter)

	width := s.mapGrid.Width
	height := s.mapGrid.Height

	for query.Next() {
		pos := (*components.Position)(query.Get(posID))
		vel := (*components.Velocity)(query.Get(velID))

		// If the entity is actually moving
		if vel.X != 0 || vel.Y != 0 {
			// Get integer map coordinates
			tileX := int(pos.X)
			tileY := int(pos.Y)

			// Bounds check
			if tileX >= 0 && tileX < width && tileY >= 0 && tileY < height {
				index := tileY*width + tileX
				// Increment foot traffic.
				// In a real scenario, continual increments logically reduce movement cost constants on that tile.
				s.mapGrid.TileStates[index].FootTraffic++
			}
		}
	}
}
