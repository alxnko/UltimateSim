package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 67 - The Subterranean Mining Engine
// Bridges Geography (Resource Nodes), Economy (Storage/Market), Biology (Stamina/Pain),
// and Entropy (Cave-ins).

type MineData struct {
	Entity ecs.Entity
	Mine   *components.MineComponent
}

type EmployerData struct {
	Entity  ecs.Entity
	Storage *components.StorageComponent
	Market  *components.MarketComponent
}

type MiningSystem struct {
	mapGrid *engine.MapGrid

	// Reusable slices/maps for DOD zero-allocation
	employersCache map[uint64]EmployerData
	minesCache     []MineData

	// Filters
	minerFilter    ecs.Filter
	mineFilter     ecs.Filter
	employerFilter ecs.Filter

	// Component IDs
	npcID      ecs.ID
	jobID      ecs.ID
	posID      ecs.ID
	vitalsID   ecs.ID
	mineID     ecs.ID
	storageID  ecs.ID
	marketID   ecs.ID
	villageID  ecs.ID
	businessID ecs.ID
	identID    ecs.ID
}

// NewMiningSystem creates a new MiningSystem.
func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	npcID := ecs.ComponentID[components.NPC](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)

	mineID := ecs.ComponentID[components.MineComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	businessID := ecs.ComponentID[components.BusinessEntity](world)
	identID := ecs.ComponentID[components.Identity](world)

	minerMask := filter.All(npcID, jobID, posID, vitalsID)
	mineMask := filter.All(mineID)
	// Employers must have Storage and Market (usually Villages)
	// For simplicity in filter, query all with Storage/Market and check for Village/Business inside loop
	empMask := filter.All(identID, storageID, marketID)

	return &MiningSystem{
		mapGrid:        mapGrid,
		employersCache: make(map[uint64]EmployerData, 100),
		minesCache:     make([]MineData, 0, 100),
		minerFilter:    minerMask,
		mineFilter:     mineMask,
		employerFilter: empMask,
		npcID:          npcID,
		jobID:          jobID,
		posID:          posID,
		vitalsID:       vitalsID,
		mineID:         mineID,
		storageID:      storageID,
		marketID:       marketID,
		villageID:      villageID,
		businessID:     businessID,
		identID:        identID,
	}
}

// Update evaluates miners and executes resource extraction.
func (s *MiningSystem) Update(world *ecs.World) {
	// 1. Cache Employers mapped by Identity ID (Zero-allocation DOD optimization)
	clear(s.employersCache)
	empQuery := world.Query(s.employerFilter)
	for empQuery.Next() {
		entity := empQuery.Entity()
		if !world.Has(entity, s.villageID) && !world.Has(entity, s.businessID) {
			continue
		}

		ident := (*components.Identity)(empQuery.Get(s.identID))
		storage := (*components.StorageComponent)(empQuery.Get(s.storageID))
		market := (*components.MarketComponent)(empQuery.Get(s.marketID))
		s.employersCache[ident.ID] = EmployerData{
			Entity:  entity,
			Storage: storage,
			Market:  market,
		}
	}

	// 2. Cache Mines (Zero-allocation DOD optimization)
	s.minesCache = s.minesCache[:0]
	mineQuery := world.Query(s.mineFilter)
	for mineQuery.Next() {
		mine := (*components.MineComponent)(mineQuery.Get(s.mineID))
		s.minesCache = append(s.minesCache, MineData{
			Entity: mineQuery.Entity(),
			Mine:   mine,
		})
	}

	// 3. Process Miners
	minerQuery := world.Query(s.minerFilter)
	for minerQuery.Next() {
		job := (*components.JobComponent)(minerQuery.Get(s.jobID))

		// Only process JobMiner
		if job.JobID != components.JobMiner || job.EmployerID == 0 {
			continue
		}

		pos := (*components.Position)(minerQuery.Get(s.posID))
		vitals := (*components.VitalsComponent)(minerQuery.Get(s.vitalsID))

		// Find the mine for this employer
		var activeMine *components.MineComponent
		for _, m := range s.minesCache {
			if m.Mine.EmployerID == job.EmployerID {
				activeMine = m.Mine
				break
			}
		}

		if activeMine == nil {
			continue // No physical mine to work at
		}

		// Distance check
		dx := pos.X - activeMine.X
		dy := pos.Y - activeMine.Y
		distSq := dx*dx + dy*dy

		if distSq > 1.0 {
			continue // Too far, must physically walk to the mine
		}

		// At the mine. Process stamina.
		if vitals.Stamina >= 2.0 {
			vitals.Stamina -= 2.0
		} else {
			vitals.Stamina = 0
			// Exhaustion -> Pain -> Stress
			if vitals.Pain < 100 {
				vitals.Pain += 5.0
				if vitals.Pain > 100 {
					vitals.Pain = 100
				}
			}
		}

		// Only extract if they have stamina or are pushing through the pain
		if vitals.Stamina > 0 || vitals.Pain < 100 {
			// Get MapGrid tile
			x, y := int(activeMine.X), int(activeMine.Y)
			if x >= 0 && x < s.mapGrid.Width && y >= 0 && y < s.mapGrid.Height {
				idx := y*s.mapGrid.Width + x
				depot := &s.mapGrid.Resources[idx]

				extractedIron := uint32(0)
				extractedStone := uint32(0)

				if depot.IronValue > 0 {
					depot.IronValue--
					extractedIron++
				}
				if depot.StoneValue > 0 {
					depot.StoneValue--
					extractedStone++
				}

				if extractedIron > 0 || extractedStone > 0 {
					// Add to employer storage
					if empData, exists := s.employersCache[job.EmployerID]; exists {
						empData.Storage.Iron += extractedIron
						empData.Storage.Stone += extractedStone

						// Butterfly Effect: Lower local prices to stimulate economy
						if extractedIron > 0 && empData.Market.IronPrice > 0.1 {
							empData.Market.IronPrice -= 0.05
						}
						if extractedStone > 0 && empData.Market.StonePrice > 0.1 {
							empData.Market.StonePrice -= 0.05
						}
					}

					// Cave-in RNG based on mine depth
					if activeMine.Depth > 0 {
						// e.g. Depth 100 = 100/10000 = 1% chance per tick
						roll := engine.GetRandomInt() % 10000
						if roll < int(activeMine.Depth) {
							// Cave-in!
							vitals.Blood -= 50.0
							vitals.Pain += 50.0
							if vitals.Blood < 0 {
								vitals.Blood = 0
							}
							if vitals.Pain > 100 {
								vitals.Pain = 100
							}
						}
					}
				}
			}
		}
	}
}
