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

// Phase 09.3: Infrastructure Wear System (Desire Paths)

// GetBaseMovementCost returns the initial movement cost for a specific biome.
// Costs > 1.0 slow down entities, while 1.0 is the baseline speed.
func GetBaseMovementCost(biomeID uint8) float32 {
	switch biomeID {
	case BiomeOcean:
		return 1.0 // Maritime travel is currently base cost (future phase)
	case BiomeBeach:
		return 1.5
	case BiomeScorched, BiomeBare:
		return 2.0
	case BiomeTundra, BiomeSnow:
		return 3.0
	case BiomeTemperateDesert, BiomeSubtropicalDesert:
		return 2.5
	case BiomeShrubland:
		return 1.8
	case BiomeGrassland:
		return 1.0
	case BiomeTemperateDeciduousForest, BiomeTropicalSeasonalForest:
		return 3.5
	case BiomeTemperateRainForest, BiomeTropicalRainForest:
		return 5.0
	case BiomeMountain:
		return 10.0
	default:
		return 1.0
	}
}

// GetEffectiveMovementCost calculates the current movement cost factoring in FootTraffic and Winter modifiers.
// As FootTraffic increases, the cost asymptotically approaches 1.0 (baseline flat road).
func GetEffectiveMovementCost(biomeID uint8, footTraffic uint32, isWinter bool) float32 {
	baseCost := GetBaseMovementCost(biomeID)

	if baseCost <= 1.0 {
		if isWinter {
			return baseCost * 1.5 // Phase 13.4: Statically inflate numerical costs
		}
		return baseCost // Already at maximum speed efficiency
	}

	// Each 1000 FootTraffic reduces the cost above 1.0 by half
	// To avoid math.Pow float64 conversions and maintain strict DOD speed,
	// we use a simple deterministic integer/float division reduction.

	// For example, if baseCost is 5.0 (diff 4.0), and footTraffic is 1000:
	// reduction factor = 1.0 + (1000 / 1000.0) = 2.0
	// new cost = 1.0 + (4.0 / 2.0) = 3.0

	diff := baseCost - 1.0
	trafficFactor := float32(footTraffic) / 1000.0

	effectiveCost := 1.0 + (diff / (1.0 + trafficFactor))

	if isWinter {
		effectiveCost *= 1.5 // Phase 13.4: Statically inflate numerical costs
	}

	return effectiveCost
}
