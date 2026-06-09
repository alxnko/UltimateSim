package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 67: The Subterranean Mining Engine
// Bridges Geography, Biology, Economy, and Entropy by having JobMiner NPCs
// physically drain Stamina to extract Iron/Stone, subsequently storing it
// in their Employer's StorageComponent, lowering market prices, and reducing tile Elevation.

type employerMiningData struct {
	Storage *components.StorageComponent
	Market  *components.MarketComponent
}

type MiningSystem struct {
	world   *ecs.World
	mapGrid *engine.MapGrid

	minerFilter    ecs.Filter
	villageFilter  ecs.Filter
	businessFilter ecs.Filter

	employerCache map[uint64]*employerMiningData

	jobID     ecs.ID
	posID     ecs.ID
	vitalsID  ecs.ID
	identID   ecs.ID
	storageID ecs.ID
	marketID  ecs.ID
}

func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	identID := ecs.ComponentID[components.Identity](world)

	minerMask := filter.All(jobID, posID, vitalsID)

	storageID := ecs.ComponentID[components.StorageComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	businessID := ecs.ComponentID[components.BusinessComponent](world)

	villageMask := filter.All(identID, storageID, marketID, villageID)
	businessMask := filter.All(identID, storageID, marketID, businessID)

	return &MiningSystem{
		world:          world,
		mapGrid:        mapGrid,
		minerFilter:    &minerMask,
		villageFilter:  &villageMask,
		businessFilter: &businessMask,
		employerCache:  make(map[uint64]*employerMiningData),
		jobID:          jobID,
		posID:          posID,
		vitalsID:       vitalsID,
		identID:        identID,
		storageID:      storageID,
		marketID:       marketID,
	}
}

func (s *MiningSystem) Update(world *ecs.World) {
	if s.mapGrid == nil {
		return
	}

	// 1. Pre-cache employer data into an O(1) map for DOD performance
	clear(s.employerCache)

	vQuery := world.Query(s.villageFilter)
	for vQuery.Next() {
		id := (*components.Identity)(vQuery.Get(s.identID))
		storage := (*components.StorageComponent)(vQuery.Get(s.storageID))
		market := (*components.MarketComponent)(vQuery.Get(s.marketID))
		s.employerCache[id.ID] = &employerMiningData{
			Storage: storage,
			Market:  market,
		}
	}

	bQuery := world.Query(s.businessFilter)
	for bQuery.Next() {
		id := (*components.Identity)(bQuery.Get(s.identID))
		storage := (*components.StorageComponent)(bQuery.Get(s.storageID))
		market := (*components.MarketComponent)(bQuery.Get(s.marketID))
		s.employerCache[id.ID] = &employerMiningData{
			Storage: storage,
			Market:  market,
		}
	}

	// 2. Process active miners
	mQuery := world.Query(s.minerFilter)
	for mQuery.Next() {
		job := (*components.JobComponent)(mQuery.Get(s.jobID))
		if job.JobID != components.JobMiner {
			continue
		}

		vitals := (*components.VitalsComponent)(mQuery.Get(s.vitalsID))
		if vitals.Stamina <= 0 {
			continue // Too tired to mine
		}

		employer, exists := s.employerCache[job.EmployerID]
		if !exists {
			continue // No valid employer
		}

		pos := (*components.Position)(mQuery.Get(s.posID))
		x := int(pos.X)
		y := int(pos.Y)

		if x < 0 || x >= s.mapGrid.Width || y < 0 || y >= s.mapGrid.Height {
			continue // Out of bounds
		}

		idx := y*s.mapGrid.Width + x

		// Try to extract Iron first
		if s.mapGrid.Resources[idx].IronValue > 0 {
			// Biological bridge
			vitals.Stamina -= 2.0
			if vitals.Stamina < 0 {
				vitals.Stamina = 0
			}

			// Geography -> Economy bridge
			s.mapGrid.Resources[idx].IronValue--
			employer.Storage.Iron++

			// Price saturation (Economy bridge)
			employer.Market.IronPrice *= 0.99
			if employer.Market.IronPrice < 1.0 {
				employer.Market.IronPrice = 1.0
			}

			// Environmental entropy / scarring
			if s.mapGrid.Tiles[idx].Elevation > 0 {
				s.mapGrid.Tiles[idx].Elevation--
			}

			continue // Acted this tick
		}

		// Try to extract Stone
		if s.mapGrid.Resources[idx].StoneValue > 0 {
			// Biological bridge
			vitals.Stamina -= 1.0
			if vitals.Stamina < 0 {
				vitals.Stamina = 0
			}

			// Geography -> Economy bridge
			s.mapGrid.Resources[idx].StoneValue--
			employer.Storage.Stone++

			// Price saturation (Economy bridge)
			employer.Market.StonePrice *= 0.99
			if employer.Market.StonePrice < 1.0 {
				employer.Market.StonePrice = 1.0
			}

			// Environmental entropy / scarring
			if s.mapGrid.Tiles[idx].Elevation > 0 {
				s.mapGrid.Tiles[idx].Elevation--
			}
		}
	}
}
