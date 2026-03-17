package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 38.1 - Ecological Pressure (The Exposure Engine)
// ExposureSystem connects Phase 2 (Geography) to Phase 19.4 (Biology) and Phase 13 (Economy).
// It evaluates the MapGrid temperature at the NPC's location. If the temperature is extreme
// (> 200 for heat, < 50 for cold), it inflicts Pain on the NPC unless they have sufficient
// Needs.Safety (abstracting shelter/clothing).

type ExposureSystem struct {
	mapGrid *engine.MapGrid
	filter  ecs.Filter

	// Component IDs
	posID    ecs.ID
	vitalsID ecs.ID
	needsID  ecs.ID
}

func NewExposureSystem(world *ecs.World, mapGrid *engine.MapGrid) *ExposureSystem {
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)

	mask := ecs.All(posID, vitalsID, needsID)

	return &ExposureSystem{
		mapGrid:  mapGrid,
		filter:   &mask,
		posID:    posID,
		vitalsID: vitalsID,
		needsID:  needsID,
	}
}

func (s *ExposureSystem) Update(world *ecs.World) {
	// Query all entities with Position, Vitals, and Needs
	query := world.Query(s.filter)

	for query.Next() {
		pos := (*components.Position)(query.Get(s.posID))
		vitals := (*components.VitalsComponent)(query.Get(s.vitalsID))
		needs := (*components.Needs)(query.Get(s.needsID))

		// Convert position to 1D grid index
		x := int(pos.X)
		y := int(pos.Y)

		// Bounds check
		if x < 0 || x >= s.mapGrid.Width || y < 0 || y >= s.mapGrid.Height {
			continue
		}

		idx := y*s.mapGrid.Width + x
		tileTemp := s.mapGrid.Tiles[idx].Temperature

		// Check for extreme temperatures
		if tileTemp > 200 || tileTemp < 50 {
			// Check if NPC is protected by Safety (shelter/clothing)
			if needs.Safety < 50.0 {
				vitals.Pain += 1.0
			}
		}
	}
}
