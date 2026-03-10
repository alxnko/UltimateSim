# Phase 8: 2D Visual Layer (Ebitengine)

_Objective: Establish the primary visual interface for the simulation data, translating abstract math and arrays into a playable Grand Strategy pixel environment._

## 8.1 Window Management & Camera

- **Ebitengine Integration:** Hook `github.com/hajimehoshi/ebiten/v2`.
- **Viewport Rules:** Establish window bounds and input capture mapping to camera vectors (Pan, Zoom).
- Camera math calculates arbitrary matrices translating the `MapGrid` bounds into specific pixel offsets on the active monitor resolution.

## 8.2 Sub-Tick Interpolation

- **The 144Hz Goal:** The ECS executes at an unyielding 60 TPS to maintain logic determinism.
- **`Render()` function architecture:**
  - Read `alpha` float variable produced by Phase 1's `TickManager` (representing time elapsed between ECS ticks).
  - Query `PositionComponent` and `VelocityComponent`.
  - Draw coordinates = `Position + (Velocity * alpha)`.
  - Results in buttery smooth 144Hz+ rendering without corrupting the 60 TPS data layer.

## 8.3 Map Rendering & Biomes

- **Grid Iteration:** Loop the static `MapGrid`.
- **Color Mapping:** Map `TileData.BiomeID` constants to static color arrays (e.g., Tundra = Hex `#FFFFFF`, Ocean = Hex `#0000FF`). Output to primary screen buffer.

## 8.4 Entity rendering

- **ECS Query:** Construct arche query targeting `PositionComponent`.
- Draw visually distinct sprite mappings via Ebitengine's `DrawImageOptions`:
  - Wandering AI Clusters as small dots.
  - `VillageEntity` as larger nodes.
  - `RuinComponent` entities as dark gray static indicators.

## 8.5 Visualizing Desire Paths

- **Dynamic Floor Updates:** Read the numerical `FootTraffic` value spanning `TileStateComponent` grids (Phase 2.5).
- If `FootTraffic > RenderThreshold`, override the base Biome color locally, drawing physical dirt/stone paths mapping historically heavy transit vectors over the pixel grid.
