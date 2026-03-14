package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.1 & 14: Settlement Conversion System
// Spawns a new stationary settlement (Village) where an NPC group stopped
// if their velocity is 0 for 1000 consecutive ticks on a resource-rich tile.
// The NPC updates its CityID to live in the new Village instead of despawning.

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
	npcID := ecs.ComponentID[components.NPC](world)
	slID := ecs.ComponentID[components.SettlementLogic](world)
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)

	// We only process entities that are part of a migrating family cluster
	filter := ecs.All(npcID, slID, posID, velID)
	query := world.Query(filter)

	s.toRemove = s.toRemove[:0] // Clear slice to reuse capacity, now used to hold NPCs that settle

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
	marketID := ecs.ComponentID[components.MarketComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	// Optional inherited components (copy from parent)
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	legID := ecs.ComponentID[components.Legacy](world)

	for _, e := range s.toRemove {
		pos := (*components.Position)(world.Get(e, posID))

		// Extract some data
		var inheritedID components.Identity
		var inheritedGen components.GenomeComponent
		var inheritedLeg components.Legacy

		if world.Has(e, idID) {
			inheritedID = *(*components.Identity)(world.Get(e, idID))
		}
		if world.Has(e, genID) {
			inheritedGen = *(*components.GenomeComponent)(world.Get(e, genID))
		}
		if world.Has(e, legID) {
			inheritedLeg = *(*components.Legacy)(world.Get(e, legID))
		}

		// Save old pos
		oldPos := *pos

		// Reset NPC's SettlementLogic so they don't keep spawning villages
		if world.Has(e, slID) {
			sl := (*components.SettlementLogic)(world.Get(e, slID))
			sl.TicksAtZeroVelocity = 0
		}

		// Spawn new Village
		newEntity := world.NewEntity(villageID, posID, storageID, popID, marketID, idID, genID, legID)

		newPos := (*components.Position)(world.Get(newEntity, posID))
		*newPos = oldPos // exact float vector

		newStorage := (*components.StorageComponent)(world.Get(newEntity, storageID))
		// Initialize some basic storage
		newStorage.Wood = 100
		newStorage.Food = 100
		newStorage.Stone = 0
		newStorage.Iron = 0

		newPop := (*components.PopulationComponent)(world.Get(newEntity, popID))
		newPop.Count = 1 // abstract headcount for new village

		newMarket := (*components.MarketComponent)(world.Get(newEntity, marketID))
		newMarket.FoodPrice = 1.0
		newMarket.WoodPrice = 1.0
		newMarket.StonePrice = 1.0
		newMarket.IronPrice = 1.0

		// Re-attach inherited core components
		if world.Has(newEntity, idID) {
			newId := (*components.Identity)(world.Get(newEntity, idID))
			*newId = inheritedID
			// Set the NPC's Affiliation.CityID to the new Village Identity ID
			if world.Has(e, affID) {
				aff := (*components.Affiliation)(world.Get(e, affID))
				aff.CityID = uint32(newId.ID)
			}
		}
		if world.Has(newEntity, genID) {
			*(*components.GenomeComponent)(world.Get(newEntity, genID)) = inheritedGen
		}
		if world.Has(newEntity, legID) {
			*(*components.Legacy)(world.Get(newEntity, legID)) = inheritedLeg
		}
	}
}
