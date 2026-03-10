# Phase 2: Geography & Headless World Generation

_Objective: Map numerical data arrays that construct the physical terrain of the game world independent of rendering._

## 2.1 The Map Data Array

- **`MapGrid` Struct:** A contiguous 1D array masquerading via access functions as a 2D matrix (e.g., `Grid[y * width + x]`). Dramatically faster for cache-lines than `[][]Tile`.
- **`TileData` Struct:** A tightly packed integer stack holding `uint8` values:
  - `Elevation` (0-255)
  - `Moisture` (0-255)
  - `Temperature` (0-255)

## 2.2 Procedural Generation Pipeline

- **Noise Function:** Utilize `/pkg/math` algorithms (Simplex/Perlin). Feed the `GlobalRNG` seed as the deterministic offset.
- **Layer Generation:**
  1.  Generate Elevation Map.
  2.  Generate Moisture Map.
  3.  Generate Temperature Map (Latitude based + Noise modifier).

## 2.3 Biome Mapping

- **`BiomeMapper` Math:** Use a standard Whittaker classification table function logic (e.g., High Moisture + High Temp = Jungle; Low Moisture + Cold = Tundra). Maps the generated layers into a `uint8 BiomeID`.

## 2.4 Static Resource Depots

- **`ResourceDepot` Component/Struct:** Attach to specific Tiles or manage as a parallel grid array to reduce overhead.
- **Population Logic:**
  - Iterate grid: If `BiomeID` == Forest, set `WoodValue` randomly based on Seed.
  - If `BiomeID` == Mountain, set `StoneValue` and query secondary math for `IronValue`.

## 2.5 The Infrastructure Layer (`TileStateComponent`)

- **Desire Paths Base:** Implement an array or parallel component tracking `FootTraffic uint32`.
- Starts at 0. This variable will be hooked globally, so any entity executing a movement over this tile coordinate increments the integer. It forms the foundation for dynamic movement cost discounting algorithmized in Phase 9.
