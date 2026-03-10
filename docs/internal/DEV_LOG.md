# Developer Knowledge Base: Internal Activity Log

## Current Phase / Task
- **Phase 2: Geography & Headless World Generation** (from `docs/roadmap/02_geography.md`)
- *Completed Phase 1: Initialization, Determinism, & ECS Bootstrapping*

## Active Component IDs & Data Structures
*Note: All structs must follow strict flat memory rules for Data-Oriented Design (DOD) to ensure cache alignment.*
- Structs use integer IDs instead of pointers (e.g., `TargetID uint64`).
- Use `uint8` and `uint16` where possible to minimize memory overhead instead of `int`.

**Implemented Structures (`internal/components/basic.go`):**
- `Identity`: `ID uint64`
- `Position`: `X, Y float32`
- `Velocity`: `X, Y float32`

**Implemented Structures (`internal/engine/map_grid.go`):**
- `TileData`: `Elevation`, `Moisture`, `Temperature`, `BiomeID` (all `uint8`). Packed precisely into 4 bytes for optimal cache alignment.
- `MapGrid`: Contiguous 1D array slice `Tiles []TileData` masquerading as a 2D matrix.

**Design Decision Log (Phase 02):**
- **Data Types & CPU Cache**: Using a 1D contiguous array `[]TileData` rather than `[][]TileData`. This bypasses pointer indirection across multiple slices, packing millions of grid tiles into tightly sequential memory blocks. When iterating across the map generation loops, this approach maximizes the L1/L2 cache hit rate, preventing cache misses common with 2D sliced pointers in Go. The `uint8` limits memory to 3 bytes per tile perfectly aligned for cache-lines.
- **Phase 02.2: Procedural Generation Pipeline**: Implemented `GenerateMap` (`internal/engine/map_generator.go`) utilizing a custom deterministic `Perlin` noise generator (`pkg/math/noise.go`). The generation algorithm iterates sequentially over the `MapGrid.Tiles` 1D array, maintaining absolute L1/L2 cache locality and dodging memory fragmentation. By seeding `math/rand/v2`'s `ChaCha8` engine with distinct deterministic modifiers, Elevation, Moisture, and Temperature map layers are consistently reproducible across simulation instances while maximizing iteration speeds via DOD principles. Tested via End-to-End deterministic tests in `map_generator_test.go`.
- **Phase 02.3: Biome Mapping**: Implemented a simplified Whittaker classification table algorithm in `internal/engine/biome_mapper.go` (`DetermineBiome`). Integrated it directly into the `GenerateMap` sequential pipeline. Adding `BiomeID` to `TileData` brought its total size from 3 bytes to 4 bytes, creating perfect 32-bit alignment and boosting sequential L1/L2 Cache hit rates because the Go compiler no longer inserts hidden padding.

**Design Decision Log (Phase 01):**
- **Data Types & CPU Cache**: `float32` was deliberately chosen over Go's default `float64` for `Position` and `Velocity` to strictly adhere to Data-Oriented Design (DOD) constraints. A `float32` takes 4 bytes instead of 8, doubling the density of our flat arrays. This tightly packed memory ensures significantly higher L1/L2 cache hit rates when the ECS iterates sequentially over 100,000+ entities, guaranteeing our 60 TPS performance goal is met.

## Global RNG Seeding Strategy
- **Seed Methodology**: A single, global singleton seed handles all stochastic events (terrain generation, birth systems, plague spawns, weather phenomena) to maintain absolute determinism across all simulation components.
- **Implementation**: Utilizes Go's `math/rand/v2` with `ChaCha8` engine for deterministic pseudorandom number generation.
- **Thread Safety**: Implemented behind a `sync.Mutex` in `internal/engine/rng.go` to prevent race conditions during highly parallelized ECS operations while maintaining strict sequencing.

## Phase 01.3: ECS Core (arche-go) Setup
- Implemented TickManager and System interface to manage arche-go World with 60 TPS cap and alpha calculation for rendering.
- **Performance & Cache Locality**: We maintain flat memory arrays for all Entity ID queries using `arche-go`. `float32` vs `float64` is preferred to halve the byte size and double the L1/L2 cache hit rate during continuous loops inside Systems.
- **MovementSystem Implementation**: Created `/internal/systems/movement.go` mapping `Velocity` to `Position` continuously.
  - Traceability: `// Phase 01.3: ECS Core Setup - MovementSystem`
  - DOD Alignment: We iterate strictly over matching entities sequentially accessing flat `float32` memory for X/Y coordinates directly via Arche pointers to limit OS-level caching jumps. E2E Deterministic tested in `movement_test.go` ensuring absolute identically repeatable states.

## Phase 01.4 & 01.6: Hardware Affinity, Telemetry & Profiling
- Implemented the game entrypoint in `cmd/game/main.go`.
- Created two primary goroutines: one for the core 60 TPS `TickManager` simulation loop, and another for the rendering context.
- **DOD CPU Cache Protection**: Applied `runtime.LockOSThread()` to both the simulation and render goroutines. Pinning the threads to a specific OS thread prevents cache invalidation that happens when the Go runtime scheduler migrates a goroutine across multicore CPUs (e.g. Ryzen architectures). Maintaining CPU core locality guarantees our ECS tight-loops preserve L1/L2 data access performance.
- Launched a `net/http/pprof` endpoint on `localhost:6060` for continuous profiling of goroutines and memory allocation overhead. Added automated E2E testing to verify deterministic consistency and tick orchestration in `cmd/game/main_test.go`.
- Completed Phase 1.1 requirements by creating the `pkg/math` directory to prepare for HPA* and grid conversion algorithms.
- Completed Phase 1.6 Telemetry requirements by implementing a command-line readout in `TickManager` (`internal/engine/tick_manager.go`) that computes and outputs the average `Ticks Processing Time (ms)` every second. Added an End-to-End test (`internal/engine/tick_manager_telemetry_test.go`) to ensure this logging logic works as expected without disrupting simulation limits.
