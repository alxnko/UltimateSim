package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 55 - The Ecological Collapse Engine
// DeforestationSystem links Geography (MapGrid) directly to Economy (StorageComponent).
// NPCs with JobLumberjack physically extract WoodValue from their local tile
// and add it to their employer's (or their own village's) StorageComponent.Wood.
// Depleting the WoodValue to 0 causes long-term ecological collapse and winter vulnerability.

type DeforestationSystem struct {
	mapGrid *engine.MapGrid
	tickCounter uint64

	// Cached Component IDs
	npcID      ecs.ID
	posID      ecs.ID
	jobID      ecs.ID
	storageID  ecs.ID
	idID       ecs.ID
	businessID ecs.ID
}

func NewDeforestationSystem(world *ecs.World, mapGrid *engine.MapGrid) *DeforestationSystem {
	return &DeforestationSystem{
		mapGrid:    mapGrid,
		npcID:      ecs.ComponentID[components.NPC](world),
		posID:      ecs.ComponentID[components.Position](world),
		jobID:      ecs.ComponentID[components.JobComponent](world),
		storageID:  ecs.ComponentID[components.StorageComponent](world),
		idID:       ecs.ComponentID[components.Identity](world),
		businessID: ecs.ComponentID[components.BusinessComponent](world),
	}
}

func (s *DeforestationSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Only process harvesting periodically (every 50 ticks) to simulate labor time
	if s.tickCounter%50 != 0 {
		return
	}

	if s.mapGrid == nil {
		return
	}

	// 1. Build a DOD map of active Employers (Businesses or Villages) that have a StorageComponent
	employerStorage := make(map[uint64]*components.StorageComponent)

	// Since businesses and villages can employ, we query anything with Identity + Storage
	storageQuery := world.Query(ecs.All(s.idID, s.storageID))
	for storageQuery.Next() {
		id := (*components.Identity)(storageQuery.Get(s.idID))
		storage := (*components.StorageComponent)(storageQuery.Get(s.storageID))
		employerStorage[id.ID] = storage
	}

	// 2. Iterate flat slice of Lumberjacks
	lumberjackQuery := world.Query(ecs.All(s.npcID, s.posID, s.jobID))

	for lumberjackQuery.Next() {
		job := (*components.JobComponent)(lumberjackQuery.Get(s.jobID))

		if job.JobID != components.JobLumberjack {
			continue // Only lumberjacks harvest wood
		}

		pos := (*components.Position)(lumberjackQuery.Get(s.posID))

		// Map position to grid index
		x, y := int(pos.X), int(pos.Y)
		if x < 0 || x >= s.mapGrid.Width || y < 0 || y >= s.mapGrid.Height {
			continue
		}

		idx := y*s.mapGrid.Width + x

		// Extract WoodValue from Geography
		if s.mapGrid.Resources[idx].WoodValue > 0 {
			// Deplete Geography
			s.mapGrid.Resources[idx].WoodValue--

			// Find employer and add to their Economy
			if job.EmployerID != 0 {
				if storage, exists := employerStorage[job.EmployerID]; exists {
					storage.Wood++
				}
			} else {
				// If self-employed or no employer, we can't store it here easily without a personal storage.
				// For the Ecology Engine, we assume the wood just drops or they have an employer.
				// In a "Total Simulation", unassigned lumberjacks might just carry it in their Needs.Wealth,
				// but to keep it perfectly DOD, we require an employer storage.
			}
		}
	}
}
