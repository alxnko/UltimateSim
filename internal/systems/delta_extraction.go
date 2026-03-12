package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/network"
	"github.com/mlange-42/arche/ecs"
)

// Phase 12.2: Delta Extraction Queries
// Queries entities that are actively moving and constructs a payload of tightly packed struct arrays
// to synchronize over standard network protocols.

// DeltaExtractionSystem extracts dynamic movement deltas for active entities.
type DeltaExtractionSystem struct {
	filter        ecs.Filter
	currentDeltas []network.PositionDelta
}

// NewDeltaExtractionSystem initializes the extraction system.
func NewDeltaExtractionSystem(world *ecs.World) *DeltaExtractionSystem {
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	idID := ecs.ComponentID[components.Identity](world)

	mask := ecs.All(posID, velID, idID)

	return &DeltaExtractionSystem{
		filter:        &mask,
		currentDeltas: make([]network.PositionDelta, 0, 1024), // Pre-allocate initial capacity
	}
}

// Update extracts the positional updates and clears previous ticks' slice.
func (s *DeltaExtractionSystem) Update(world *ecs.World) {
	// Re-slice to length 0 while maintaining capacity, eliminating slice allocation garbage collection limits per frame.
	s.currentDeltas = s.currentDeltas[:0]

	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	idID := ecs.ComponentID[components.Identity](world)

	query := world.Query(s.filter)
	for query.Next() {
		vel := (*components.Velocity)(query.Get(velID))

		// Only extract network deltas if the entity is actually actively shifting positions,
		// radically reducing network throughput requirements.
		if vel.X != 0 || vel.Y != 0 {
			pos := (*components.Position)(query.Get(posID))
			id := (*components.Identity)(query.Get(idID))

			s.currentDeltas = append(s.currentDeltas, network.PositionDelta{
				EntityID: id.ID,
				X:        pos.X,
				Y:        pos.Y,
			})
		}
	}
}

// GetCurrentDeltas returns the slice of accumulated active deltas for the tick.
func (s *DeltaExtractionSystem) GetCurrentDeltas() []network.PositionDelta {
	return s.currentDeltas
}
