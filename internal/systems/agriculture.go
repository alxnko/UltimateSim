package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 58: The Agricultural Engine (AgricultureSystem)
// AgricultureSystem bridges Ecology and Economy by having JobFarmer NPCs extract
// Food from the MapGrid into their employer's storage. It naturally depletes
// the tile's Moisture, triggering organic desertification if overfarmed.

type AgricultureSystem struct {
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

// NewAgricultureSystem creates a new AgricultureSystem.
func NewAgricultureSystem(world *ecs.World, mapGrid *engine.MapGrid) *AgricultureSystem {
	return &AgricultureSystem{
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

// Update runs the agriculture logic.
func (s *AgricultureSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Process farmer harvesting every 60 ticks
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

	// 2. Iterate over all Farmers
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.posID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))

		if job.JobID != components.JobFarmer {
			continue
		}

		storage, exists := s.activeStorages[job.EmployerID]
		if !exists {
			continue
		}

		pos := (*components.Position)(eq.Get(s.posID))
		gridX, gridY := int(pos.X), int(pos.Y)

		if gridX >= 0 && gridX < s.mapGrid.Width && gridY >= 0 && gridY < s.mapGrid.Height {
			idx := gridY*s.mapGrid.Width + gridX

			// 3. Extract Food from the MapGrid
			if s.mapGrid.Resources[idx].FoodValue > 0 {
				harvestAmount := uint8(1)
				if s.mapGrid.Resources[idx].FoodValue < harvestAmount {
					harvestAmount = s.mapGrid.Resources[idx].FoodValue
				}

				s.mapGrid.Resources[idx].FoodValue -= harvestAmount
				storage.Food += uint32(harvestAmount)

				// 4. Ecological Cost (Desertification)
				if s.mapGrid.Tiles[idx].Moisture > 0 {
					s.mapGrid.Tiles[idx].Moisture--

					// Recalculate Biome based on new lower moisture
					tile := s.mapGrid.Tiles[idx]
					s.mapGrid.Tiles[idx].BiomeID = engine.DetermineBiome(tile.Elevation, tile.Moisture, tile.Temperature)
				}
			}
		}
	}
}
