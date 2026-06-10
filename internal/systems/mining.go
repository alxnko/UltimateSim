package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 67 - The Subterranean Mining Engine
// Bridges Geography, Biology, Economy, and Entropy.
// JobMiner NPCs drain stamina to extract Iron and Stone from MapGrid ResourceDepots,
// store them in Employer Storage, lower market prices, and reduce tile Elevation over time.

type employerMiningData struct {
	Storage *components.StorageComponent
	Market  *components.MarketComponent
}

type MiningSystem struct {
	world   *ecs.World
	mapGrid *engine.MapGrid

	villageFilter  ecs.Filter
	businessFilter ecs.Filter
	minerFilter    ecs.Filter

	employerCache map[uint64]*employerMiningData
}

func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	vID := ecs.ComponentID[components.Village](world)
	sID := ecs.ComponentID[components.StorageComponent](world)
	mID := ecs.ComponentID[components.MarketComponent](world)
	idID := ecs.ComponentID[components.Identity](world)

	villageMask := filter.All(vID, sID, mID, idID)

	bID := ecs.ComponentID[components.BusinessComponent](world)
	businessMask := filter.All(bID, sID, mID, idID)

	npcID := ecs.ComponentID[components.NPC](world)
	jID := ecs.ComponentID[components.JobComponent](world)
	pID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)

	minerMask := filter.All(npcID, jID, pID, vitalsID)

	return &MiningSystem{
		world:          world,
		mapGrid:        mapGrid,
		villageFilter:  villageMask,
		businessFilter: businessMask,
		minerFilter:    minerMask,
		employerCache:  make(map[uint64]*employerMiningData),
	}
}

func (s *MiningSystem) Update(world *ecs.World) {
	s.updateCaches()
	s.processMiners()
}

func (s *MiningSystem) updateCaches() {
	clear(s.employerCache)

	sID := ecs.ComponentID[components.StorageComponent](s.world)
	mID := ecs.ComponentID[components.MarketComponent](s.world)
	idID := ecs.ComponentID[components.Identity](s.world)

	// Cache Villages
	qVill := s.world.Query(s.villageFilter)
	for qVill.Next() {
		id := (*components.Identity)(qVill.Get(idID))
		stor := (*components.StorageComponent)(qVill.Get(sID))
		mark := (*components.MarketComponent)(qVill.Get(mID))

		s.employerCache[id.ID] = &employerMiningData{
			Storage: stor,
			Market:  mark,
		}
	}

	// Cache Businesses
	qBus := s.world.Query(s.businessFilter)
	for qBus.Next() {
		id := (*components.Identity)(qBus.Get(idID))
		stor := (*components.StorageComponent)(qBus.Get(sID))
		mark := (*components.MarketComponent)(qBus.Get(mID))

		s.employerCache[id.ID] = &employerMiningData{
			Storage: stor,
			Market:  mark,
		}
	}
}

func (s *MiningSystem) processMiners() {
	if s.mapGrid == nil {
		return
	}

	jID := ecs.ComponentID[components.JobComponent](s.world)
	pID := ecs.ComponentID[components.Position](s.world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](s.world)

	qMiner := s.world.Query(s.minerFilter)
	for qMiner.Next() {
		job := (*components.JobComponent)(qMiner.Get(jID))
		if job.JobID != components.JobMiner {
			continue
		}

		vitals := (*components.VitalsComponent)(qMiner.Get(vitalsID))
		// Need stamina to mine
		if vitals.Stamina < 10.0 {
			continue
		}

		pos := (*components.Position)(qMiner.Get(pID))
		x, y := int(pos.X), int(pos.Y)

		if x < 0 || y < 0 || x >= s.mapGrid.Width || y >= s.mapGrid.Height {
			continue
		}

		idx := y*s.mapGrid.Width + x
		depot := &s.mapGrid.Resources[idx]
		tile := &s.mapGrid.Tiles[idx]

		employer, eExists := s.employerCache[job.EmployerID]
		if !eExists {
			continue
		}

		// Perform mining
		mined := false
		if depot.IronValue > 0 {
			depot.IronValue--
			employer.Storage.Iron++
			employer.Market.IronPrice *= 0.99 // Price drops
			mined = true
		} else if depot.StoneValue > 0 {
			depot.StoneValue--
			employer.Storage.Stone++
			employer.Market.StonePrice *= 0.99 // Price drops
			mined = true
		}

		if mined {
			vitals.Stamina -= 5.0
			if tile.Elevation > 0 {
				tile.Elevation-- // Entropy
			}
		}
	}
}
