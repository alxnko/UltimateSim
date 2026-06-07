package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 67: The Subterranean Mining Engine
// MiningSystem handles physical extraction of stone and iron by NPCs with JobMiner,
// depositing the resources to their employer, affecting local market supply (prices),
// depleting the map resources, and physically digging into the terrain (Elevation drop)
// while exhausting the NPC's Stamina.

type miningEmployerData struct {
	storage *components.StorageComponent
	market  *components.MarketComponent
}

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

	// Active employer cache
	activeEmployers map[uint64]miningEmployerData
}

// NewMiningSystem creates a new MiningSystem.
func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	return &MiningSystem{
		world:           world,
		mapGrid:         mapGrid,
		npcID:           ecs.ComponentID[components.NPC](world),
		posID:           ecs.ComponentID[components.Position](world),
		jobID:           ecs.ComponentID[components.JobComponent](world),
		vitalsID:        ecs.ComponentID[components.VitalsComponent](world),
		storageID:       ecs.ComponentID[components.StorageComponent](world),
		marketID:        ecs.ComponentID[components.MarketComponent](world),
		idID:            ecs.ComponentID[components.Identity](world),
		businessID:      ecs.ComponentID[components.BusinessComponent](world),
		villageID:       ecs.ComponentID[components.Village](world),
		activeEmployers: make(map[uint64]miningEmployerData),
	}
}

// Update runs the mining logic.
func (s *MiningSystem) Update() {
	s.tickStamp++

	// Process mining every 50 ticks to balance resource acquisition
	if s.tickStamp%50 != 0 {
		return
	}

	// 1. Build a map of active employers (Villages or Businesses) for O(1) lookup
	// Only those with storage and market are fully engaged in the simulation loop.
	clear(s.activeEmployers)

	// Query Villages
	vq := s.world.Query(filter.All(s.villageID, s.storageID, s.marketID, s.idID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		storage := (*components.StorageComponent)(vq.Get(s.storageID))
		market := (*components.MarketComponent)(vq.Get(s.marketID))
		s.activeEmployers[id.ID] = miningEmployerData{storage: storage, market: market}
	}

	// Query Businesses
	bq := s.world.Query(filter.All(s.businessID, s.storageID, s.marketID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		storage := (*components.StorageComponent)(bq.Get(s.storageID))
		market := (*components.MarketComponent)(bq.Get(s.marketID))
		s.activeEmployers[id.ID] = miningEmployerData{storage: storage, market: market}
	}

	// 2. Iterate over all Miners
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.posID, s.vitalsID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		employer, exists := s.activeEmployers[job.EmployerID]
		if !exists {
			// Employer doesn't exist or lacks required components, can't deposit
			continue
		}

		vitals := (*components.VitalsComponent)(eq.Get(s.vitalsID))

		// If too exhausted or unconscious, cannot mine
		if vitals.Stamina < 10.0 || vitals.Consciousness <= 0 {
			continue
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			stoneMined := false
			ironMined := false

			// 3. Extract Stone from the MapGrid
			if s.mapGrid.Resources[idx].StoneValue > 0 {
				harvestAmount := uint8(2)
				if s.mapGrid.Resources[idx].StoneValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].StoneValue
				}
				s.mapGrid.Resources[idx].StoneValue -= harvestAmount
				employer.storage.Stone += uint32(harvestAmount)

				// Market price drops organically as supply increases
				employer.market.StonePrice *= 0.99
				if employer.market.StonePrice < 1.0 {
					employer.market.StonePrice = 1.0
				}
				stoneMined = true
			}

			// 4. Extract Iron from the MapGrid
			if s.mapGrid.Resources[idx].IronValue > 0 {
				harvestAmount := uint8(1)
				if s.mapGrid.Resources[idx].IronValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].IronValue
				}
				s.mapGrid.Resources[idx].IronValue -= harvestAmount
				employer.storage.Iron += uint32(harvestAmount)

				// Market price drops organically as supply increases
				employer.market.IronPrice *= 0.99
				if employer.market.IronPrice < 1.0 {
					employer.market.IronPrice = 1.0
				}
				ironMined = true
			}

			// 5. Apply Biological & Geographic Consequences
			if stoneMined || ironMined {
				// Digging drains stamina
				vitals.Stamina -= 5.0
				if vitals.Stamina < 0 {
					vitals.Stamina = 0
				}

				// The physical act of mining hollows out the terrain
				if s.mapGrid.Tiles[idx].Elevation > 0 {
					s.mapGrid.Tiles[idx].Elevation--
				}
			}
		}
	}
}
