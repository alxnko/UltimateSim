package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 67: The Subterranean Mining Engine (MiningSystem)
// MiningSystem bridges Geography, Biology, Economy, and Entropy by having
// JobMiner NPCs physically drain Stamina to extract Iron and Stone from
// engine.MapGrid ResourceDepot arrays, storing them in their Employer's
// StorageComponent and dynamically lowering market prices.

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
	marketID   ecs.ID
	idID       ecs.ID
	businessID ecs.ID
	villageID  ecs.ID

	// Active employers cache
	activeStorages map[uint64]*components.StorageComponent
	activeMarkets  map[uint64]*components.MarketComponent
}

func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	return &MiningSystem{
		world:          world,
		mapGrid:        mapGrid,
		npcID:          ecs.ComponentID[components.NPC](world),
		posID:          ecs.ComponentID[components.Position](world),
		jobID:          ecs.ComponentID[components.JobComponent](world),
		vitalsID:       ecs.ComponentID[components.VitalsComponent](world),
		storageID:      ecs.ComponentID[components.StorageComponent](world),
		marketID:       ecs.ComponentID[components.MarketComponent](world),
		idID:           ecs.ComponentID[components.Identity](world),
		businessID:     ecs.ComponentID[components.BusinessComponent](world),
		villageID:      ecs.ComponentID[components.Village](world),
		activeStorages: make(map[uint64]*components.StorageComponent),
		activeMarkets:  make(map[uint64]*components.MarketComponent),
	}
}

func (s *MiningSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Process miner extraction every 30 ticks
	if s.tickStamp%30 != 0 {
		return
	}

	// 1. Build a map of active employers (Villages or Businesses) for O(1) lookup
	clear(s.activeStorages)
	clear(s.activeMarkets)

	// Query Villages
	vq := s.world.Query(filter.All(s.villageID, s.storageID, s.marketID, s.idID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		storage := (*components.StorageComponent)(vq.Get(s.storageID))
		market := (*components.MarketComponent)(vq.Get(s.marketID))
		s.activeStorages[id.ID] = storage
		s.activeMarkets[id.ID] = market
	}

	// Query Businesses
	bq := s.world.Query(filter.All(s.businessID, s.storageID, s.marketID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		storage := (*components.StorageComponent)(bq.Get(s.storageID))
		market := (*components.MarketComponent)(bq.Get(s.marketID))
		s.activeStorages[id.ID] = storage
		s.activeMarkets[id.ID] = market
	}

	// 2. Iterate over all Miners
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.posID, s.vitalsID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		vitals := (*components.VitalsComponent)(eq.Get(s.vitalsID))

		// Ensure Miner has enough stamina
		if vitals.Stamina < 5.0 {
			continue
		}

		storage, existsStorage := s.activeStorages[job.EmployerID]
		market, existsMarket := s.activeMarkets[job.EmployerID]

		if !existsStorage || !existsMarket {
			continue
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			extracted := false

			// 3. Extract Iron from the MapGrid
			if s.mapGrid.Resources[idx].IronValue > 0 {
				harvestAmount := uint8(1)
				if s.mapGrid.Resources[idx].IronValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].IronValue
				}

				s.mapGrid.Resources[idx].IronValue -= harvestAmount
				storage.Iron += uint32(harvestAmount)

				// Dynamically lower market price
				market.IronPrice -= 0.1
				if market.IronPrice < 1.0 {
					market.IronPrice = 1.0
				}

				extracted = true
			}

			// 4. Extract Stone from the MapGrid
			if s.mapGrid.Resources[idx].StoneValue > 0 {
				harvestAmount := uint8(1)
				if s.mapGrid.Resources[idx].StoneValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].StoneValue
				}

				s.mapGrid.Resources[idx].StoneValue -= harvestAmount
				storage.Stone += uint32(harvestAmount)

				// Dynamically lower market price
				market.StonePrice -= 0.1
				if market.StonePrice < 1.0 {
					market.StonePrice = 1.0
				}

				extracted = true
			}

			// 5. Biological Cost
			if extracted {
				vitals.Stamina -= 5.0
			}
		}
	}
}
