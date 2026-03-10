package engine

import (
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/pkg/math"
)

// Phase 02.2: Procedural Generation Pipeline
// Objective: Populate the MapGrid with generated Elevation, Moisture, and Temperature data.

const (
	// Noise scale affects how "zoomed in" the noise map is.
	noiseScale = 0.05
)

// GenerateMap proceduraly generates the terrain data for the given MapGrid.
// It iterates sequentially over the 1D contiguous array `grid.Tiles` to maximize CPU L1/L2 cache hits.
// It uses the provided seed to initialize distinct deterministic Perlin noise generators for each layer.
func GenerateMap(grid *MapGrid, seed [32]byte) {
	// Initialize a local PRNG for generating resources deterministically
	localRNG := rand.New(rand.NewChaCha8(seed))

	// Initialize separate deterministic generators for each layer.
	// We modify the base seed slightly to ensure different noise patterns for Elevation, Moisture, etc.
	elevSeed := seed
	elevSeed[0] ^= 0x01
	elevNoise := math.NewPerlin(elevSeed)

	moistSeed := seed
	moistSeed[0] ^= 0x02
	moistNoise := math.NewPerlin(moistSeed)

	tempSeed := seed
	tempSeed[0] ^= 0x03
	tempNoise := math.NewPerlin(tempSeed)

	// Iterate over the 1D array masquerading as a 2D matrix.
	// Index calculation: i = y * width + x
	for y := 0; y < grid.Height; y++ {
		// Latitude calculation for temperature:
		// Let's say equator is at Height / 2.
		// Latitude goes from 0.0 (poles) to 1.0 (equator).
		distToEquator := float32(y) - float32(grid.Height)/2.0
		if distToEquator < 0 {
			distToEquator = -distToEquator
		}
		// Base temperature based on latitude
		latBaseTemp := 1.0 - (distToEquator / (float32(grid.Height) / 2.0))

		for x := 0; x < grid.Width; x++ {
			i := y*grid.Width + x

			nx := float32(x) * noiseScale
			ny := float32(y) * noiseScale

			// 1. Generate Elevation Map
			// Noise2D returns [-1.0, 1.0]. Normalize to [0, 255].
			eVal := elevNoise.Noise2D(nx, ny)
			grid.Tiles[i].Elevation = normalizeNoise(eVal)

			// 2. Generate Moisture Map
			mVal := moistNoise.Noise2D(nx, ny)
			grid.Tiles[i].Moisture = normalizeNoise(mVal)

			// 3. Generate Temperature Map (Latitude based + Noise modifier)
			tVal := tempNoise.Noise2D(nx, ny)
			// Combine base latitude temp with noise modifier
			// E.g., 70% latitude base, 30% noise modifier.
			combinedTemp := (latBaseTemp * 0.7) + (((tVal + 1.0) / 2.0) * 0.3)
			// Clamp to [0, 1] then scale to [0, 255]
			if combinedTemp < 0.0 {
				combinedTemp = 0.0
			} else if combinedTemp > 1.0 {
				combinedTemp = 1.0
			}
			grid.Tiles[i].Temperature = uint8(combinedTemp * 255.0)

			// 4. Determine Biome
			grid.Tiles[i].BiomeID = DetermineBiome(grid.Tiles[i].Elevation, grid.Tiles[i].Moisture, grid.Tiles[i].Temperature)

			// 5. Phase 02.4: Static Resource Depots
			// Initialize resources based on BiomeID to populate parallel array.
			// This preserves cache lines and handles deterministic RNG inside the loop.
			switch grid.Tiles[i].BiomeID {
			case BiomeTemperateDeciduousForest, BiomeTemperateRainForest, BiomeTropicalSeasonalForest, BiomeTropicalRainForest:
				// Base wood value (e.g., 50-150)
				grid.Resources[i].WoodValue = uint8(50 + localRNG.IntN(101))
			case BiomeMountain:
				// Base stone value (e.g., 100-255)
				grid.Resources[i].StoneValue = uint8(100 + localRNG.IntN(156))
				// Secondary roll for Iron
				if localRNG.IntN(100) < 30 { // 30% chance for Iron
					grid.Resources[i].IronValue = uint8(20 + localRNG.IntN(81)) // 20-100 Iron
				}
			}
		}
	}
}

// normalizeNoise maps a [-1.0, 1.0] float to a [0, 255] uint8.
func normalizeNoise(val float32) uint8 {
	// Map [-1.0, 1.0] to [0.0, 1.0]
	normalized := (val + 1.0) / 2.0
	// Clamp in case noise slightly exceeds bounds
	if normalized < 0.0 {
		normalized = 0.0
	} else if normalized > 1.0 {
		normalized = 1.0
	}
	return uint8(normalized * 255.0)
}
