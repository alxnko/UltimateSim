package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 06.1: Societal Hierarchies

type CityBinderSystem struct {
	TicksElapsed uint32
}

// NewCityBinderSystem creates a new CityBinderSystem.
func NewCityBinderSystem() *CityBinderSystem {
	return &CityBinderSystem{
		TicksElapsed: 0,
	}
}

func (s *CityBinderSystem) Update(world *ecs.World) {
	s.TicksElapsed++
	if s.TicksElapsed%10000 != 0 {
		return
	}

	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	villageID := ecs.ComponentID[components.Village](world)
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)

	// Collect active villages
	type villageData struct {
		pos components.Position
		id  uint32
	}

	var villages []villageData
	villageQuery := world.Query(filter.All(posID, villageID, identID))
	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(posID))
		ident := (*components.Identity)(villageQuery.Get(identID))
		villages = append(villages, villageData{
			pos: *pos,
			id:  uint32(ident.ID),
		})
	}

	if len(villages) == 0 {
		return // No villages to bind to
	}

	// Update wandering NPCs
	clusterQuery := world.Query(filter.All(posID, affID, npcID))
	for clusterQuery.Next() {
		pos := (*components.Position)(clusterQuery.Get(posID))
		aff := (*components.Affiliation)(clusterQuery.Get(affID))

		// Find nearest village
		nearestID := villages[0].id
		minDistSq := float32(-1.0)

		for _, v := range villages {
			dx := v.pos.X - pos.X
			dy := v.pos.Y - pos.Y
			distSq := dx*dx + dy*dy

			if minDistSq < 0 || distSq < minDistSq {
				minDistSq = distSq
				nearestID = v.id
			}
		}

		aff.CityID = nearestID
	}
}
