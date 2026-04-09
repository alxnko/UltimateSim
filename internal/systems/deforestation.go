package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 55 - The Ecological Collapse Engine (DeforestationSystem)
// Bridges Geography and Economy by having JobLumberjack NPCs physically extract WoodValue
// from their local MapGrid tile and add it to their employer's (or village's) StorageComponent.Wood.

type DeforestationSystem struct {
	mapGrid *engine.MapGrid

	npcID     ecs.ID
	posID     ecs.ID
	jobID     ecs.ID
	vitalsID  ecs.ID

	villageID ecs.ID
	storageID ecs.ID
	identID   ecs.ID
	busID     ecs.ID
	affilID   ecs.ID

	// Cache
	storages map[uint64]*components.StorageComponent
	initialized bool
}

func NewDeforestationSystem(mapGrid *engine.MapGrid) *DeforestationSystem {
	return &DeforestationSystem{
		mapGrid:  mapGrid,
		storages: make(map[uint64]*components.StorageComponent),
	}
}

func (s *DeforestationSystem) Update(world *ecs.World) {
	if !s.initialized {
		s.npcID = ecs.ComponentID[components.NPC](world)
		s.posID = ecs.ComponentID[components.Position](world)
		s.jobID = ecs.ComponentID[components.JobComponent](world)
		s.vitalsID = ecs.ComponentID[components.VitalsComponent](world)

		s.villageID = ecs.ComponentID[components.Village](world)
		s.storageID = ecs.ComponentID[components.StorageComponent](world)
		s.identID = ecs.ComponentID[components.Identity](world)
		s.busID = ecs.ComponentID[components.BusinessEntity](world)
		s.affilID = ecs.ComponentID[components.Affiliation](world)

		s.initialized = true
	}

	clear(s.storages)

	// Pre-cache employer/village storages
	filterVil := ecs.All(s.villageID, s.storageID)
	queryVil := world.Query(filterVil)
	for queryVil.Next() {
		entID := world.Get(queryVil.Entity(), s.identID)
		if entID != nil {
			id := (*components.Identity)(entID).ID
			storage := (*components.StorageComponent)(queryVil.Get(s.storageID))
			s.storages[id] = storage
		}
	}

	// Phase 15.1: Business entity storages
	filterBus := ecs.All(s.busID, s.identID, s.storageID)
	queryBus := world.Query(filterBus)
	for queryBus.Next() {
		id := (*components.Identity)(queryBus.Get(s.identID)).ID
		storage := (*components.StorageComponent)(queryBus.Get(s.storageID))
		s.storages[id] = storage
	}


	filter := ecs.All(s.npcID, s.posID, s.jobID, s.vitalsID)
	query := world.Query(filter)

	for query.Next() {
		job := (*components.JobComponent)(query.Get(s.jobID))

		if job.JobID != components.JobLumberjack {
			continue
		}

		vitals := (*components.VitalsComponent)(query.Get(s.vitalsID))
		if vitals.Consciousness <= 0 || vitals.Stamina < 10 {
			continue // Too tired to chop
		}

		pos := (*components.Position)(query.Get(s.posID))

		x, y := int(pos.X), int(pos.Y)
		if x < 0 || x >= s.mapGrid.Width || y < 0 || y >= s.mapGrid.Height {
			continue
		}

		idx := y*s.mapGrid.Width + x
		wood := s.mapGrid.Resources[idx].WoodValue

		if wood > 0 {
			// Extract wood
			s.mapGrid.Resources[idx].WoodValue--
			vitals.Stamina -= 5

			// Try to deposit to employer
			if storage, ok := s.storages[job.EmployerID]; ok {
				storage.Wood++
			} else {
				// Try village storage if no direct employer
				affil := (*components.Affiliation)(world.Get(query.Entity(), s.affilID))
				if affil != nil {
					if vStorage, ok := s.storages[uint64(affil.CityID)]; ok {
						vStorage.Wood++
					}
				}
			}
		}
	}
}
