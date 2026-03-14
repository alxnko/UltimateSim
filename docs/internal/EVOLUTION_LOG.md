# Evolution Log

This file tracks autonomous additions to the total simulation that bridge gaps identified in the vision.

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
