package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/network"
	"github.com/mlange-42/arche/ecs"
)

// Phase 12.3: Robust Client Prediction & Smoothing
// ClientPredictionSystem takes network.PositionDelta updates from the server
// and smoothly interpolates local entity coordinates towards authoritative ones,
// effectively synchronizing unpredictable inputs (like player overrides)
// without explicitly processing deterministic distant AI movement.

type ClientPredictionSystem struct {
	filter        ecs.Filter
	incomingDeltas []network.PositionDelta
	deltaMap      map[uint64]network.PositionDelta
}

func NewClientPredictionSystem(world *ecs.World) *ClientPredictionSystem {
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)

	mask := ecs.All(posID, idID)

	return &ClientPredictionSystem{
		filter:   &mask,
		deltaMap: make(map[uint64]network.PositionDelta),
	}
}

// QueueDeltas loads incoming network updates into the system for the next tick.
func (s *ClientPredictionSystem) QueueDeltas(deltas []network.PositionDelta) {
	s.incomingDeltas = deltas
}

// Update evaluates queued deltas and smoothly interpolates local positions towards them.
func (s *ClientPredictionSystem) Update(world *ecs.World) {
	if len(s.incomingDeltas) == 0 {
		return
	}

	// 1. Pre-calculate mapping outside the ECS loop to preserve O(1) matching
	// and avoid polluting the sequential arche-go iterator.
	// We clear the map rather than re-allocating it to avoid GC spikes.
	for k := range s.deltaMap {
		delete(s.deltaMap, k)
	}

	for _, delta := range s.incomingDeltas {
		s.deltaMap[delta.EntityID] = delta
	}

	// 2. Iterate entities to apply smoothing
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)

	query := world.Query(s.filter)
	for query.Next() {
		id := (*components.Identity)(query.Get(idID))

		// O(1) check if this specific entity has an incoming override
		if delta, exists := s.deltaMap[id.ID]; exists {
			pos := (*components.Position)(query.Get(posID))

			// Simple interpolation to smoothly shift local state to authoritative server state.
			// Using float32 math directly to adhere to Phase 1 DOD constraints.
			pos.X += (delta.X - pos.X) * 0.1
			pos.Y += (delta.Y - pos.Y) * 0.1
		}
	}

	// 3. Clear slice for the next frame
	s.incomingDeltas = s.incomingDeltas[:0]
}
