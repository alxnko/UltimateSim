package engine

import (
	"testing"
)

// Phase 02.3: Biome Mapping Tests
func TestDetermineBiome(t *testing.T) {
	tests := []struct {
		name        string
		elevation   uint8
		moisture    uint8
		temperature uint8
		expected    uint8
	}{
		{"Ocean", 50, 200, 200, BiomeOcean},
		{"Beach", 90, 200, 200, BiomeBeach},
		{"Snow (High Elev)", 220, 150, 50, BiomeSnow},
		{"Bare (High Elev, Dry)", 220, 50, 150, BiomeBare},
		{"Mountain (High Elev, Normal)", 220, 150, 150, BiomeMountain},
		{"Tundra (Cold, Dry)", 100, 100, 50, BiomeTundra},
		{"Snow (Cold, Wet)", 100, 200, 50, BiomeSnow},
		{"Temperate Desert", 100, 50, 150, BiomeTemperateDesert},
		{"Grassland", 100, 100, 150, BiomeGrassland},
		{"Temperate Deciduous Forest", 100, 150, 150, BiomeTemperateDeciduousForest},
		{"Temperate Rain Forest", 100, 250, 150, BiomeTemperateRainForest},
		{"Subtropical Desert", 100, 50, 200, BiomeSubtropicalDesert},
		{"Shrubland", 100, 100, 200, BiomeShrubland},
		{"Tropical Seasonal Forest", 100, 150, 200, BiomeTropicalSeasonalForest},
		{"Tropical Rain Forest", 100, 250, 200, BiomeTropicalRainForest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineBiome(tt.elevation, tt.moisture, tt.temperature)
			if got != tt.expected {
				t.Errorf("DetermineBiome(%d, %d, %d) = %d; want %d", tt.elevation, tt.moisture, tt.temperature, got, tt.expected)
			}
		})
	}
}
