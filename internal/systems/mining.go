package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 67 - The Subterranean Mining Engine
// MiningSystem bridges Geography, Biology, and Economy by having JobMiner NPCs
// physically drain Stamina to extract Iron and Stone from MapGrid ResourceDepots,
// subsequently storing them in their Employer's StorageComponent.

type MiningSystem struct {
	world     *ecs.World
	mapGrid   *engine.MapGrid
	tickStamp uint64

	// Component IDs
	npcID      ecs.ID
	posID      ecs.ID
	jobID      ecs.ID
	vitalsID   ecs.ID
	storageID  ecs.ID
	idID       ecs.ID
	businessID ecs.ID
	villageID  ecs.ID

	// Active storage cache
	activeStorages map[uint64]*components.StorageComponent
}

// NewMiningSystem creates a new MiningSystem.
func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	return &MiningSystem{
		world:          world,
		mapGrid:        mapGrid,
		npcID:          ecs.ComponentID[components.NPC](world),
		posID:          ecs.ComponentID[components.Position](world),
		jobID:          ecs.ComponentID[components.JobComponent](world),
		vitalsID:       ecs.ComponentID[components.VitalsComponent](world),
		storageID:      ecs.ComponentID[components.StorageComponent](world),
		idID:           ecs.ComponentID[components.Identity](world),
		businessID:     ecs.ComponentID[components.BusinessComponent](world),
		villageID:      ecs.ComponentID[components.Village](world),
		activeStorages: make(map[uint64]*components.StorageComponent),
	}
}

// Update runs the mining logic.
func (s *MiningSystem) Update() {
	s.tickStamp++

	// Process mining harvesting every 60 ticks
	if s.tickStamp%60 != 0 {
		return
	}

	// 1. Build a map of active storages (Villages or Businesses) for O(1) lookup
	clear(s.activeStorages)

	// Query Villages
	vq := s.world.Query(filter.All(s.villageID, s.storageID, s.idID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		storage := (*components.StorageComponent)(vq.Get(s.storageID))
		s.activeStorages[id.ID] = storage
	}

	// Query Businesses
	bq := s.world.Query(filter.All(s.businessID, s.storageID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		storage := (*components.StorageComponent)(bq.Get(s.storageID))
		s.activeStorages[id.ID] = storage
	}

	// 2. Iterate over all Miners with Vitals
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.posID, s.vitalsID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		storage, exists := s.activeStorages[job.EmployerID]
		if !exists {
			// Employer doesn't exist or has no storage, can't deposit
			continue
		}

		vitals := (*components.VitalsComponent)(eq.Get(s.vitalsID))

		// 3. Physical Check: Miner needs Stamina to mine
		if vitals.Stamina < 10.0 {
			continue // Too exhausted to mine
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			extracted := false

			// 4. Extract Stone from the MapGrid
			if s.mapGrid.Resources[idx].StoneValue > 0 {
				harvestAmount := uint8(2) // Base harvest amount for stone
				if s.mapGrid.Resources[idx].StoneValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].StoneValue
				}
				s.mapGrid.Resources[idx].StoneValue -= harvestAmount
				storage.Stone += uint32(harvestAmount)
				extracted = true
			}

			// 5. Extract Iron from the MapGrid
			if s.mapGrid.Resources[idx].IronValue > 0 {
				harvestAmount := uint8(1) // Base harvest amount for iron (harder to mine)
				if s.mapGrid.Resources[idx].IronValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].IronValue
				}
				s.mapGrid.Resources[idx].IronValue -= harvestAmount
				storage.Iron += uint32(harvestAmount)
				extracted = true
			}

			// 6. Biological Cost (Stamina drain)
			if extracted {
				vitals.Stamina -= 5.0
				if vitals.Stamina < 0 {
					vitals.Stamina = 0
				}
			}
		}
	}
}
