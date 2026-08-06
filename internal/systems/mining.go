package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 67 - The Subterranean Mining Engine (MiningSystem)
// MiningSystem bridges Geography, Biology, Economy, and Entropy.
// JobMiner NPCs physically drain Stamina to extract Iron and Stone from MapGrid
// ResourceDepot arrays, storing them in their Employer's StorageComponent,
// dynamically lowering market prices, and reducing the tile's Elevation.

type MiningSystem struct {
	world       *ecs.World
	mapGrid     *engine.MapGrid
	tickCounter uint64

	// Filter
	filter ecs.Filter

	// Component IDs
	npcID      ecs.ID
	jobID      ecs.ID
	posID      ecs.ID
	vitalsID   ecs.ID
	storageID  ecs.ID
	marketID   ecs.ID
	idID       ecs.ID
	businessID ecs.ID
	villageID  ecs.ID

	// Active employer cache — cache values, not component pointers — GC
	// corruption class, see banditry.go. These maps live across ticks (and
	// across other systems' structural changes), so they hold entity handles
	// that are re-fetched via world.Get at use time, never *components.X.
	activeStorages map[uint64]ecs.Entity
	activeMarkets  map[uint64]ecs.Entity
}

func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	f := filter.All(
		ecs.ComponentID[components.NPC](world),
		ecs.ComponentID[components.JobComponent](world),
		ecs.ComponentID[components.Position](world),
		ecs.ComponentID[components.VitalsComponent](world),
	)

	return &MiningSystem{
		world:          world,
		mapGrid:        mapGrid,
		filter:         &f,
		npcID:          ecs.ComponentID[components.NPC](world),
		jobID:          ecs.ComponentID[components.JobComponent](world),
		posID:          ecs.ComponentID[components.Position](world),
		vitalsID:       ecs.ComponentID[components.VitalsComponent](world),
		storageID:      ecs.ComponentID[components.StorageComponent](world),
		marketID:       ecs.ComponentID[components.MarketComponent](world),
		idID:           ecs.ComponentID[components.Identity](world),
		businessID:     ecs.ComponentID[components.BusinessComponent](world),
		villageID:      ecs.ComponentID[components.Village](world),
		activeStorages: make(map[uint64]ecs.Entity),
		activeMarkets:  make(map[uint64]ecs.Entity),
	}
}

func (s *MiningSystem) Update(world *ecs.World) {
	s.tickCounter++

	clear(s.activeStorages)
	clear(s.activeMarkets)

	// Pre-cache Villages (entity handles only; components re-fetched at use time)
	vq := s.world.Query(filter.All(s.villageID, s.idID, s.storageID, s.marketID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		ent := vq.Entity()
		s.activeStorages[id.ID] = ent
		s.activeMarkets[id.ID] = ent
	}

	// Pre-cache Businesses
	bq := s.world.Query(filter.All(s.businessID, s.idID, s.storageID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		ent := bq.Entity()
		s.activeStorages[id.ID] = ent
		// Businesses might not always have MarketComponent, but if they do, cache it
		if s.world.Has(ent, s.marketID) {
			s.activeMarkets[id.ID] = ent
		}
	}

	eq := s.world.Query(s.filter)
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		vitals := (*components.VitalsComponent)(eq.Get(s.vitalsID))

		// Immediately skip if stamina is exhausted to prevent redundant work
		if vitals.Stamina <= 0.0 {
			continue
		}

		storageEnt, hasStorage := s.activeStorages[job.EmployerID]
		if !hasStorage || !s.world.Alive(storageEnt) {
			continue
		}
		storage := (*components.StorageComponent)(s.world.Get(storageEnt, s.storageID))

		var market *components.MarketComponent // might stay nil
		if marketEnt, hasMarket := s.activeMarkets[job.EmployerID]; hasMarket && s.world.Alive(marketEnt) {
			market = (*components.MarketComponent)(s.world.Get(marketEnt, s.marketID))
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			depot := &s.mapGrid.Resources[idx]
			tile := &s.mapGrid.Tiles[idx]

			hasMined := false

			// Extract Iron
			if depot.IronValue > 0 {
				harvestAmount := uint8(1)
				if depot.IronValue < harvestAmount {
					harvestAmount = depot.IronValue
				}
				depot.IronValue -= harvestAmount
				storage.Iron += uint32(harvestAmount)

				if market != nil && market.IronPrice > 0.01 {
					market.IronPrice -= 0.01 // Dynamically lower market prices
				}
				hasMined = true
			} else if depot.StoneValue > 0 {
				// Extract Stone if no Iron
				harvestAmount := uint8(1)
				if depot.StoneValue < harvestAmount {
					harvestAmount = depot.StoneValue
				}
				depot.StoneValue -= harvestAmount
				storage.Stone += uint32(harvestAmount)

				if market != nil && market.StonePrice > 0.01 {
					market.StonePrice -= 0.01 // Dynamically lower market prices
				}
				hasMined = true
			}

			if hasMined {
				// Drain Stamina
				vitals.Stamina -= 5.0
				if vitals.Stamina < 0.0 {
					vitals.Stamina = 0.0
				}

				// Lower Elevation Over Time (Entropy)
				// For simulation simplicity, we decrease it slightly when mining occurs
				// We'll decrement elevation every 10 ticks a miner extracts resources
				if s.tickCounter%10 == 0 {
					if tile.Elevation > 0 {
						tile.Elevation--
						// Update Biome ID
						tile.BiomeID = engine.DetermineBiome(tile.Elevation, tile.Moisture, tile.Temperature)
					}
				}
			}
		}
	}
}
