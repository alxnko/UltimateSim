package render

import (
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 08.3: Map Rendering & Biomes
// Color Mapping: Map TileData.BiomeID constants to static color arrays.

// BiomeColors maps the BiomeID to a specific color for rendering.
var BiomeColors = map[uint8]color.RGBA{
	engine.BiomeOcean:                    {R: 0, G: 0, B: 255, A: 255},       // Ocean = Blue (#0000FF)
	engine.BiomeBeach:                    {R: 237, G: 201, B: 175, A: 255},   // Beach = Sandy
	engine.BiomeScorched:                 {R: 85, G: 85, B: 85, A: 255},      // Scorched = Dark Gray
	engine.BiomeBare:                     {R: 136, G: 136, B: 136, A: 255},   // Bare = Gray
	engine.BiomeTundra:                   {R: 255, G: 255, B: 255, A: 255},   // Tundra = White (#FFFFFF)
	engine.BiomeSnow:                     {R: 240, G: 240, B: 255, A: 255},   // Snow = Whiteish
	engine.BiomeTemperateDesert:          {R: 201, G: 210, B: 155, A: 255},   // Temperate Desert = Light Green/Yellow
	engine.BiomeShrubland:                {R: 136, G: 153, B: 119, A: 255},   // Shrubland = Muted Green
	engine.BiomeGrassland:                {R: 136, G: 170, B: 85, A: 255},    // Grassland = Green
	engine.BiomeTemperateDeciduousForest:{R: 103, G: 148, B: 89, A: 255},    // Temperate Forest = Darker Green
	engine.BiomeTemperateRainForest:      {R: 68, G: 136, B: 85, A: 255},     // Temperate Rain Forest = Deep Green
	engine.BiomeSubtropicalDesert:        {R: 210, G: 185, B: 139, A: 255},   // Subtropical Desert = Tan
	engine.BiomeTropicalSeasonalForest:   {R: 85, G: 153, B: 68, A: 255},     // Tropical Seasonal = Vibrant Green
	engine.BiomeTropicalRainForest:       {R: 51, G: 119, B: 85, A: 255},     // Tropical Rain Forest = Dark Vibrant Green
	engine.BiomeMountain:                 {R: 100, G: 100, B: 100, A: 255},   // Mountain = Dark Gray/Brownish
}
