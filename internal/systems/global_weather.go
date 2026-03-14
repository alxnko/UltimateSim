package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 19.2: Ecological Drift (Climate Change)
// GlobalWeatherSystem evaluates the global map every 100,000 ticks.
// It uniformly shifts temperature, potentially mutating biomes and localized resources.

type GlobalWeatherSystem struct {
	mapGrid     *engine.MapGrid
	currentTick uint64
}

// NewGlobalWeatherSystem creates a new GlobalWeatherSystem.
func NewGlobalWeatherSystem(world *ecs.World, mapGrid *engine.MapGrid) *GlobalWeatherSystem {
	return &GlobalWeatherSystem{
		mapGrid:     mapGrid,
		currentTick: 0,
	}
}

// Update executes the system logic per tick.
func (s *GlobalWeatherSystem) Update(world *ecs.World) {
	s.currentTick++

	// Execute every 100,000 ticks
	if s.currentTick > 0 && s.currentTick%100000 == 0 {
		// Calculate the degree to which we want to shift climate
		// For deterministic simplicity, we increment global temperatures by 5
		tempShift := uint8(5)

		for i := 0; i < len(s.mapGrid.Tiles); i++ {
			// Increase temperature, bounded at 255
			oldTemp := s.mapGrid.Tiles[i].Temperature
			if uint16(oldTemp)+uint16(tempShift) > 255 {
				s.mapGrid.Tiles[i].Temperature = 255
			} else {
				s.mapGrid.Tiles[i].Temperature += tempShift
			}

			// If a forest exists and becomes too hot, it mutates to Grassland (Plains)
			biome := s.mapGrid.Tiles[i].BiomeID
			if biome == engine.BiomeTemperateDeciduousForest ||
				biome == engine.BiomeTemperateRainForest ||
				biome == engine.BiomeTropicalSeasonalForest ||
				biome == engine.BiomeTropicalRainForest {

				// Check if the new temperature crosses a threshold where it dries up.
				// Based on determineBiome, temperate is < 170. If it pushes past, it might become SubtropicalDesert,
				// but for phase 19.2 explicit mechanic: "Forests may dry into Plains, rendering localized WoodValue drops"
				// We enforce mutation strictly if it exceeds 165 for temperate or is just uniformly dried via math

				// Using deterministic chance driven by exact temperature difference
				if s.mapGrid.Tiles[i].Temperature > 165 {
					// Mutate into Grassland (Plains equivalent)
					s.mapGrid.Tiles[i].BiomeID = engine.BiomeGrassland

					// Nullify the wood value for that specific coordinate index, forcing geopolitics
					s.mapGrid.Resources[i].WoodValue = 0
				}
			}
		}
	}
}
