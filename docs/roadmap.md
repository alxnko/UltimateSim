# Boundless Sovereigns: Development Roadmap

This document outlines the pure technical implementation roadmap for the Go-based engine. All logic is executed via the `arche-go` ECS.

## Phase 1: Engine Foundation & Headless Sim

**Goal:** Initialize the project, establish the ECS architecture, and prove performance scaling (100k entities at 60 TPS).

- **1.1 Repository Setup:**
  - `go mod init`
  - Establish standard Go layout (`/cmd/game`, `/internal/ecs`, `/pkg/math`).
  - Set up generic game loop (Tick, Update, Render).
- **1.2 ECS Core (`arche-go`):**
  - Initialize `arche.World`.
  - Implement `SystemRunner` pipeline.
  - Implement baseline Components: `Position`, `Velocity`, `Identity` (ID, Name, Traits).
- **1.3 Map Data Structure:**
  - Implement 2D array representing the world grid.
  - Tile structs: `BiomeType`, `Elevation`, `ResourcePool`.
- **1.4 AI: Spawning & Movement:**
  - Implement `SpawnerSystem` (Randomly disperse initial `FamilyCluster` entities).
  - Implement `NeedsComponent` (Food, Sleep, Safety).
  - Implement `WanderSystem` (A\* pathfinding towards optimal map tiles based on Needs).
- **1.5 Settlement State Machine:**
  - Implement `SettlementSystem`.
  - Logic: Convert `FamilyCluster` to `Village` entity if movement velocity == 0 and tile resources > threshold.
- **1.6 Benchmarking:**
  - Integrate `net/http/pprof`.
  - CLI output validating TPS with 100,000 active wandering entities.

## Phase 2: Relational Data & Memetics

**Goal:** Implement the complex network graphs for NPCs (Social linking, hooks, gossip).

- **2.1 The Social Graph:**
  - Implement `AffiliationComponent` (FamilyID, ClanID, GuildID, CityID).
  - Implement assignment systems to link nearby entities to the same CityID.
- **2.2 The Transactional Hook System:**
  - Implement `MemoryComponent` (Array of interactions).
  - Implement `HookMatrix`: a distinct system tracking global +/- integer scores between Entity IDs.
- **2.3 The Gossip Algorithm:**
  - Implement `SecretComponent` (attached to specific entities).
  - Implement `InformationLeakageSystem`: Calculate proximity radius, check `GossipTrait` modifier, duplicate `SecretComponent` to nearby entities.
- **2.4 Cultural Drift Systems:**
  - Implement `CultureComponent` (LanguageID, BeliefArray).
  - Implement `LinguisticDriftSystem`: Periodically mutate LanguageIDs of isolated settlements.
  - Implement `TranslationPenalty`: Drop/warp Secrets passed between different LanguageIDs.
- **2.5 Institutional Memory:**
  - Create dedicated `LedgerEntities` (abstract data storage).
  - Implement Propaganda events: Delete memory arrays in local NPC components.

## Phase 3: 2D Rendering & Physical Logistics

**Goal:** Transition from CLI to graphic rendering using Ebitengine. Visualize physical ECS data.

- **3.1 Ebitengine Integration:**
  - Integrate `github.com/hajimehoshi/ebiten/v2`.
  - Implement Camera (Pan/Zoom matrix math over the 2D array).
  - Render Map Data (Biome coloring) and Settlement Data (Entity sprites).
- **3.2 A\* Caravans & Logistics:**
  - Implement `CaravanEntity` (Position, PayloadComponent, Destination).
  - Implement `RoutingSystem`: Navigating the 2D array, avoiding mountains/water.
  - Render Caravan sprites moving on the map.
- **3.3 Spoilage & Heatmaps:**
  - Implement `DecaySystem`: Decrementing values in `InventoryComponent` over time.
  - Implement Heatmap rendering: Modify Map Data tile textures (create "Dirt Roads") where Caravan throughput is highest.

## Phase 4: Economy & State Feedback Loops

**Goal:** Implement the algorithmic governors (Debt, Inflation, Winter, State Collapse).

- **4.1 Currency & Inflation:**
  - Implement `TreasuryComponent`.
  - Implement Coinage IDs. Create `DebasementSystem` triggering percentage nerfs to purchasing power in local radii.
- **4.2 Debt Execution:**
  - Implement `LoanContractComponent`.
  - Logic: If Loan expires, ECS transfers assigned `Property` or `Hook` components from Debtor ID to Creditor ID.
- **4.3 Job Switching (Price Normalization):**
  - Implement `JobComponent`.
  - Implement `LaborShiftSystem`: Algorithms evaluate regional inventory arrays; switch JobComponent integers based on highest-value commodity.
- **4.4 Weather & Biology:**
  - Integrate global integer Time/Calendar.
  - Implement `WinterPulse`: Apply universal modifiers (higher metabolic burn, slower A\* movement).
  - Implement `DiseaseEntity`: Random spawn. Apply `SickMultiplier` to local area `NeedComponents`. Apply permanent `ImmuneTag` to survivors.
- **4.5 State Fracture:**
  - Implement `AdministrativeDecaySystem`. Calculate distance from Capital ID.
  - If Decay > `LegitimacyScore`, remove outer City IDs from Country ID array (Geopolitical fracture).

## Phase 5: The 3D Perspective (Raylib)

**Goal:** Implement the local possession camera, allowing player control of single entities in 3D.

- **5.1 Raylib Integration:**
  - Integrate `github.com/gen2brain/raylib-go/raylib`.
  - Implement context-switching framework to pause Ebitengine updates and invoke Raylib OpenGL context.
- **5.2 Procedural Mappings:**
  - Algorithm: Translate the abstract ECS `CityData` array into a 3D walkable bounding-box mesh.
- **5.3 Character Controller:**
  - Implement 3rd-person camera and WASD movement physics.
  - Bind player input to override the `WanderSystem` logic for the target Entity ID.
- **5.4 UI / Interaction Overlays:**
  - Implement Raygui dialogue panels.
  - Expose ECS `HookMatrix` and `MemoryComponent` data visually in the 3D dialogue options.
- **5.5 Succession Hook:**
  - Event logic: On player entity death, query `SocialStack` for Heir ID.
  - Re-bind 3rd-person controller to Heir ID. Carry over specific Data Components.

## Phase 6: Networking & Production

**Goal:** Multiplayer state sync, data persistence, and modding API.

- **6.1 Go UDP/TCP Server:**
  - Implement raw UDP/TCP socket server.
  - Implement Client connection management and lobby state.
- **6.2 ECS State Delta Sync:**
  - Build compression algorithms to output only modified ECS components (Position Deltas, Hook Deltas) per tick.
  - Write Client prediction routines.
- **6.3 SQLite Persistence:**
  - Implement `go-sqlite3`.
  - Write massive batch insert/update routines to dump `arche-go` arrays to disk.
- **6.4 Embedded Scripting (Modding):**
  - Integrate `go.starlark.net`.
  - Expose specific Go structs/functions to the Starlark runtime, allowing external overriding of AI weights and Map Generation logic.
- **6.5 Optimization & Release:**
  - Deep `pprof` analysis on networking and Raylib rendering overhead.
  - Cross-compile deployment pipelines (Windows, Linux, macOS).
