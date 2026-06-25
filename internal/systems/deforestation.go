package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 55: The Ecological Collapse Engine (DeforestationSystem)
// DeforestationSystem handles the physical extraction of wood from the MapGrid
// by NPCs with JobLumberjack and deposits it into their employer's storage.

type DeforestationSystem struct {
	world     *ecs.World
	mapGrid   *engine.MapGrid
	tickStamp uint64

	// Component IDs
	npcID      ecs.ID
	posID      ecs.ID
	jobID      ecs.ID
	storageID  ecs.ID
	idID       ecs.ID
	businessID ecs.ID
	villageID  ecs.ID

	// Active storage cache
	activeStorages map[uint64]*components.StorageComponent
}

// NewDeforestationSystem creates a new DeforestationSystem.
func NewDeforestationSystem(world *ecs.World, mapGrid *engine.MapGrid) *DeforestationSystem {
	return &DeforestationSystem{
		world:          world,
		mapGrid:        mapGrid,
		npcID:          ecs.ComponentID[components.NPC](world),
		posID:          ecs.ComponentID[components.Position](world),
		jobID:          ecs.ComponentID[components.JobComponent](world),
		storageID:      ecs.ComponentID[components.StorageComponent](world),
		idID:           ecs.ComponentID[components.Identity](world),
		businessID:     ecs.ComponentID[components.BusinessComponent](world),
		villageID:      ecs.ComponentID[components.Village](world),
		activeStorages: make(map[uint64]*components.StorageComponent),
	}
}

// Update runs the deforestation logic.
func (s *DeforestationSystem) Update() {
	s.tickStamp++

	// Process lumberjack harvesting every 60 ticks (1 in-game hour approx)
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

	// 2. Iterate over all Lumberjacks
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.posID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobLumberjack {
			continue
		}

		storage, exists := s.activeStorages[job.EmployerID]
		if !exists {
			// Employer doesn't exist or has no storage, can't deposit
			continue
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			// 3. Extract Wood from the MapGrid
			if s.mapGrid.Resources[idx].WoodValue > 0 {
				harvestAmount := uint8(1) // Base harvest amount
				if s.mapGrid.Resources[idx].WoodValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].WoodValue
				}

				s.mapGrid.Resources[idx].WoodValue -= harvestAmount

				// Prevent compounding float issues, add directly to Storage.Wood
				storage.Wood += uint32(harvestAmount)

				// 4. Ecological Collapse (Deforestation)
				if s.mapGrid.Resources[idx].WoodValue == 0 {
					tile := s.mapGrid.Tiles[idx]

					// If the biome is a forest, downgrade it to grassland
					if tile.BiomeID == engine.BiomeTemperateDeciduousForest ||
						tile.BiomeID == engine.BiomeTemperateRainForest ||
						tile.BiomeID == engine.BiomeTropicalSeasonalForest ||
						tile.BiomeID == engine.BiomeTropicalRainForest {

						tile.BiomeID = engine.BiomeGrassland
						s.mapGrid.Tiles[idx] = tile
					}
				}
			}
		}
	}
}
