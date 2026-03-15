# Evolution Log

## Evolution: Phase 18.3 / Phase 30.2 - Sentencing & Prisons (Fines & Wealth Transfer)
- **Goal:** Execute the "Systemic Emergence" objective by bridging the existing Justice Engine (Phase 18) and the Economic layers (Phase 15/16). Simulating how legal enforcement acts as a mechanism of wealth transfer, enriching the state while punishing criminals with poverty.
- **DOD Implementation:**
  - Expanded `adminJurisdictionData` cache inside `internal/systems/justice.go` to hold a pre-cached pointer to the jurisdiction's `TreasuryComponent`.
  - Avoided nested `arche-go` queries by using this O(1) pointer map during the O(G*C) Guard-vs-Criminal evaluation loop.
  - When a Guard successfully sentences a Criminal, the system calculates a `fine` based on the `CrimeMarker.Bounty`. The criminal's `Needs.Wealth` is drained, and the `collectedFine` is explicitly added to the pre-cached `TreasuryComponent.Wealth` belonging to the Guard's `CityID`.
- **The Butterfly Effect:**
  - Plugs deeply into Phase 15.3 (Predatory Lending), Phase 21 (Desperation), and Phase 26 (Banditry).
  - Starving NPCs steal to survive, triggering a `CrimeMarker`.
  - Guards catch them. Instead of just banishing them, the state actively confiscates their remaining physical wealth (`Needs.Wealth`) as a fine, enriching the Country's Treasury.
  - The State grows richer, enabling more infrastructure/armies, while the criminal becomes completely destitute (0.0 Wealth).
  - The newly impoverished and banished NPC, with skyrocketing `DesperationComponent.Level`, is forced to seek out Predatory Lenders (Phase 15.3) or turn to full-scale Banditry (Phase 26) on the frontiers.
  - Thus, strict law enforcement organically fuels peripheral crime and indentured servitude loops without hardcoded narratives.

## Evolution: Phase 30.1 - Ideological Economy (The Tithe Engine)
- **Goal:** Execute the "Systemic Emergence" objective by bridging the existing Memetic Engine (Phase 07/20) and Economic Engine (Phase 13/15). Simulating how religious devotion natively acts as a macroeconomic tax draining localized wealth.
- **DOD Implementation:**
  - Designed `TitheSystem` (`internal/systems/tithe.go`) adhering to `arche-go` ECS constraints.
  - The system iterates over `JobPreacher` entities, caching their positions and their primary `Belief` (highest weight) into a flat `[]preacherData` array to dodge nested queries.
  - Sequentially parses all devout NPCs (`Needs.Wealth > 0`) and iterates over the cached preachers to find spatial overlap (`distSq < 25.0`) with a matching primary belief.
- **The Butterfly Effect:**
  - Plugs deeply into Phase 21 (Desperation) and Phase 26 (Banditry).
  - A charismatic preacher spreads an ideology, converting a village.
  - The `TitheSystem` inherently activates, and devout citizens continuously transfer 10% of their physical `Needs.Wealth` to the preacher every 50 ticks.
  - Over time, this wealth extraction mathematically bankrupts the devout citizens.
  - Now impoverished, they cannot afford local `MarketComponent` food prices when winter hits. They starve, building `DesperationComponent.Level`.
  - The `DesperationSystem` naturally intercepts this poverty, converting the devout-but-starving citizens into `JobBandit` entities (Phase 26) who ravage local logistics. Religion organically and mathematically drives localized crime loops without scripted events.

This file tracks autonomous additions to the total simulation that bridge gaps identified in the vision.

## Evolution: Phase 29.1 - Geopolitical Resource Wars
- **Goal:** Execute the "Systemic Emergence" objective by bridging the existing Macro-Economics (Famine and Storage) and Sovereignty (Countries) directly into the Justice and Blood Feud engines. Simulating how starving nations orchestrate invasions against wealthy neighbors.
- **DOD Implementation:**
  - Added `WarTrackerComponent` strictly mapped to 8-byte bounds.
  - Implemented `ResourceWarSystem` (`internal/systems/resource_war.go`) adhering to `arche-go` ECS standards by extracting Country Capital metrics into a flat array (`[]capitalWarData`) to dodge nested queries.
  - Sequentially parses `MarketComponent.FoodPrice` to trigger the invasion condition (`> 8.0`) mapping to high `StorageComponent.Food` targets.
- **The Butterfly Effect:**
  - Plugs deeply into Phase 13.1 (Market Logic), Phase 16.1 (Countries), Phase 18 (Justice), and Phase 23 (Blood Feuds).
  - When a Capital experiences extreme famine, its local market `FoodPrice` inevitably skyrockets.
  - If a wealthy neighbor is within logistical striking distance (`RadiusSquared <= 2500.0`), the system unilaterally declares a "Resource War".
  - The `ResourceWarSystem` maps every `NPC` entity belonging to the starving country and uses the `SparseHookGraph` to inject a massive `-100` relationship hook against every `NPC` in the wealthy defending country.
  - This natively and instantly triggers the `BloodFeudSystem` at a massive, population-wide scale. The entire starving populace paths towards and murders the defending populace to take their resources, turning localized crime into a full-scale geopolitical war purely through data thresholds.
  - Verified completely via deterministic `go test ./internal/systems -run TestResourceWarSystem_Integration -count=2`.

## Evolution: Phase 15.3 - Predatory Lending Engine
- **Goal:** Execute the "Systemic Emergence" objective by bridging the existing Economic Agency (Businesses) and State Failure (Debt Default) layers. Simulating how wealthy individuals or guilds trap desperate/starving citizens in predatory debt loops.
- **DOD Implementation:**
  - Designed `LendingSystem` (`internal/systems/lending.go`) adhering to `arche-go` ECS constraints.
  - The system iterates over potential `Creditors` (Wealth >= 500.0) mapping their values to a flat cache `[]lendingNodeData` outside the nested query loops.
  - Sequentially parses `Debtors` (Wealth < 50.0 and `DesperationComponent.Level` >= 20) and mathematically evaluates spatial distances to wealthy Creditors.
- **The Butterfly Effect:**
  - Integrates seamlessly with Phase 21 (Desperation) and Phase 10.1 (Debt Default).
  - When an NPC reaches starvation bounds (`Desperation >= 20`), they are intercepted by a wealthy `Creditor` before they can resort to Phase 26 `Banditry`.
  - The `LendingSystem` physically transfers 100.0 Wealth to the Debtor and attaches a `LoanContractComponent` to them, effectively resetting their famine state but assigning a massive liability.
  - If the NPC cannot generate enough physical resources (`StorageComponent`) to repay the loan by the `DueTick`, the `DebtDefaultSystem` evaluates the breach of contract.
  - The default natively executes the transfer of the Debtor's `Affiliation.GuildID` to match the Creditor's `AssetID`, effectively chaining the bankrupt citizen to the Creditor's guild as an indentured servant, perfectly simulating the math limits of medieval debt structures.

## Evolution: Phase 20.3 - Traumatic Traditions (Ideological Xenophobia)
- **Goal:** Execute the "Systemic Emergence" objective by bridging Biological entropy with the Memetic engine ("Massive societal trauma (plague) causes algorithmic Traumatic Traditions (e.g., permanent Xenophobia)").
- **DOD Implementation:**
  - Expanded `JurisdictionComponent` in `internal/components/basic.go` to include a `Trauma uint16` counter, correctly adjusting tests to ensure perfect 4-byte padding bounds.
  - Modified `DeathSystem` to explicitly increment the `Trauma` value of any local `JurisdictionComponent` when an entity dies of starvation or plague (via `Needs.Food <= 0`).
  - Added `TraumaticTraditionsSystem` to automatically assign a `BeliefXenophobia` component (Weight: 100) to surviving NPCs in regions where the `Trauma` threshold is exceeded.
  - Added `XenophobiaSystem` to process entities with `BeliefXenophobia`. If they interact with someone of a different `LanguageID` (Foreigner), they instantly assign a `-100` negative hook against them using the `SparseHookGraph`.
- **The Butterfly Effect:**
  - When a Plague or Famine hits a jurisdiction, a massive die-off occurs.
  - The survivors are traumatized and become Xenophobic.
  - If a foreign trade Caravan or migrating wanderer enters the traumatized city, the Xenophobes instantly form a massive grudge against them.
  - This natively hooks into `BloodFeudSystem` (Phase 23.1), causing the traumatized citizens to organically murder the foreigners on sight, subsequently generating generational Clan feuds and triggering international Justice (Phase 18) interventions. Natively linking Biology, Epistemology, and Justice without hardcoded events.

## Evolution: Phase 23.1 - The Blood Feud Engine
- **Goal:** Execute the "Systemic Emergence" objective by implementing a completely new sub-system requested in the vision ("Blood Feuds: Unresolved murders before a state exists trigger Blood Feuds. The ECS memory ensures grandchildren of rival clans still possess deep Negative Hooks, leading to endless frontier violence.")
- **DOD Implementation:**
  - Expanded `internal/components/basic.go` with `InteractionMurder uint8 = 5`.
  - Created `BloodFeudSystem` (`internal/systems/blood_feud.go`) operating on a strict flat iteration over NPC entities to compute squared distances.
  - Used existing `engine.SparseHookGraph` to dynamically retrieve and spend deep negative hooks efficiently without introducing memory bottlenecks.
- **The Butterfly Effect:**
  - Deep grudges (Hook <= -50) automatically trigger murders.
  - Murder immediately causes generational hatred: all nearby clan members of the victim receive a massive negative hook (-100) against the killer, and a secondary negative hook (-50) against the killer's clan members.
  - This perfectly binds the Social Layer (Clans/Hooks) with Biology (Death) and Memory (InteractionMurder). Over time, as jurisdictions form (Phase 18 Justice), these generational murders are natively re-contextualized as Crimes, engaging guards and bounties emergently.

## Evolution: Phase 19.3 - Biological Entropy (Aging)
- **Goal:** Fulfill the "Biology: aging" requirement from the `vision.md` golden rules, adding an inevitable ceiling to population bloat.
- **DOD Implementation:**
  - Expanded `Identity` and `CitizenData` structs with an `Age uint16` field while preserving rigorous byte packing (`Identity` correctly padded to 32 bytes, `CitizenData` strictly padded to 20 bytes).
  - Added `AgingSystem` that executes every 360 ticks (1 "Year").
- **The Butterfly Effect:**
  - As NPCs pass 50 years of age, their genetic `Health` continuously deteriorates.
  - As NPCs pass 80 years of age, they face an exponentially increasing chance of "Sudden Death" (overriding `Needs.Food = 0`).
  - This hooks seamlessly into `DeathSystem` (Phase 03.3), which then spawns Legacy Items (Phase 09.5) and strips economic demand from `PriceDiscoverySystem` (Phase 13.1), resulting in massive integrated ripples.

## Evolution: Phase 22.1 - The Corruption Engine
- **Goal:** Execute the "Systemic Emergence" objective by implementing a completely new sub-system requested in the vision ("Kings rule via Legitimacy Scores; if a deadly secret is gossiped about the King, the standing army revolts... Contractual Law & Blackmail") while directly tying Economy, Justice, and Sovereignty together.
- **DOD Implementation:**
  - Expanded `JurisdictionComponent` to include `Corruption uint32` while keeping struct bounds small.
  - Intercepted logic in `JusticeSystem` (Phase 18) specifically when guards are punishing criminals.
  - Implemented dynamic cache mapping inside `AdministrativeFractureSystem` (Phase 16) to inject the active `Corruption` values natively without locking queries.
- **The Butterfly Effect:**
  - When famine hits, NPCs turn to theft (Phase 21 Desperation).
  - If a wealthy NPC commits a crime or is marked for justice, they now Bribe the guard natively (losing wealth, ignoring banishment).
  - This local bribe generates a single `Corruption` point on the Country's Capital.
  - Over time, high `Corruption` acts as a frictional multiplier in `AdministrativeFractureSystem` against distance, causing once perfectly stable, distant sub-cities to prematurely secede, fracturing sprawling empires purely via localized street-level bribery.

## Evolution: Phase 04.5 - The Epistemological Layer (Propaganda Erasure)
- **Goal:** Execute the "Systemic Emergence" objective by implementing a completely new sub-system requested in the vision ("Conquerors can enact Propaganda & Erasure by killing elders and burning ledgers, actively overriding the historical memory of the younger generation").
- **DOD Implementation:**
  - Expanded `JurisdictionComponent` to include `BannedSecretID uint32` while maintaining exact 16-byte alignment.
  - Added `LedgerComponent` (24-byte slice wrapper) and `Ledger` tag component to physicalize history into items.
  - Created `PropagandaSystem` which iterates in O(N^2) against `Jurisdiction` entities mapping young and old NPCs, triggering memory slice truncation for the youth and O(1) Needs starvation execution for the elders.
- **The Butterfly Effect:**
  - Plugs deeply into Phase 18 (Justice/Jurisdiction), Phase 19 (Aging), Phase 03 (Death), and Phase 07 (Information Leakage).
  - An administration can actively "ban" a secret in a radius. If an elder knows it, the state kills them. If a youth knows it, the state mind-wipes them. If a ledger (book) records it, the state burns it.
  - Verified 100% deterministic through `go test ./internal/systems -v -run TestPropagandaSystem_Integration -count=2` without locking Arche-Go internal queries dynamically.
## Evolution: Phase 15.4 - Organic Inflation via Debasement
- **Goal:** Execute the "Systemic Emergence" objective by implementing the missing Total Simulation mechanic: "If a King debases coinage (mixing lead with gold), NPCs detect it and cause organic Inflation."
- **DOD Implementation:**
  - Expanded `CurrencyComponent` and `CountryComponent` in `internal/components/basic.go` to include a `Debasement float32` field, adjusting DOD padding tests accordingly.
  - Modified `MintingSystem` (Phase 15.3) so that if a Capital entity possesses a `CountryComponent` with `Debasement` > 0, the Iron cost to mint physically drops, allowing faster minting but directly stamping the physical `CoinEntity` with the debasement rate.
  - Created `InflationSystem` (`internal/systems/inflation.go`) that iteratively evaluates all physical coins, aggregates their total debasement by specific `(X, Y)` grid coordinates, and unilaterally multipliers local `MarketComponent` prices by `1.0 + Average Debasement`.
- **The Butterfly Effect:**
  - A King decides to lower minting costs by debasing the currency (`CountryComponent.Debasement = 0.5`).
  - The `MintingSystem` begins churning out cheaper physical coins.
  - As these coins physically move across the map (e.g. via `Caravan` routing), `InflationSystem` detects their localized concentration in specific Villages.
  - The Village's local `MarketComponent` prices organically skyrocket.
  - This immediately hooks into Phase 21 (Desperation) and Phase 13.2 (Career Change), causing local starving NPCs to resort to crime and Blacksmiths to abandon their jobs, bridging Logistics and Sovereignty seamlessly.

## Evolution: Phase 24.1 - The Labor Union Engine
- **Goal:** Execute the "Systemic Emergence" objective by fulfilling the Vision's explicit requirement ("Plagues spread along trade routes causing massive labor shortages. Surviving peasants demand higher wages, spontaneously forming revolutionary Trade Unions") by connecting Biology, Economy, and Justice.
- **DOD Implementation:**
  - Expanded `MarketComponent` in `internal/components/basic.go` to include `WageRate float32`, correctly asserting 24-byte DOD packing constraints.
  - Added `StrikeMarker` component to specifically track unpaid labor grudges against target employers.
  - Modified `PriceDiscoverySystem` to deterministically calculate dynamic `WageRate` bounds scaling inversely with `Population.Count`, successfully integrating massive plagues directly into local macroeconomic costs.
  - Modified `JobMarketSystem` to assign `StrikeMarker` structurally outside the active ECS lock when `Business` treasuries fail to cover spiked wages. Modified hiring loops to strictly ignore marked strikers using `filter.Without()`.
  - Added `LaborUnionSystem` (`internal/systems/labor_union.go`) which extracts active strikes into a flat slice, tracking active jobs.
- **The Butterfly Effect:**
  - Plugs deeply into Phase 10 (Plagues), Phase 13 (Economy), Phase 15 (Employment), and Phase 23 (Blood Feuds).
  - When a Plague decimates a population, `WageRate` deterministically spikes.
  - Local businesses inevitably drain their `TreasuryComponent` and fail to pay wages.
  - Unpaid workers quit and structurally receive a `StrikeMarker`.
  - When a business replaces the striker with a new NPC (a "Scab"), the `LaborUnionSystem` mathematically hooks them together using `engine.SparseHookGraph`, injecting a massive `-50` relationship hook.
  - This natively triggers the `BloodFeudSystem` logic. The starving, unpaid Striker procedurally murders the Scab in the street. This action triggers `JusticeSystem` logic, bringing Guards into the conflict natively.

## Evolution: Phase 19.4 - Advanced Biology (Vitals)
- **Goal:** Execute the "Systemic Emergence" objective by implementing the requested Depth feature: "Replacing a simple Health component with Vitals including stamina, blood, and pain."
- **DOD Implementation:**
  - Added `VitalsComponent` in `internal/components/basic.go` with 4 `float32` fields (Stamina, Blood, Pain, Consciousness), strictly adhering to 16-byte bounds.
  - Verified sizes in `basic_test.go` using `unsafe.Sizeof` to ensure cache-line packing.
  - Modified `MovementSystem` to drain Stamina actively when moving, halve speeds on low Stamina, and completely intercept physical velocity if Consciousness <= 0.
  - Modified `MetabolismSystem` to detect starvation (`Food == 0`), applying massive Pain buildup, leading to an inevitable collapse of Consciousness.
- **The Butterfly Effect:**
  - Connects Physicality (Movement), Biology (Metabolism), and Economics (Food prices).
  - If food is expensive and an NPC starves, they experience massive Pain.
  - When Pain crosses 50.0, Consciousness rapidly drains.
  - Once unconscious, the `MovementSystem` intercepts and forces the velocity vector to 0.
  - The NPC is paralyzed, unable to trade or work, organically escalating their demise entirely through emergent systemic intersection rather than hardcoded event triggers. Verified completely via deterministic `go test ./internal/systems -run TestVitalsSystem_Integration`.

## Evolution: Phase 25.1 - Social Legacy & Succession Engine
- **Goal:** Execute the "Systemic Emergence" objective by implementing a missing mechanic from the Vision ("When the player character dies, they continue the legacy as a child, inheriting not just items, but Social Standing and Debts.").
- **DOD Implementation:**
  - Expanded `DeathSystem` (`internal/systems/death.go`) to inject `engine.SparseHookGraph` dependency.
  - Added `heirData` cache to `DeathSystem` struct to strictly avoid ECS structural query locking while accumulating data about dying NPCs.
  - Added new iterative utility methods `GetAllHooks`, `GetAllIncomingHooks`, and `RemoveAllHooks` directly to the `SparseHookGraph` instance (`internal/engine/sparse_hook_graph.go`).
- **The Butterfly Effect:**
  - When an NPC starves or dies of old age, their `Prestige`, `InheritedDebt`, and all `SparseHookGraph` positive/negative relationships are queried.
  - The system iterates over the entire `Map` looking for a suitable heir containing an identical `FamilyID`.
  - The matching heir (a child or kin) inherently receives all the outgoing and incoming grudges, and all financial debt, from the parent.
  - This profoundly binds Phase 23 (Blood Feuds) with Phase 03 (Death), ensuring that grudges outlive single entities and truly persist across generational divides, sparking endless frontier violence entirely driven by DOD logic.

## Evolution: Phase 26.1 - Caravan Banditry & Supply Chain Collapse
- **Goal:** Execute the "Systemic Emergence" objective by bridging the existing Logistics (Caravan Routes), Economy (Desperation/Needs), and Justice (CrimeMarker) layers. Simulating how starvation forces individuals to become bandits disrupting cross-map logistics.
- **DOD Implementation:**
  - Expanded `JobComponent` constants in `internal/components/basic.go` to include `JobBandit uint8 = 7`.
  - Implemented `BanditrySystem` in `internal/systems/banditry.go` strictly adhering to Arche-Go ECS standards.
  - The system pre-caches all `Caravan` entities and their physical `Payload` into a flat array (`cData`) outside the nested query loop for O(1) matching.
- **The Butterfly Effect:**
  - Integrates seamlessly with Phase 21 (Desperation). When an NPC reaches `Desperation.Level >= 50`, they are dynamically assigned `JobBandit`.
  - Bandits mathematically calculate squared distances to `Caravan` arrays traversing the HPA* nodes.
  - If a Caravan passes too closely (`distSq < 2.0`), the Bandit intercepts the node, completely draining the caravan's physical `Food` into their own `Needs`, resetting their starvation limits.
  - The system dynamically injects an `InteractionTheft` struct into the NPC's `Memory` buffer and attaches a `CrimeMarker` with a massive 250 Bounty.
  - The `Caravan` entity is structurally wiped from the ECS (`world.RemoveEntity`), meaning the targeted `Village` expecting those trade goods will inevitably suffer famine in the following ticks.
  - The `JusticeSystem` (Phase 18) instantly parses the new `CrimeMarker` and dispatches Guards to physically hunt the new Bandit.

## Evolution: Phase 28.1 - The Vassal Rebellion Engine
- **Goal:** Execute the "Systemic Emergence" objective by bridging the existing Logistics/Economy (Debasement and Famine), Sovereignty (Loyalty and Countries), and Justice (Blood Feuds) loops. Simulating how bad economic conditions organically lead to secessions and rebellions.
- **DOD Implementation:**
  - Implemented `VassalRebellionSystem` in `internal/systems/vassal_rebellion.go` adhering to `arche-go` ECS standards by preventing nested queries.
  - Used pre-allocated maps `capitalDataMap` and `secededVillages` to map Country logic outside of the iteration loops.
  - Iterates through all Villages sequentially, evaluating `LoyaltyComponent` drain based on `Debasement` and `FoodPrice`.
- **The Butterfly Effect:**
  - Integrates seamlessly with Phase 15.4 (Organic Inflation), Phase 21 (Desperation), and Phase 23.1 (The Blood Feud Engine).
  - A Capital enacts high debasement (`Debasement > 0.0`) or experiences famine (`FoodPrice > 5.0`).
  - Sub-cities suffer `LoyaltyComponent` drain. When it reaches 0, the village unilaterally secedes (`CountryID = 0`).
  - `DesperationComponent.Level >= 50` citizens within the seceding village immediately seed a `-100` relationship hook into the `SparseHookGraph` against the Capital's ruler ID.
  - This natively triggers the `BloodFeudSystem` which causes the citizens to form a rebellion force actively fighting the administration they used to belong to.
  - Verified 100% deterministic through `go test ./internal/systems -v -run TestVassalRebellion -count=2`.

## Evolution: Phase 27.1 - The Military Revolt Engine
- **Goal:** Execute the "Systemic Emergence" objective by implementing a missing mechanic from the Vision ("Kings rule via Legitimacy Scores; if a deadly secret is gossiped about the King, the standing army revolts.").
- **DOD Implementation:**
  - Designed `MilitaryRevoltSystem` (`internal/systems/military_revolt.go`) adhering to `arche-go` ECS standards by preventing nested queries.
  - Used pre-allocated slice `[]adminJurisdictionRevoltData` to map `JurisdictionComponent.BannedSecretID` per ticks.
  - Iterates flat slice of `militaryRevoltNodeData` checking for `JobGuard` status and `SecretComponent` overlaps.
- **The Butterfly Effect:**
  - Ties directly into Phase 07 (Information Leakage), Phase 18 (Justice/Jurisdiction), and Phase 23 (Blood Feuds).
  - A Capital actively tries to suppress a `BannedSecretID` (via `PropagandaSystem`).
  - However, if the `GossipDistributionSystem` successfully slips the secret past censors and a `JobGuard` NPC learns the truth, they revolt natively.
  - The Guard instantly drops their `JobGuard` status (switching to `JobBandit`) and severs `EmployerID`.
  - More destructively, it immediately seeds a `-100` relationship hook into the `SparseHookGraph` against the Capital's ruler ID, triggering the `BloodFeudSystem` which causes the former military force to actively murder the administration it used to protect.
  - Verified 100% deterministic through `go test ./internal/systems -v -run TestMilitaryRevoltSystem_Integration -count=2`.

## Phase 30: The Carceral State & Blackmail Engine (Integration)

**The "Why" (Gap):**
The Justice system (Guards punishing/fining/banishing Criminals) was acting as a flat loop—while bribes modified an abstract `Corruption` counter, there was no tangible social fallout or individual leverage created. Furthermore, the act of a guard banishing a criminal simply despawned them from the city without any personal consequence, leaving the `BloodFeudSystem` entirely disconnected from the justice/law pillar.

**The "What" (Innovation):**
Integrated the Justice pillar deeply into the Information (Secrets/Gossip) and Social Hierarchy (SparseHookGraph) systems.
1.  **Carceral Resentment:** When a criminal is punished (fined and banished) by a Guard, they naturally harbor deep hatred towards their punisher. The ECS now injects a severe `-50` hook against the enforcing Guard. This natively hooks into the `BloodFeudSystem` (Phase 23), meaning a starving NPC caught stealing bread will now attempt to murder the Guard who exiled them, sparking a clan-wide frontier war entirely emergently.
2.  **Corruption Blackmail:** When a wealthy criminal successfully bribes a Guard, the criminal gains blackmail leverage. The ECS injects a `+50` hook over the Guard and simultaneously generates a new `Secret` (e.g. "guard_2001_corrupted") into the `SecretRegistry`. This rumor is deposited directly into the criminal's `SecretComponent`, allowing the `GossipDistributionSystem` to spread the Guard's corruption dynamically across the city.

**DOD & Scalability Strategy:**
-   Passed the existing `SparseHookGraph` into `JusticeSystem` instead of inventing a new relational mapping system.
-   Reused the existing `SecretRegistry` to intern the rumor string, generating a highly cache-friendly `uint32` SecretID.
-   Maintained flat-array O(N) evaluations, ensuring no nested arche-go queries were required to evaluate social connections during the law enforcement phase.

## Evolution: Phase 31 - Systemic Entropy (Natural Disasters)

**The "Why" (Gap):**
The simulation lacked true massive entropic events outside of human or biological (plague) causes. The Vision document specifies that "Natural Disasters shift basic map parameters, forcing massive population resets." The world map was too stable.

**The "What" (Innovation):**
Implemented a fully integrated `NaturalDisasterSystem`. This deterministic algorithm spawns massive destructive events (e.g., Earthquakes) that physically alter the grid and trigger cascading failures.
1. **Map Modification:** The disaster zeroes out static resources (`WoodValue`, `StoneValue`) and destroys infrastructure (`FootTraffic`) across a massive radius.
2. **Biological Damage:** The disaster evaluates all living NPCs caught in the radius and spikes their `VitalsComponent.Pain`, potentially leading to paralysis or death.
3. **Economic Ruin:** Any `Village` caught in the blast radius has its `StorageComponent` wiped completely.

**The Butterfly Effect:**
This touches almost every system. A wiped `StorageComponent` causes immediate extreme famine, spiking local prices (`PriceDiscoverySystem`). Starving citizens reach critical desperation, becoming Bandits (`BanditrySystem`), which then attack Caravans. If an NPC dies from the disaster, the `Succession Engine` shifts their hooks and debts to their heirs. The loss of roads (`FootTraffic`) forces the HPA* pathfinding to calculate slower routes, further delaying rescue caravans.
