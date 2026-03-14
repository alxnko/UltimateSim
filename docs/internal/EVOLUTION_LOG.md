# Evolution Log

This file tracks autonomous additions to the total simulation that bridge gaps identified in the vision.

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
