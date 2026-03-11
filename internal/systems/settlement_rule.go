package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.1: Settlement Conversion System
// Transforms migrating groups (FamilyCluster) into stationary settlements (Village)
// if their velocity is 0 for 1000 consecutive ticks on a resource-rich tile.

type SettlementRuleSystem struct {
	mapGrid  *engine.MapGrid
	toRemove []ecs.Entity // Collect entities for removal after the iteration loop
}

func NewSettlementRuleSystem(mapGrid *engine.MapGrid) *SettlementRuleSystem {
	return &SettlementRuleSystem{
		mapGrid:  mapGrid,
		toRemove: make([]ecs.Entity, 0, 100),
	}
}

func (s *SettlementRuleSystem) Update(world *ecs.World) {
	fcID := ecs.ComponentID[components.FamilyCluster](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// We only process entities that are part of a migrating family cluster
	filter := ecs.All(fcID, slID, posID, velID)
	query := world.Query(filter)

	s.toRemove = s.toRemove[:0] // Clear slice to reuse capacity

	for query.Next() {
		vel := (*components.Velocity)(query.Get(velID))
		sl := (*components.SettlementLogic)(query.Get(slID))

		if vel.X == 0 && vel.Y == 0 {
			sl.TicksAtZeroVelocity++
		} else {
			sl.TicksAtZeroVelocity = 0
		}

		if sl.TicksAtZeroVelocity >= 1000 {
			pos := (*components.Position)(query.Get(posID))

			// Evaluate if the current position is resource-rich
			x, y := int(pos.X), int(pos.Y)

			if x >= 0 && x < s.mapGrid.Width && y >= 0 && y < s.mapGrid.Height {
				depot := s.mapGrid.Resources[y*s.mapGrid.Width+x]
				if uint32(depot.WoodValue)+uint32(depot.FoodValue) > 50 {
					s.toRemove = append(s.toRemove, query.Entity())
				}
			}
		}
	}

	// Spawn new Village entities
	villageID := ecs.ComponentID[components.Village](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)

	// Optional inherited components (copy from parent)
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.Genetics](world)
	legID := ecs.ComponentID[components.Legacy](world)

	for _, e := range s.toRemove {
		pos := (*components.Position)(world.Get(e, posID))

		// Extract some data before despawning
		var inheritedID components.Identity
		var inheritedGen components.Genetics
		var inheritedLeg components.Legacy

		if world.Has(e, idID) {
			inheritedID = *(*components.Identity)(world.Get(e, idID))
		}
		if world.Has(e, genID) {
			inheritedGen = *(*components.Genetics)(world.Get(e, genID))
		}
		if world.Has(e, legID) {
			inheritedLeg = *(*components.Legacy)(world.Get(e, legID))
		}

		// Save old pos before removing
		oldPos := *pos

		// Despawn old entity
		world.RemoveEntity(e)

		// Spawn new Village
		newEntity := world.NewEntity(villageID, posID, storageID, popID, idID, genID, legID)

		newPos := (*components.Position)(world.Get(newEntity, posID))
		*newPos = oldPos // exact float vector

		newStorage := (*components.StorageComponent)(world.Get(newEntity, storageID))
		// Initialize some basic storage
		newStorage.Wood = 100
		newStorage.Food = 100
		newStorage.Stone = 0
		newStorage.Iron = 0

		newPop := (*components.PopulationComponent)(world.Get(newEntity, popID))
		newPop.Count = 10 // abstract headcount for new village

		// Re-attach inherited core components
		if world.Has(newEntity, idID) {
			*(*components.Identity)(world.Get(newEntity, idID)) = inheritedID
		}
		if world.Has(newEntity, genID) {
			*(*components.Genetics)(world.Get(newEntity, genID)) = inheritedGen
		}
		if world.Has(newEntity, legID) {
			*(*components.Legacy)(world.Get(newEntity, legID)) = inheritedLeg
		}
	}
}
