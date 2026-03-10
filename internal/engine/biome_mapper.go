package engine

// Phase 02.3: Biome Mapping
// Objective: Map the generated layers into a uint8 BiomeID using Whittaker classification table function logic.

const (
	BiomeOcean uint8 = iota
	BiomeBeach
	BiomeScorched
	BiomeBare
	BiomeTundra
	BiomeSnow
	BiomeTemperateDesert
	BiomeShrubland
	BiomeGrassland
	BiomeTemperateDeciduousForest
	BiomeTemperateRainForest
	BiomeSubtropicalDesert
	BiomeTropicalSeasonalForest
	BiomeTropicalRainForest
	BiomeMountain // Optional, derived primarily from high elevation
)

// DetermineBiome maps elevation, moisture, and temperature (0-255) to a specific BiomeID.
// It uses a simplified Whittaker classification approach.
func DetermineBiome(elevation, moisture, temperature uint8) uint8 {
	// Water Level Threshold
	if elevation < 85 {
		return BiomeOcean
	}

	// Beach threshold
	if elevation < 95 {
		return BiomeBeach
	}

	// High elevation thresholds (Mountains/Snow)
	if elevation > 210 {
		if temperature < 100 {
			return BiomeSnow
		}
		if moisture < 100 {
			return BiomeBare
		}
		return BiomeMountain
	}

	// Tundra & Cold regions (Temperature < 85)
	if temperature < 85 {
		if moisture < 128 {
			return BiomeTundra
		}
		return BiomeSnow // Wet and cold
	}

	// Temperate regions (85 <= Temperature < 170)
	if temperature < 170 {
		if moisture < 85 {
			return BiomeTemperateDesert
		}
		if moisture < 140 {
			return BiomeGrassland
		}
		if moisture < 200 {
			return BiomeTemperateDeciduousForest
		}
		return BiomeTemperateRainForest
	}

	// Subtropical / Tropical regions (Temperature >= 170)
	if moisture < 85 {
		return BiomeSubtropicalDesert
	}
	if moisture < 140 {
		return BiomeShrubland // Or Savanna
	}
	if moisture < 200 {
		return BiomeTropicalSeasonalForest
	}
	return BiomeTropicalRainForest
}
