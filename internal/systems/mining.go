package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 67: The Subterranean Mining Engine (MiningSystem)
// MiningSystem bridges Geography, Biology, Economy, and Entropy.
// JobMiner NPCs physically drain Stamina to extract Iron and Stone from the MapGrid,
// lowering the tile's Elevation over time and storing resources in their Employer's Storage.

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

	// Reusable filter for querying
	minerFilter ecs.Filter
}

func NewMiningSystem(world *ecs.World, mapGrid *engine.MapGrid) *MiningSystem {
	minerFilter := filter.All(
		ecs.ComponentID[components.NPC](world),
		ecs.ComponentID[components.Position](world),
		ecs.ComponentID[components.JobComponent](world),
		ecs.ComponentID[components.VitalsComponent](world),
	)

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
		minerFilter:    minerFilter,
	}
}

func (s *MiningSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Process mining every 30 ticks (frequent but balanced)
	if s.tickStamp%30 != 0 {
		return
	}

	// 1. Build O(1) cache of active storages (Villages or Businesses)
	clear(s.activeStorages)

	vq := s.world.Query(filter.All(s.villageID, s.storageID, s.idID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		storage := (*components.StorageComponent)(vq.Get(s.storageID))
		s.activeStorages[id.ID] = storage
	}

	bq := s.world.Query(filter.All(s.businessID, s.storageID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		storage := (*components.StorageComponent)(bq.Get(s.storageID))
		s.activeStorages[id.ID] = storage
	}

	// 2. Iterate over Miners
	query := s.world.Query(s.minerFilter)
	for query.Next() {
		job := (*components.JobComponent)(query.Get(s.jobID))

		if job.JobID != components.JobMiner {
			continue
		}

		storage, exists := s.activeStorages[job.EmployerID]
		if !exists {
			continue // No storage, can't deposit
		}

		vitals := (*components.VitalsComponent)(query.Get(s.vitalsID))

		// 3. Biological constraint: Must have enough Stamina to mine
		staminaCost := float32(10.0)
		if vitals.Stamina < staminaCost {
			continue // Exhausted
		}

		pos := (*components.Position)(query.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			depot := &s.mapGrid.Resources[idx]
			tile := &s.mapGrid.Tiles[idx]

			minedAnything := false

			// 4. Extract Iron
			if depot.IronValue > 0 {
				harvestAmount := uint8(1)
				if depot.IronValue < harvestAmount {
					harvestAmount = depot.IronValue
				}
				depot.IronValue -= harvestAmount
				storage.Iron += uint32(harvestAmount)
				minedAnything = true
			}

			// 5. Extract Stone (if we didn't just extract iron, or we do both. Let's do both if available, but consume more stamina?)
			// Let's just do one or the other, prioritizing iron, or both if we want faster extraction.
			// The instructions say "extract Iron and Stone". We'll do both simultaneously for simplicity.
			if depot.StoneValue > 0 {
				harvestAmount := uint8(1)
				if depot.StoneValue < harvestAmount {
					harvestAmount = depot.StoneValue
				}
				depot.StoneValue -= harvestAmount
				storage.Stone += uint32(harvestAmount)
				minedAnything = true
			}

			// 6. Entropy/Geography: Deplete stamina and lower elevation
			if minedAnything {
				vitals.Stamina -= staminaCost
				if tile.Elevation > 0 {
					// Slow elevation drop to simulate digging into the earth
					// E.g. every few mining ticks, drop elevation. Or always drop it a bit.
					// We'll just drop it by 1 every successful mine.
					tile.Elevation -= 1

					// Recalculate biome based on new elevation
					tile.BiomeID = engine.DetermineBiome(tile.Elevation, tile.Moisture, tile.Temperature)
				}
			}
		}
	}
}
