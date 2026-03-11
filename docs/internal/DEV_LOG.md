# Developer Knowledge Base: Internal Activity Log

## Current Phase / Task
- **Phase 07.3: Linguistic Drift** (from `docs/roadmap/07_cognitive_engine.md`)
- *Completed Phase 07.2: Information Leakage (GossipDistributionSystem)*
- *Completed Phase 07.1: Secret Registry (String Interning)*
- *Completed Phase 5.4: Birth & Genetics Math*
- *Completed Phase 5.3: Arche-Go Component Filters*
- *Completed Phase 5.2: The Ruin Transformation*
- *Completed Phase 5.1: Settlement Conversion System*
- *Completed Phase 4.4: Resolving Kinematics*
- *Completed Phase 4.3: Trait & Need Driven Targeting*
- *Completed Phase 4.2: Async Path Queue Pool*
- *Completed Phase 4.1: Hierarchical Pathfinding (HPA\*) Implementation - Grid Abstractor*
- *Completed Phase 3: The Genesis Entities (Spawning & Genes)*
- *Completed Phase 2: Geography & Headless World Generation*
- *Completed Phase 1: Initialization, Determinism, & ECS Bootstrapping*

- **Phase 05.4: Birth & Genetics Math**: Implemented `BirthSystem` (`internal/systems/birth.go`) and expanded `PopulationComponent` with a dynamic `Citizens []CitizenData` array. `CitizenData` strictly follows DOD by embedding `Genetics` (four `uint8` fields) and `BaseTraits` (`uint32`). This creates a perfectly flat 8-byte structure, guaranteeing cache alignment and avoiding hidden compiler padding. The `BirthSystem` deterministically processes parent traits by sequentially iterating over arrays, maximizing CPU cache locality during the biological inheritance algorithms.

## Active Component IDs & Data Structures
*Note: All structs must follow strict flat memory rules for Data-Oriented Design (DOD) to ensure cache alignment.*
- Structs use integer IDs instead of pointers (e.g., `TargetID uint64`).
- Use `uint8` and `uint16` where possible to minimize memory overhead instead of `int`.

**Implemented Structures (`internal/components/basic.go`):**
- `Identity`: `ID uint64`, `Name string`, `BaseTraits uint32`
- `Genetics`: `Strength`, `Beauty`, `Health`, `Intellect` (all `uint8`)
- `Legacy`: `Prestige`, `InheritedDebt` (all `uint32`)
- `Needs`: `Food`, `Rest`, `Safety`, `Wealth` (all `float32`)
- `Position`: `X, Y float32`
- `Velocity`: `X, Y float32`
- `StorageComponent`: `Wood`, `Stone`, `Iron`, `Food` (all `uint32`)
- `PopulationComponent`: `Count uint32`
- `SettlementLogic`: `TicksAtZeroVelocity uint16`

**Implemented Structures (`internal/engine/map_grid.go`):**
- `TileData`: `Elevation`, `Moisture`, `Temperature`, `BiomeID` (all `uint8`). Packed precisely into 4 bytes for optimal cache alignment.
- `MapGrid`: Contiguous 1D array slice `Tiles []TileData` masquerading as a 2D matrix.

**Design Decision Log (Phase 02):**
- **Phase 02.4 & 02.5: Static Resource Depots and Infrastructure Layer**: Added `ResourceDepot` (`WoodValue`, `StoneValue`, `IronValue`) and `TileState` (`FootTraffic`) to `MapGrid`. Crucially, these were implemented as *parallel arrays* (`Resources []ResourceDepot`, `TileStates []TileState`) rather than inflating the `TileData` struct. This DOD design choice preserves `TileData`'s 4-byte size, ensuring maximum L1/L2 cache locality during the heaviest terrain generation loops. The resource generation logic inside `GenerateMap` uses a local deterministically seeded `ChaCha8` engine to map resources specifically to Forest and Mountain biomes securely. Verified memory layout with `unsafe.Sizeof` tests in `map_grid_test.go`.
- **Data Types & CPU Cache**: Using a 1D contiguous array `[]TileData` rather than `[][]TileData`. This bypasses pointer indirection across multiple slices, packing millions of grid tiles into tightly sequential memory blocks. When iterating across the map generation loops, this approach maximizes the L1/L2 cache hit rate, preventing cache misses common with 2D sliced pointers in Go. The `uint8` limits memory to 3 bytes per tile perfectly aligned for cache-lines.
- **Phase 02.2: Procedural Generation Pipeline**: Implemented `GenerateMap` (`internal/engine/map_generator.go`) utilizing a custom deterministic `Perlin` noise generator (`pkg/math/noise.go`). The generation algorithm iterates sequentially over the `MapGrid.Tiles` 1D array, maintaining absolute L1/L2 cache locality and dodging memory fragmentation. By seeding `math/rand/v2`'s `ChaCha8` engine with distinct deterministic modifiers, Elevation, Moisture, and Temperature map layers are consistently reproducible across simulation instances while maximizing iteration speeds via DOD principles. Tested via End-to-End deterministic tests in `map_generator_test.go`.
- **Phase 02.3: Biome Mapping**: Implemented a simplified Whittaker classification table algorithm in `internal/engine/biome_mapper.go` (`DetermineBiome`). Integrated it directly into the `GenerateMap` sequential pipeline. Adding `BiomeID` to `TileData` brought its total size from 3 bytes to 4 bytes, creating perfect 32-bit alignment and boosting sequential L1/L2 Cache hit rates because the Go compiler no longer inserts hidden padding.

**Design Decision Log (Phase 07):**
- **Phase 07.3: Linguistic Drift**: Implemented `CultureComponent` (`internal/components/basic.go`) and `LanguageDriftSystem` (`internal/systems/language_drift.go`). `CultureComponent` uses a single `LanguageID uint16` to strictly limit size overhead to 2 bytes, strictly following DOD principles and keeping RAM footprint flat. `LanguageDriftSystem` executes every 100 ticks for performance optimization, scanning the `Memory` ring buffer. It references entities via cached flat memory `Map` mappings to prevent `arche-go` API `world.Get` bottlenecks. If an entity misses parent `LanguageID` interactions for 10000 ticks, it diverges into a Dialect. Meanwhile, cross-language interactions accumulate in a localized `pidginTracker` structure; upon reaching 50000 hits, it fuses both identities into a new Pidgin language block, directly applying mathematical sociolinguistics without deeply nested graph arrays.
- **Phase 07.2: Information Leakage (GossipDistributionSystem)**: Implemented `GossipDistributionSystem` (`internal/systems/gossip_distribution.go`). It extracts valid gossiping entities (`Position`, `SecretComponent`, `Memory`, `Identity`) into a flat `[]nodeData` slice array. This DOD structural pattern preserves critical L1/L2 hits while running $O(N^2)$ distance and proximity calculations every 10 ticks, sidestepping nested `arche-go` iterator bottlenecks. Added `InteractionGossip` and enlarged `MemoryEvent.Value` from `int8` to `int32` to directly store the 32-bit `SecretID`, all while perfectly maintaining the 24-byte padding limit required for `MemoryEvent` alignment. Additionally, added `TraitGossip` as a bitmask modifier that doubles transmission odds.
- **Phase 07.1: Secret Registry (String Interning)**: Added `SecretRegistry` to avoid duplicate string allocations across thousands of entities sharing secrets/gossip. It runs as a singleton and provides thread-safe access using `sync.RWMutex`, assigning unique `uint32` IDs to mapped strings. Designed `Secret` component struct (`OriginID uint64`, `SecretID uint32`, `Virality uint8`) carefully verifying its packed DOD alignment constraints size mapping to exactly 16 bytes. Used standard slice header in `SecretComponent` tracking arrays of information dynamically for flexibility without bloating the base ECS archetype allocation.

**Design Decision Log (Phase 06):**
- **Phase 06.1: Societal Hierarchies (`AffiliationComponent`)**: Added `Affiliation` struct containing `ClanID`, `GuildID`, `CityID`, and `CountryID`. These are packed strictly as `uint32` values, totaling exactly 16 bytes for 16-byte CPU cache alignment. This avoids traversing any pointers for political mapping, ensuring immediate O(1) array-index lookup speed during ECS iterations. Created `CityBinderSystem` running every 10,000 ticks to calculate spatial radii and bind migrating `FamilyCluster` entities to the nearest active `Village`.
- **Phase 06.2: Interaction Telemetry (`MemoryComponent`)**: Implemented the `MemoryEvent` struct tightly packed into 24 bytes (on 64-bit systems) using `uint64` for TargetID and TickStamp, paired with minimal `uint8` and `int8` for Type and Value. The `Memory` struct utilizes a circular ring buffer (`[50]MemoryEvent` array) plus a `uint8` Head tracker, capping the footprint to 1208 bytes. This guarantees historical event logging remains constrained in memory and avoids dynamic slice allocations, thus eliminating GC spikes during intense NPC interactions.
- **Phase 06.3: The Sparse Hook Graph implementation**: Implemented `SparseHookGraph` (`internal/engine/sparse_hook_graph.go`) using `map[uint64]map[uint64]int` secured by `sync.RWMutex` to handle concurrent hook writes/reads safely without halting the 60 TPS simulation loop. This sparse mapping explicitly prevents a massive $10^{10}$ continuous integer allocation bomb that a standard 2D matrix ($100,000 \times 100,000$) would trigger, consuming RAM only when a relationship explicitly exists. Verified through robust threaded testing.

**Design Decision Log (Phase 05):**
- **Phase 05.1: Settlement Conversion System**: Implemented `SettlementRuleSystem` (`internal/systems/settlement_rule.go`) and the components `StorageComponent`, `PopulationComponent`, and `SettlementLogic` alongside `FamilyCluster` and `Village` tag components. When a `FamilyCluster` reaches 1000 consecutive ticks at 0 velocity on a resource-rich tile (`WoodValue + FoodValue > 50`), the system despawns the entity and replaces it with a new `Village` entity holding storage and population. `StorageComponent` is limited to `uint32` flat arrays holding base resources, maintaining perfect 16-byte alignment. `SettlementLogic` safely uses `uint16` to cap tracking ticks while halving the usual `int` byte cost. Entities despawning/spawning is structured outside the arche-go `query.Next()` loop using pre-allocated slices clearing via `toRemove[:0]` to guarantee zero Garbage Collection panic limits and maintain L1/L2 hits on iteration. Tested via Deterministic E2E suites verifying proper spawning limits.
- **Phase 05.2: The Ruin Transformation**: Implemented `RuinTransformationSystem` (`internal/systems/ruin_transformation.go`) and `RuinComponent`. When a settlement's `PopulationComponent.Count` drops to 0, instead of despawning the entity and erasing its history, the system removes the `PopulationComponent` and `NeedsComponent` and attaches the `RuinComponent`. This retains the former identity name and sets up a decay timer. `RuinComponent` utilizes a `uint32` for `Decay` to maintain DOD size limits (ensuring the struct fits within a strict 24-byte alignment, verified by `basic_test.go`).
- **Phase 05.3: Arche-Go Component Filters**: Updated the core iterations in `MetabolismSystem` and `DeathSystem` to explicitly use `ecs.All(...).Without(RuinComponentID)`. By ensuring these high-frequency system loops completely skip over ruined entities at the query builder level, we save tens of thousands of CPU cycles per frame that would otherwise be wasted trying to calculate metabolism or starvation logic on abandoned structures.

**Design Decision Log (Phase 04):**
- **Phase 04.4: Resolving Kinematics**: Upgraded `MovementSystem` (`internal/systems/movement.go`) to consume `Path` components actively provided by the Path Queue Pool. Logic relies strictly on sequential struct iterations via `arche-go`. Mathematical distance checks calculate if node waypoints are reached using `float32` arrays, and boundary mappings verify components do not map outside array bounds. This ensures all kinematic tracking aligns perfectly with DOD L1/L2 data access performance principles instead of relying on decoupled object logic. Added rigorous End-to-End deterministic testing verifying boundary validation.
- **Phase 04.3: Trait & Need Driven Targeting**: Built `WanderSystem` (`internal/systems/wander.go`) to evaluate `Needs.Food` and trigger asynchronous pathfinding requests dynamically. It uses pure sequential flat-memory iteration over `MapGrid.Resources`. To facilitate this cleanly without polluting cache lines, `FoodValue uint8` was added to `ResourceDepot` (`internal/engine/map_grid.go`), ensuring the struct perfectly aligns to 4 bytes (32-bit), optimizing L1/L2 hits while meeting feature requirements. Implemented RiskTaker and Cautious traits via bitmask logic on `Identity.BaseTraits`.
- **Phase 04.2: Async Path Queue Pool**: Created `PathRequestQueue` in `internal/engine/path_queue.go` to handle heavy HPA* computations without blocking the 60 TPS simulation loop. It employs a worker pool of dedicated persistent goroutines that receive `PathRequest` payloads via channels and return `PathResult` slices. Also introduced `PathComponent` (`internal/components/basic.go`) containing a `[]Position` array of upcoming path nodes, retaining `float32` flat-memory adherence for DOD iteration. Deterministic stability across multi-goroutine setups confirmed via `-count=2` testing.
- **Phase 04.1: Hierarchical Pathfinding (HPA\*) Implementation**: Implemented the macro-level structure for HPA* (`pkg/math/hpa/grid.go`). To strictly adhere to Data-Oriented Design (DOD) guidelines, `AbstractGrid` handles regions by packing them into a flat `[]Cluster` array rather than a 2D slice or pointers. Struct components use `uint16` to maintain extremely small byte footprints and cache locality. `Cluster.X` and `Cluster.Y` use standard `int` to cleanly map into the 1D arrays of `MapGrid`, while `Node` structures representing tactical paths retain tightly packed arrays using `float32` for pathfinding movement costs. Tested edge case grid distributions and enforced a "Deterministic Check" verification.

**Design Decision Log (Phase 03):**
- **Phase 03.1: Genesis Base Structs**: Expanded `basic.go` with `Identity`, `Genetics`, and `Legacy` components. Struct sizes enforce DOD flat memory limits. Verified 32-byte alignment for `Identity`, 4-byte for `Genetics`, and 8-byte for `Legacy` via `unsafe.Sizeof` unit tests.
- **Phase 03.2: The Genesis Spawner**: Implemented `FamilySpawnerSystem` in `internal/systems/spawner.go`. It queries `MapGrid` to extract all valid habitable locations (Biome != Ocean) into a flat slice array to reduce redundant map lookups. It deterministically places 100 `arche.Entity` families, initializing all baseline Components (`Needs`, `Genetics`, `Identity`, etc). To maintain absolute determinism across simulation instances, we utilize `engine.GetRandomInt()` recursively via the `GlobalRNG` engine for map coordinates and a unified sum technique for approximating the bell curve of the `Genetics` stats.
- **Phase 03.3: Metabolism & Death System**: Implemented `MetabolismSystem` (subtracts `Needs.Food` based on `Genetics.Health`) and `DeathSystem` (despawns entity if `Needs.Food <= 0`). Built E2E deterministic test suites. These iterate cleanly over arche-go arrays, maintaining `float32` variables to adhere to the Phase 1 DOD cache size constraints.

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
