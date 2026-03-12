# Boundless Sovereigns: Definitive Development Roadmap

This is the central index for the hyper-granular technical pipeline for the `arche-go` ECS engine. To ensure absolute data retention and clarity, each phase is broken out into its own detailed implementation file.

## The 13 Phases of the Total Simulation

- **[Phase 1: Initialization, Determinism, & ECS Bootstrapping](roadmap/01_foundation.md):** Repository layout, DOD constraints, fixed-tick ECS loop, tick-render decoupling, and Thread Pinning on Ryzen CPUs.
- **[Phase 2: Geography & Headless World Generation](roadmap/02_geography.md):** Perlin Biome mapping, Static resources, and infrastructure `TileStateComponent`.
- **[Phase 3: The Genesis Entities (Spawning & Genes)](roadmap/03_genesis.md):** Base structs (`Position`, `Velocity`, `Identity`), `GeneticsComponent`, `NeedsComponent`, and the `MetabolismSystem`.
- **[Phase 4: Autonomous Nodes (HPA\* & Migration)](roadmap/04_autonomous_nodes.md):** Async Path Queues, `pkg/math` Hierarchical Pathfinding, and `WanderSystem` AI state extraction.
- **[Phase 5: Settlement Birth, Genetics, & Ruins](roadmap/05_settlements_ruins.md):** Conversions to `VillageEntity`, `BirthSystem` inheritance, `RuinComponent`, and `arche-go` system filters.
- **[Phase 6: The Social Graph & Sparse Hooks](roadmap/06_social_graph.md):** `AffiliationComponent`, `MemoryComponent`, and the multi-gigabyte RAM saver: `SparseHookGraph`.
- **[Phase 7: The Cognitive Engine (Language & Memetics)](roadmap/07_cognitive_engine.md):** Interned `SecretRegistry`, `BeliefComponent` (Ideology Spread), `LinguisticDriftSystem`, and Translation Penalties.
- **[Phase 8: 2D Visual Layer (Ebitengine)](roadmap/08_visual_2d.md):** Ebiten hooks, sub-tick interpolation drawing, and Map rendering.
# Boundless Sovereigns: Definitive Development Roadmap

This is the central index for the hyper-granular technical pipeline for the `arche-go` ECS engine. To ensure absolute data retention and clarity, each phase is broken out into its own detailed implementation file.

## The 13 Phases of the Total Simulation

- **[Phase 1: Initialization, Determinism, & ECS Bootstrapping](roadmap/01_foundation.md):** Repository layout, DOD constraints, fixed-tick ECS loop, tick-render decoupling, and Thread Pinning on Ryzen CPUs.
- **[Phase 2: Geography & Headless World Generation](roadmap/02_geography.md):** Perlin Biome mapping, Static resources, and infrastructure `TileStateComponent`.
- **[Phase 3: The Genesis Entities (Spawning & Genes)](roadmap/03_genesis.md):** Base structs (`Position`, `Velocity`, `Identity`), `GeneticsComponent`, `NeedsComponent`, and the `MetabolismSystem`.
- **[Phase 4: Autonomous Nodes (HPA\* & Migration)](roadmap/04_autonomous_nodes.md):** Async Path Queues, `pkg/math` Hierarchical Pathfinding, and `WanderSystem` AI state extraction.
- **[Phase 5: Settlement Birth, Genetics, & Ruins](roadmap/05_settlements_ruins.md):** Conversions to `VillageEntity`, `BirthSystem` inheritance, `RuinComponent`, and `arche-go` system filters.
- **[Phase 6: The Social Graph & Sparse Hooks](roadmap/06_social_graph.md):** `AffiliationComponent`, `MemoryComponent`, and the multi-gigabyte RAM saver: `SparseHookGraph`.
- **[Phase 7: The Cognitive Engine (Language & Memetics)](roadmap/07_cognitive_engine.md):** Interned `SecretRegistry`, `BeliefComponent` (Ideology Spread), `LinguisticDriftSystem`, and Translation Penalties.
- **[Phase 8: 2D Visual Layer (Ebitengine)](roadmap/08_visual_2d.md):** Ebiten hooks, sub-tick interpolation drawing, and Map rendering.
- **[Phase 9: Logistics, Infrastructure, & Artifacts](roadmap/09_logistics_artifacts.md):** `CaravanEntity` spawning, `DecaySystem`, Desire Paths (`FootTraffic`), and `LegendComponent` legacy spawns.
- **[Phase 10: State Failure & Frictional Limits](roadmap/10_state_failure.md):** `LoanContractComponent` default logic, `AdministrativeDecaySystem`, and `DiseaseEntity` lethality grids.
- **[Phase 11: Possession Mechanics (Orchestrator)](roadmap/11_possession.md):** **The Graphics Orchestrator**, Ebiten/Raylib context switching, and Instanced 3D Rendering of village meshes.
- **[Phase 12: Network Delta Sync & Multiplayer](roadmap/12_multiplayer.md):** Deterministic state predictions, UDP payload parsing, and sparse update transfers.
- **[Phase 13: Stability, Tooling, & Balance Loops](roadmap/13_stability.md):** `MarketComponent` local price discovery, `CareerChangeSystem`, Jealousy metrics, `WinterPulse`, and `go-sqlite3` saves.
- **[Phase 14: True Individual NPCs & Dynamic Villages](roadmap/14_individual_npcs.md):** Migrating from abstracted Clusters to individual `NPC` entities, adding `FamilyID/ClanID` logic, and implementing dynamic Village hubs.
- **[Phase 15: Economic Agency, Businesses, & Currencies](roadmap/15_economic_agency.md):** NPCs starting businesses, employment/wage systems, localized currencies, and physical workplace requirements.
- **[Phase 16: Geopolitical Sovereignty & Unions](roadmap/16_geopolitical_unions.md):** Countries, diplomatic unions (war/economic/monetary), shared currencies, and profit-driven unification logic.

## The Expansion Slots (Infinite ECS Extensibility)

Because the architecture relies on decoupled, data-driven "Lego" pieces, these systems will be slotted in passively once the core engine achieves stability.

- **[Phase 17: Maritime Reach & Naval Logistics](roadmap/17_maritime_trade.md):** Ships, ocean-specific HPA\* grid routing, and maritime piracy limits.
- **[Phase 18: The Justice Engine & Legal Logic](roadmap/18_justice_engine.md):** `JurisdictionComponent`, contraband evaluation arrays, and active guard/punishment systems.
- **[Phase 19: Advanced Biology & Ecology](roadmap/19_advanced_biology.md):** `GenomeComponent` recessive traits mapping, inbreeding math, and macro-climate drift.
- **[Phase 20: Esoteric Systems & Ideological Apex](roadmap/20_esoteric_systems.md):** Expanding `BeliefID` spreading, triggering holy wars, and abstract numerical "Magic" physics utilizing map `ManaComponent` arrays.
