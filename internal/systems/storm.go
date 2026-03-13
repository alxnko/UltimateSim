package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.3: Maritime Attrition & Piracy
// StormSystem utilizes deterministic weather logic to spawn dynamic hurricane vectors over water arrays,
// executing massive DecaySystem damage to traversing ShipComponent hulls.

type StormSystem struct {
	mapGrid     *engine.MapGrid
	stormChance float32
	toRemove    []ecs.Entity // Pre-allocated slice for DOD entity removal
}

func NewStormSystem(grid *engine.MapGrid) *StormSystem {
	return &StormSystem{
		mapGrid:     grid,
		stormChance: 0.05, // 5% chance of storm damage per tick on ocean
		toRemove:    make([]ecs.Entity, 0, 100),
	}
}

func (s *StormSystem) Update(world *ecs.World) {
	if s.mapGrid == nil {
		return
	}

	shipID := ecs.ComponentID[components.ShipComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	filter := ecs.All(shipID, posID)
	query := world.Query(filter)

	s.toRemove = s.toRemove[:0]

	for query.Next() {
		pos := (*components.Position)(query.Get(posID))
		ship := (*components.ShipComponent)(query.Get(shipID))

		// Check if the ship is actually on an Ocean tile
		idx := int(pos.Y)*s.mapGrid.Width + int(pos.X)
		if idx >= 0 && idx < len(s.mapGrid.Tiles) && s.mapGrid.Tiles[idx].BiomeID == engine.BiomeOcean {
			// Deterministic RNG roll for storm damage
			if engine.GetRandomFloat32() < s.stormChance {
				// Base hull logic, default assumption is initialized high, decrementing by chunk
				if ship.Hull <= 10 {
					ship.Hull = 0
					s.toRemove = append(s.toRemove, query.Entity())
				} else {
					ship.Hull -= 10
				}
			}
		}
	}

	// Remove destroyed ships outside the iteration query loop
	for _, entity := range s.toRemove {
		world.RemoveEntity(entity)
	}
}
