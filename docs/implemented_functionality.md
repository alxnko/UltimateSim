# Implemented Functionality

This document serves as the comprehensive and definitive index of all actually implemented packages, ECS Components, ECS Systems, and underlying logic within the Boundless Sovereigns simulation engine.

**Note to AI Agents:** This document must be kept completely up-to-date. Any time a new struct, system, or mechanic is created, modified, or identified as undocumented, it must be added here immediately.

---

## 5. Network (`internal/network`)

- **`Server` (`server.go`)**: The primary multi-protocol orchestration server.
  - Handles concurrent TCP (`handleTCPConnection`) for reliable transactions (Ledgers, SparseHookGraph).
  - Handles concurrent UDP (`listenUDP`) for high-frequency positional float array updates.
- **`DeltaExtractionSystem` (`systems/delta_extraction.go`)**: Queries the ECS for shifting entities and extracts payload array structs mapping only these fractional data modifications.
- **`ClientPredictionSystem` (`systems/client_prediction.go`)**: Evaluates queued `PositionDelta` payloads to smoothly interpolate client positions towards server authority, correcting unpredictabilities like player override states.

## 1. Engine (`internal/engine`)

The core deterministic systems powering the total simulation and managing the world state.

- **`MapGrid` (`map_grid.go`)**: Holds the 1D flat array representation of the world.
  - Types: `MapGrid`, `TileData`, `ResourceDepot`, `TileState`
- **`TickManager` (`tick_manager.go`)**: Fixed-tick loop capped at 60 TPS enforcing strictly ordered `SystemRunner` phases.
  - Types: `TickManager`, `SystemPhase`, `System` interface.
- **`RNG` (`rng.go`)**: The singleton cryptographically-secure random number generator (ChaCha8) that maintains absolute determinism across simulation instances.
- **`PathRequestQueue` (`path_queue.go`)**: A worker pool of persistent goroutines that handles heavy asynchronous HPA* tactical pathfinding to prevent blocking the ECS loop.
  - Types: `PathRequestQueue`, `PathRequest`, `PathResult`, `Vec2`
- **`SparseHookGraph` (`sparse_hook_graph.go`)**: A memory-optimized relational database storing social "Hooks" and obligations between 100,000+ entities without combinatorial RAM explosions.
- **`SecretRegistry` (`secret_registry.go`)**: An interning dictionary for cognitive concepts, languages, and shared historical knowledge, returning a global `uint32` reference ID to minimize memory.
- **`BiomeMapper` (`biome_mapper.go`)**: Translates noise values (elevation, moisture, temperature) into distinct biomes and determines their baseline movement costs.
- **`MapGenerator` (`map_generator.go`)**: Procedurally generates the `MapGrid` based on a seeded 2D Perlin noise state.

---

## 2. ECS Components (`internal/components`)

All entities in the engine are composed of strict, flat-memory data structs strictly adhering to Data-Oriented Design (DOD). Located in `basic.go` (and related test files).

- **`Position`**: `float32` (X, Y) physical location.
- **`Velocity`**: `float32` (X, Y) kinematic vector.
- **`Identity`**: Name, ID, BaseTraits, and Age.
- **`Genetics`**: Inheritable stats like Health, Strength, Intelligence.
- **`Needs`**: Biological requirements like Food, Water, Rest.
- **`Path`**: Array of upcoming coordinate nodes for asynchronous travel.
- **`Legacy`**: Heritage mapping and biological succession IDs.
- **`RuinComponent`**: Decay tracker for destroyed settlements.
- **`Possessed`**: Marker component bypassing AI for player-controlled entities.
- **`JobComponent`**: Flag identifying an NPC's role (e.g. Farmer, Lumberjack, Artisan) to handle labor bounds.
*(Add all other newly identified/created components here)*

---

## 3. ECS Systems (`internal/systems`)

Systems perform decoupled, stateless logic by iterating over matched entities.

### Birth, Spawning & Genetics
- **`FamilySpawnerSystem` (`spawner.go`)**: The Genesis routine. Scans valid biomes and deterministically spawns the initial 100 family clusters, initializing Needs, Genetics, and Identity.
- **`BirthSystem` (`birth.go`)**: Handles generational reproduction, genetic crossover, and the passing on of traits.

### Biology & Mortality
- **`MetabolismSystem` (`metabolism.go`)**: Continuously drains `Needs.Food` based on internal `Genetics.Health` values.
- **`DeathSystem` (`death.go`)**: Reaps entities when biological needs hit zero. Removes identity and spawns physical items/corpses via inheritance logic (`itemSpawnData`).
- **`DiseaseVectorSystem` (`disease_vector.go`)**: Tracks lethality and immune systems across spatial networks (`immuneData`).

### Movement & Autonomy
- **`WanderSystem` (`wander.go`)**: AI state evaluation. Triggers asynchronous pathfinding based on immediate `Needs` vs Map resources.
- **`MovementSystem` (`movement.go`)**: The core kinematics loop mapping `Velocity` vectors onto `Position` coordinates while traversing `Path` nodes.
- **`CaravanSpawnerSystem` (`caravan_spawner.go`)**: Initiates logistical trade routes across the map.

### Geography, Infrastructure & Entropy
- **`CareerChangeSystem` (`career_change.go`)**: Algorithmically rebalances the economy by downgrading Artisans to basic gatherers when regional market limits flag food or wood shortages.
- **`CityBinderSystem` (`city_binder.go`)**: Aggregates wandering NPCs into a structured physical settlement.
- **`SettlementRuleSystem` (`settlement_rule.go`)**: Handles town/city state mechanics.
- **`InfrastructureWearSystem` (`infrastructure_wear.go`)**: Processes organic decay of physical roads/desire paths due to usage or neglect.
- **`RuinTransformationSystem` (`ruin_transformation.go`)**: Flips dead settlements to a ruined state while preserving map geometry but altering functional logic.
- **`RustSystem` (`rust.go`)**: Entropy mapping applied to metal-based artifacts/equipment.
- **`SpoilageSystem` (`spoilage.go`)**: Entropy mapping applied to organic goods (food/grain).

### Society & Cognition
- **`GossipDistributionSystem` (`gossip_distribution.go`)**: The memetic/idea virus system where NPCs trade internal knowledge (`nodeData`).
- **`LanguageDriftSystem` (`language_drift.go`)**: Tracks physical isolation over time, incrementally turning dialects into foreign languages.
- **`DebtDefaultSystem` (`debt_default.go`)**: Executes legal logic when an entity fails to repay a ledger contract.

---

## 4. Mathematics & Pathfinding (`pkg/math`)

Raw numerical utilities strictly built for deterministic execution.

- **`HPA* (Hierarchical Pathfinding)` (`pkg/math/hpa/grid.go`)**:
  - `AbstractGrid`, `Cluster`, `Node`, `Gateway` types.
  - Used for macro-level chunk routing across an immense map.
- **`Perlin Noise` (`pkg/math/noise.go`)**:
  - A custom 2D Perlin noise implementation seeded deterministically via ChaCha8 for all terrain generation.

---


## Phase 14: True Individual NPCs & Dynamic Villages
- **Phase 14 - Individual Agents**: Shifted the primary atomic moving unit from the abstracted `FamilyCluster` tag to true individual `NPC` entities. Implemented `NPCSpawnerSystem` which spawns distinct family groups (`FamilyID`) containing individual actors rather than a single numerical group. Refactored `SettlementRuleSystem` so that when an `NPC` settles into a stationary `Village`, the `NPC` entity is explicitly retained and assigned the `Village`'s `CityID`, natively embedding them as physical residents within the dynamic hub rather than despawning them into an abstract array.

## Phase 13: Stability & Balance Loops
- **Phase 13.2 - Labor Rebalancing**: Implemented `CareerChangeSystem` and `JobComponent`. The ECS actively acts against simulation collapse (famines) by parsing the market boundaries established in Phase 13.1. When extreme Wood/Food prices trigger `MarketComponent`, the logic dynamically parses all active `JobComponent` values matching the city's ID (`Affiliation.CityID`) and immediately downgrades advanced processors (`JobArtisan`) back into base extraction jobs (`JobFarmer` or `JobLumberjack`) without nested loops.
- **Phase 13.1 - Market Logic**: `MarketComponent` maintains a tightly packed 16-byte DOD struct tracking float32 local prices for `Food`, `Wood`, `Stone`, and `Iron`. `PriceDiscoverySystem` sequentially iterates over all nodes calculating mathematical limits defining demand (derived from `PopulationComponent`) versus supply (derived from `StorageComponent`). These distinct bounds actively govern the generation of `CaravanEntity` rescues if `FoodPrice` dynamically crosses extreme float boundaries natively without requiring hardcoded nested loops.
