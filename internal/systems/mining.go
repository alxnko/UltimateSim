package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 67 - The Subterranean Mining Engine (MiningSystem)
// MiningSystem bridges Geography, Biology, Economy, and Entropy by having JobMiner NPCs
// physically drain Stamina to extract Iron and Stone from MapGrid arrays, subsequently
// storing them in their Employer's StorageComponent and dynamically lowering market prices.
// It also permanently alters the map by reducing the tile's Elevation over time.

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

	// Active storage cache (for employers)
	activeStorages map[uint64]*components.StorageComponent
	activeMarkets  map[uint64]*components.MarketComponent
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
		marketID:       ecs.ComponentID[components.MarketComponent](world),
		idID:           ecs.ComponentID[components.Identity](world),
		businessID:     ecs.ComponentID[components.BusinessComponent](world),
		villageID:      ecs.ComponentID[components.Village](world),
		activeStorages: make(map[uint64]*components.StorageComponent),
		activeMarkets:  make(map[uint64]*components.MarketComponent),
	}
}

// Update runs the mining logic.
func (s *MiningSystem) Update() {
	s.tickStamp++

	// Process mining every 30 ticks
	if s.tickStamp%30 != 0 {
		return
	}

	// 1. Build a map of active storages and markets for O(1) lookup
	clear(s.activeStorages)
	clear(s.activeMarkets)

	// Query Villages
	vq := s.world.Query(filter.All(s.villageID, s.storageID, s.idID, s.marketID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		storage := (*components.StorageComponent)(vq.Get(s.storageID))
		market := (*components.MarketComponent)(vq.Get(s.marketID))
		s.activeStorages[id.ID] = storage
		s.activeMarkets[id.ID] = market
	}

	// Query Businesses
	bq := s.world.Query(filter.All(s.businessID, s.storageID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		storage := (*components.StorageComponent)(bq.Get(s.storageID))
		s.activeStorages[id.ID] = storage
        // Note: Businesses may not have a MarketComponent, which is fine, we just won't update their prices.
        if s.world.Has(bq.Entity(), s.marketID) {
            market := (*components.MarketComponent)(bq.Get(s.marketID))
            s.activeMarkets[id.ID] = market
        }
	}

	// 2. Iterate over all Miners
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.posID, s.vitalsID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		storage, exists := s.activeStorages[job.EmployerID]
		if !exists {
			continue // Employer doesn't exist or has no storage
		}

		vitals := (*components.VitalsComponent)(eq.Get(s.vitalsID))

		// If stamina is too low, the miner cannot work
		if vitals.Stamina < 5.0 {
			continue
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX
			depot := &s.mapGrid.Resources[idx]
            extractedSomething := false

			// 3. Extract Iron and Stone from the MapGrid
			if depot.IronValue > 0 {
				depot.IronValue--
				storage.Iron++
                extractedSomething = true
			}
			if depot.StoneValue > 0 {
				depot.StoneValue--
				storage.Stone++
                extractedSomething = true
			}

            if extractedSomething {
                // Drain stamina
                vitals.Stamina -= 2.0

                // 4. Alter the Map (Elevation drop)
                tile := s.mapGrid.Tiles[idx]
                if tile.Elevation > 0 {
                    tile.Elevation--
                    s.mapGrid.Tiles[idx] = tile
                }

                // 5. Lower local market prices organically
                if market, mExists := s.activeMarkets[job.EmployerID]; mExists {
                    // Mining increases supply locally, lowering price
                    if market.IronPrice > 1.0 {
                        market.IronPrice -= 0.1
                    }
                    if market.StonePrice > 1.0 {
                        market.StonePrice -= 0.1
                    }
                }
            }
		}
	}
}
