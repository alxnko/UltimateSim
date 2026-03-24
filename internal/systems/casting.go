package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 49 - The Witch Hunt Engine
// Phase 20.2: Abstract Physics (The "Magic" Expansion)
// CastingSystem scans for NPCs with the JobCaster role.
// It checks the underlying MapGrid for ManaData. If Mana > 50, it triggers an override
// (e.g. spikes Temperature to 255) and reduces local Mana by 50.

type CastingSystem struct {
	mapGrid *engine.MapGrid
	toTag   []ecs.Entity
}

func NewCastingSystem(world *ecs.World, mapGrid *engine.MapGrid) *CastingSystem {
	return &CastingSystem{
		mapGrid: mapGrid,
		toTag:   make([]ecs.Entity, 0, 10),
	}
}

func (s *CastingSystem) Update(world *ecs.World) {
	// Query for entities that have Position and JobComponent
	posID := ecs.ComponentID[components.Position](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	esotericID := ecs.ComponentID[components.EsotericMarker](world)

	filter := ecs.All(posID, jobID)
	query := world.Query(&filter)

	s.toTag = s.toTag[:0]

	for query.Next() {
		job := (*components.JobComponent)(query.Get(jobID))
		if job.JobID != components.JobCaster {
			continue
		}

		pos := (*components.Position)(query.Get(posID))

		// Convert position to map grid coordinates
		x := int(pos.X)
		y := int(pos.Y)

		// Check bounds to ensure DOD safety
		if x < 0 || x >= s.mapGrid.Width || y < 0 || y >= s.mapGrid.Height {
			continue
		}

		// Calculate 1D index
		idx := y*s.mapGrid.Width + x

		// Access the abstract magic array
		if s.mapGrid.Mana[idx].Value >= 50 {
			// Expend Mana
			s.mapGrid.Mana[idx].Value -= 50

			// Spike the local temperature to max (255) as a magical effect
			tile := s.mapGrid.Tiles[idx]

			// Increase temperature by 100, capped at 255
			temp := int(tile.Temperature) + 100
			if temp > 255 {
				temp = 255
			}
			tile.Temperature = uint8(temp)

			// Write back tile data
			s.mapGrid.Tiles[idx] = tile

			// Evolution: Phase 49 - The Witch Hunt Engine
			// NPCs successfully casting magic are tagged with an EsotericMarker
			if !world.Has(query.Entity(), esotericID) {
				s.toTag = append(s.toTag, query.Entity())
			}
		}
	}

	for _, entity := range s.toTag {
		if world.Alive(entity) && !world.Has(entity, esotericID) {
			world.Add(entity, esotericID)
		}
	}
}
