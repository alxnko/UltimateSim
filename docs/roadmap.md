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
- **[Phase 11: Possession Mechanics (Orchestrator)](roadmap/11_possession.md):** **The Graphics Orchestrator**, Ebiten/Raylib context switching, Instanced 3D Rendering of village meshes, and foundational Action-RPG mechanics (real-time movement, combat hitboxes, stealth).
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

## The Action-RPG & Immersive Sim Overhaul

To elevate the gameplay experience into an immersive, zoomable action-RPG reminiscent of *Streets of Rogue* or *Rimworld*, the engine is actively being refactored to support micro-level interactions.

- **[Phase 21: Visual Overhaul (Tilesets & Sprites)](roadmap/21_visual_overhaul.md):** Transition to true 2D sprite rendering. Texture atlases, Y-Sorting/Z-Depth, basic animations, and camera culling for granular zoom levels.
- **[Phase 22: Interaction & Physics Engine](roadmap/22_interaction_physics.md):** Physical environment constraints. AABB collision detection, sliding kinematics, cursor raycasting, and physicalized items.
- **[Phase 23: Immersive UI & UX](roadmap/23_immersive_ui.md):** Bridging simulation data to the player. Diegetic health bars, persistent Hotbars, Inventory systems, and contextual interaction menus linked to the `SparseHookGraph`.
- **[Phase 24: Micro-Architecture](roadmap/24_micro_architecture.md):** Procedural generation of interiors. Walkable buildings, locking doors, and furniture (chests, anvils, beds) for targeted interaction and theft.
- **[Phase 25: Combat, Stealth, & Chaos](roadmap/25_action_chaos.md):** The true Action-RPG layer. Melee swing arcs, projectiles, FOV shadowcasting, noise emission limits, and dynamic NPC reactions to systemic chaos.
- **[Phase 26: Physical Labor, Crafting & Contracts](roadmap/26_labor_crafting.md):** *Rimworld*-style micro-economy. Physical construction of blueprints, workbench-dependent crafting, and dynamic job contract boards where NPCs and players buy and sell physical labor.