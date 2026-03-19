# Implemented Functionality

This document serves as the comprehensive and definitive index of all actually implemented packages, ECS Components, ECS Systems, and underlying logic within the Boundless Sovereigns simulation engine.

**Note to AI Agents:** This document must be kept completely up-to-date. Any time a new struct, system, or mechanic is created, modified, or identified as undocumented, it must be added here immediately.

---

## Phase 44: The Vassal Safety Valve Engine
- **Phase 44.1 - The Vassal Safety Valve**: Bridges Economy (Monopoly Wealth), Genetics/Traits (Jealousy), Information (Rumors), and Justice (Blood Feuds). Implemented within `VassalSafetyValveSystem` (`internal/systems/vassal_safety_valve.go`). The system periodically evaluates the wealth of all citizens per `ClanID` in a city. If a Monopolist Clan holds >50% of the city wealth (minimum 1000 total), the system identifies the wealthiest `RulerID` of the Monopolist Clan. It then iterates all jealous NPCs (`TraitJealous`) in the city belonging to rival clans. These jealous NPCs automatically generate a massive `-50` grudge hook against the Monopolist Ruler via `engine.SparseHookGraph`, natively triggering the `BloodFeudSystem` (Phase 23) which prompts organic assassinations. Additionally, the jealous NPCs concurrently generate a mutated negative `Secret` via `SecretRegistry` and append it to their `SecretComponent` to be memetically leaked to the population via `GossipDistributionSystem` (Phase 07), destroying the monopolist's legacy.

## Phase 42: The Tax Evasion Engine
- **Phase 42.1 - The Tax Evasion Engine**: Bridges Economy (Taxation), Jurisdiction (Corruption), and Justice (Hooks). Implemented within `TaxationSystem` (`internal/systems/taxation.go`). When a Village pays its 100-tick cyclic taxes to the Country Capital, it evaluates the local `LoyaltyComponent.Value` against the Capital's `JurisdictionComponent.Corruption`. If `Loyalty.Value < Corruption`, the Village actively evades the tax, halting the transfer of physical `Wealth`. Simultaneously, the system mathematically iterates over all `NPC`s residing in that village and logs a `-50` negative hook against the Capital's Ruler ID using `engine.SparseHookGraph`. This naturally spawns deep resentment loops, converting high-corruption regimes into state-sponsored Blood Feud targets.

## Phase 28: The Vassal Rebellion Engine
- **Phase 28.1 - The Vassal Rebellion Engine**: Implemented `VassalRebellionSystem` linking Economy (Inflation/Debasement), Sovereignty (Loyalty/Secession), and Justice (Blood Feuds). High debasement or extreme local food prices continuously drain a Village's `LoyaltyComponent`. When `LoyaltyComponent.Value` drops to 0, the village secedes from the Country (`Affiliation.CountryID = 0`). The system then iterates through the Village's citizens who are highly desperate (`DesperationComponent.Level >= 50`) and adds a massive negative hook (-100) via `engine.SparseHookGraph` against the Country Capital's ruler ID. This organically hooks into the `BloodFeudSystem` (Phase 23) causing a secessionist war.
## Phase 30: The Ideological Economy
- **Phase 30.1 - The Tithe Engine**: Implemented `TitheSystem` bridging `BeliefComponent` (Phase 20) with `Needs.Wealth` (Phase 15). The ECS system iteratively searches for `JobPreacher` entities mapping their active positions and dominant belief arrays to a flat cache. It then parses all devout NPCs possessing `Needs.Wealth`. If they match the `BeliefID` and are spatially overlapping the `Preacher` (distSq < 25.0), the system structurally drains 10% of their physical `Needs.Wealth` natively transferring it to the Preacher over 50-tick bounds. This bridges abstract Memetics recursively into tangible Famine/Desperation logic loops mathematically driving crime.


## Phase 25: The Social Legacy & Succession Engine
- **Phase 25.1 - Heir Inheritance**: Integrated directly into `DeathSystem` (Phase 03). When an NPC despawns via starvation or old age, the system pre-caches their `Prestige`, `InheritedDebt`, and entire array of `SparseHookGraph` incoming and outgoing edges into `heirData`. The system subsequently maps over all active living `NPC` entities to find a suitable match sharing the same `FamilyID` (Affiliation component). The heir instantly absorbs all debts, legacy prestige, and generational grudges (`AddHook`). The dying parent's physical existence is correctly cleaned up via `RemoveAllHooks`.

## Phase 23: The Blood Feud Engine
- **Phase 23.1 - Blood Feuds & Generational Hatred**: Implemented `BloodFeudSystem` (`internal/systems/blood_feud.go`) mapping the `SparseHookGraph` to physical violence. NPCs continuously parse nearby entities using cache-friendly flat arrays. If a deep negative hook (`<= -50`) is detected, the NPC logs `InteractionMurder` inside their `Memory` buffer and starves the victim to death instantly (`Needs.Food = 0`). The system then iterates all surrounding clan members of the victim and massively depreciates their hook scores towards the killer and the killer's clan members (`-100` and `-50`, respectively). This forces a "Butterfly Effect" where one grudge spirals into a continuous frontier war.

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
- **`CountryComponent`**: Higher-level tag struct (`StandardCurrencyID`) tracking macro-state parameters for Country capitals.
- **`WarTrackerComponent`**: Phase 29 tracking struct attached to Country Capitals defining the target and active status of geopolitical resource wars.
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
- **`TaxationSystem` (`taxation.go`)**: Siphons revenue linearly from sub-cities back to their macro-state Country Capital.
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



## Phase 29: Geopolitical Resource Wars
- **Phase 29.1 - Geopolitical Famine Invasions**: Bridges Macro-Economics directly into the Blood Feud and Justice engines. Implemented `WarTrackerComponent` and `ResourceWarSystem`. When a Capital entity flags a severe local famine (`MarketComponent.FoodPrice > 8.0`) and is within striking distance (`RadiusSquared <= 2500.0`) of a wealthy neighbor (`StorageComponent.Food >= 1000`), the system triggers an invasion. Using `engine.SparseHookGraph`, it structurally seeds a massive `-100` relationship grudge across every `NPC` in the invading country against every `NPC` in the defending country. This bridges macro-state logic perfectly into individual `BloodFeudSystem` executions without writing massive new pathfinding loops.
## Phase 22: The Corruption Engine
- **Phase 22.1 - Systemic Bribery & Fracture**: Deeply links Economy, Justice, and Sovereignty. Expanding `JurisdictionComponent` with a `Corruption` counter, `JusticeSystem` allows high-wealth criminals to bribe Guards, bypassing `CrimeMarker` punishment and directly incrementing `Corruption` on the local jurisdiction. The `AdministrativeFractureSystem` dynamically calculates effective physical distances modified by this `Corruption` scalar, natively forcing highly corrupt empires to shatter prematurely as the frictional decay overpowers their administrative grid bounds.

## Phase 21: Evolutionary System Integration
- **Phase 21.1 - DesperationSystem**: Links Phase 13 Economy with Phase 18 Justice. Starving NPCs lacking wealth relative to local `MarketComponent` prices build `DesperationComponent.Level`. Once it reaches critical thresholds, they bypass trade loops, forcibly steal from nearest `Village` storages, and log `InteractionTheft`. The `JusticeSystem` naturally parses these events and unleashes guards, causing a completely emergent system response combining Famine, Prices, Crime, and Banishment loops.

## Phase 18: The Justice Engine & Legal Logic
- **Phase 18.1 - Jurisdiction & Law Definitions**: Added `JurisdictionComponent` dictating squared radii bounds around Capital entities and tracking a bitmask of `IllegalActionIDs` (`InteractionAssault`, `InteractionTheft`). Refactored `MarketComponent` limits into a standalone `ContrabandComponent` struct tracking bits mapped to `ItemWood`, `ItemStone`, `ItemIron`, etc.
- **Phase 18.2 - Detection & The Guard System**: Added `JusticeSystem` (`internal/systems/justice.go`). The system utilizes arche-go queries to compare `MemoryEvent` buffers against local `JurisdictionComponent` constraints, tagging offenders with a `CrimeMarker`. Implemented a `JobGuard` target mapping where idle Guards pathfind directly towards entities bearing the `CrimeMarker`.
- **Phase 18.3 - Sentencing & Prisons**: Attached O(G*C) distance evaluations in `JusticeSystem`. When a Guard entity physically intercepts a `CrimeMarker` within a ~1.4 tile radius (`distSq < 2.0`), it executes punishment logic: extracting `Bounty` fines from the criminal's `Needs.Wealth`, and forcibly applying Banishment (stripping `Affiliation.CityID` and assigning fleeing vectors). Proven natively via end-to-end tests in `justice_test.go`.

## Phase 15: Economic Agency, Businesses, & Currencies
- **Phase 15.4 - Organic Inflation via Debasement**: Linked Economy and Country Policies via the Debasement of physical coins. Expanded `CurrencyComponent` and `CountryComponent` with a `Debasement` float32 rate. Modified `MintingSystem` to use proportionally less `StorageComponent.Iron` when minting Debased coins. Added `InflationSystem` to iteratively calculate the physical average of `CoinEntity.Debasement` across unique `Position` coordinates and forcibly multiply the overlapping `MarketComponent` local prices by `1.0 + Average Debasement`. This naturally bridges sovereign policy into localized street-level Famine/Desperation.
- **Phase 15.3 - Predatory Lending Engine**: Implemented `LendingSystem` connecting Economic Agency to State Failure (Debt). Wealthy NPCs (`Needs.Wealth >= 500`) actively seek out starving, desperate NPCs (`DesperationComponent.Level >= 20`). The system physically transfers wealth and mathematically assigns a `LoanContractComponent`. When the `DueTick` fires inside `DebtDefaultSystem`, bankrupt citizens are unilaterally dragged into indentured servitude by forcibly updating their `Affiliation.GuildID` to the creditor's asset class.

## Phase 14: True Individual NPCs & Dynamic Villages
- **Phase 14 - Individual Agents**: Shifted the primary atomic moving unit from the abstracted `FamilyCluster` tag to true individual `NPC` entities. Implemented `NPCSpawnerSystem` which spawns distinct family groups (`FamilyID`) containing individual actors rather than a single numerical group. Refactored `SettlementRuleSystem` so that when an `NPC` settles into a stationary `Village`, the `NPC` entity is explicitly retained and assigned the `Village`'s `CityID`, natively embedding them as physical residents within the dynamic hub rather than despawning them into an abstract array.

## Phase 13: Stability & Balance Loops
- **Phase 13.2 - Labor Rebalancing**: Implemented `CareerChangeSystem` and `JobComponent`. The ECS actively acts against simulation collapse (famines) by parsing the market boundaries established in Phase 13.1. When extreme Wood/Food prices trigger `MarketComponent`, the logic dynamically parses all active `JobComponent` values matching the city's ID (`Affiliation.CityID`) and immediately downgrades advanced processors (`JobArtisan`) back into base extraction jobs (`JobFarmer` or `JobLumberjack`) without nested loops.
- **Phase 13.1 - Market Logic**: `MarketComponent` maintains a tightly packed 16-byte DOD struct tracking float32 local prices for `Food`, `Wood`, `Stone`, and `Iron`. `PriceDiscoverySystem` sequentially iterates over all nodes calculating mathematical limits defining demand (derived from `PopulationComponent`) versus supply (derived from `StorageComponent`). These distinct bounds actively govern the generation of `CaravanEntity` rescues if `FoodPrice` dynamically crosses extreme float boundaries natively without requiring hardcoded nested loops.
- **Phase 16.4 - Administrative Reach & Friction**: Implemented `AdministrativeFractureSystem`. Calculates `Village` squared distance from associated `Capital` entities using DOD mapping techniques (pre-cached `map[CountryID]*components.Position`). If `MaxAdministrativeRange` is exceeded, the node unilaterally secedes, actively limiting the maximal reach of a sprawling Country/Union entity by stripping `Affiliation.CountryID = 0`.

## Phase 30: The Carceral State & Blackmail Engine
- **Phase 30.1 - Carceral Resentment**: Integrated `JusticeSystem` (Phase 18) directly with `BloodFeudSystem` (Phase 23) via the `engine.SparseHookGraph`. When a Guard catches and physically punishes (fines and banishes) a criminal, the criminal now instantly generates a deep `-50` hook against the enforcing Guard. This natively translates the economic friction of stealing (e.g. from starvation) into a generational clan war, as the banished criminal will subsequently try to murder the Guard who arrested them.
- **Phase 30.2 - Corruption Blackmail**: Further deepened the integration by linking `JusticeSystem` (Bribery) with the Information/Gossip pillar. When a wealthy criminal successfully bribes a Guard, the criminal gains a `+50` hook (leverage) over the Guard. Concurrently, the engine accesses the `SecretRegistry` to intern a new rumor ("guard_ID_corrupted") and appends this new `Secret` struct directly into the criminal's `SecretComponent`, ensuring the corrupt act can be leaked and spread via the `GossipDistributionSystem`.

## Phase 40: The Maritime Migration Engine
- **Phase 40.1 - The Maritime Migration Engine**: Bridges Phase 21 (Desperation/Poverty) with Phase 17 (Naval Logistics). Implemented `MaritimeMigrationSystem` tracking active `ShipComponent` locations natively against high-desperation NPCs. Desperate NPCs with sufficient `Needs.Wealth` board ships (despawning from the local physical map grid) and algorithmically transfer wealth for passage. This dynamically acts as a global safety valve, mathematically rebalancing `PopulationComponent` arrays during localized Famines or extreme `PriceDiscovery` spikes by allowing the population to migrate trans-oceanically without explicit instructions.

## Phase 39: The Courier Interception Engine
- **Phase 39.1 - The Courier Interception Engine**: Bridges Phase 10 (Administrative Entropy) with Phase 18 (Justice). Implemented `CourierInterceptionSystem` evaluating active `OrderEntity` bounding boxes. When a `JobBandit` is close to the traversing state order (`distSq <= 2.0`), it forcibly intercepts and destroys the order, acquiring wealth from the state secrets and instantly logging an `InteractionTheft`. The `JusticeSystem` inherently parses this crime and dispatches `JobGuard` executioners, natively spiraling frontier banditry into massive state-sponsored enforcement sweeps and delayed bureaucratic failure.

## Phase 38: Ecological Pressure
- **Phase 38.1 - The Exposure Engine**: Connects Geography directly to Biological survival and the Economy. Implemented `ExposureSystem` evaluating map grid coordinate `Temperature` values. When an NPC is caught in extreme climates (`> 200` heat or `< 50` cold), the system iterates their `VitalsComponent.Pain` loop. The engine abstracts shelter via `Needs.Safety` where NPCs with high safety are shielded from the temperature damage. This bridges dynamic weather events (like Phase 20.2 Magic Spikes) directly into biological failures and ensuing Succession/Feud generation.

## Phase 37: The Quarantine Engine
- **Phase 37.1 - The Quarantine Engine**: Bridges Phase 10 (Biology/Disease) with Phase 18 (Justice). Implemented `QuarantineComponent` and `QuarantineSystem`. The system periodically evaluates active `DiseaseEntity` bounds. If a disease spawns within a `JurisdictionComponent` radius, the Jurisdiction enacts a `QuarantineComponent`. The `JusticeSystem` naturally parses this state, dynamically classifying any NPC or Caravan attempting to traverse across the jurisdiction border as a criminal. This natively bridges biological entropy into localized logistical collapse (Famine) and systemic Blood Feuds as Guards violently enforce the lockdown on starving citizens.

## Phase 36: The Scapegoat & Witch Hunt Engine
- **Phase 36.1 - Systemic Scapegoating**: Bridges Phase 31 (Disasters/Trauma) directly to Phase 18 (Justice) and Phase 23 (Blood Feuds). Implemented `ScapegoatComponent` and `ScapegoatSystem`. When a `Jurisdiction` experiences extreme `Trauma` (>= 15), the system maps local `BeliefComponent` frequencies. It algorithmically selects a minority belief (<30%) to blame. This immediately relieves state Trauma by 10 points (Catharsis) but activates the Scapegoat marker. The `JusticeSystem` inherently flags any NPC holding this minority `BeliefID` within the radius as a criminal, initiating state-sponsored fines, banishment, and massive `-50` Blood Feud grudges against the enforcing Guards.

## Phase 34: The Information Broker Engine
- **Phase 34.1 - Information Trade System**: Treats information as a tangible commodity in the ECS. Implemented `InformationTradeSystem` in `internal/systems/information_trade.go`. Opportunistic or impoverished NPCs holding high-value `Secrets` actively seek out wealthy neighbors (`Needs.Wealth > 10.0`) in close proximity. Upon discovering a novel secret, the wealthy neighbor exchanges physical wealth based on the secret's `Virality` multiplier, acquiring the knowledge while concurrently logging mutual positive `SparseHookGraph` connections. This establishes a structural bridge between the Memetic/Information pillar and the local Economic simulation.

## Phase 41: The Ostracization Engine
- **Phase 41.1 - The Ostracization Engine**: Implemented `OstracizationSystem` to provide a psychological bridge where NPCs actively use their memories of victimization to drive economic or social isolation. The system periodically scans `Memory.Events` for unpunished `InteractionTheft` and `InteractionAssault` events. It then generates deep negative hooks (`-20` per event) against the offender via `engine.SparseHookGraph`.
- **Phase 41.2 - Information Trade Embargo**: Modified `InformationTradeSystem` to check the `SparseHookGraph` prior to executing secret exchanges. If either party harbors a deep grudge (`<= -40`) against the other, the trade is blocked. This naturally isolates unpunished thieves and abusers from the information economy, tangibly punishing endless wealth acquisition through crime.

## Phase 40: The Ruins Resettlement Engine
- **Phase 40.2 - The Ruins Resettlement Engine**: Closes the loop on Phase 05.2 (Ruin Transformation). Implemented `RuinResettlementSystem`. When an NPC becomes homeless (`Affiliation.CityID == 0`), such as via `JusticeSystem` Banishment or a `NaturalDisasterSystem` wiping out their home, they naturally migrate. If they idle at the exact map coordinate of an abandoned `RuinComponent`, the system removes the `RuinComponent` and fully restores the `Village`, `Needs`, `MarketComponent`, and `StorageComponent` structures natively. The newly founded settlement receives foundational resource bonuses (simulating stone/wood salvaged from the ruins), and the NPC adopts the newly generated `CityID`, turning them from a banished criminal into a sovereign pioneer.
