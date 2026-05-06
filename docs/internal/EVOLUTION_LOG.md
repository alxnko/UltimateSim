# Boundless Sovereigns: Evolution Log

This log traces the systemic emergence features developed to bring the simulation closer to the "Total Simulation" goal.

## Phase 67: The Subterranean Mining Engine (Butterfly Effect)

**Date:** `2023-11-20` (or current iteration)

**The "Why":**
The Vision (`docs/vision.md`) dictates that "If a sword exists, someone mined the iron...". While we had Crafting (Phase 60) consuming Iron to create wealth, and Construction (Phase 59) consuming Stone to build infrastructure, there was a missing link: How are Iron and Stone extracted from the MapGrid? Before this phase, they just magically appeared or were abstracted away.

**The "What":**
Introduced `MiningSystem` and `MineComponent`.
NPCs with `JobMiner` now physically walk to a `MineComponent`, actively draining `Stamina` to extract `Iron` and `Stone` directly from the `engine.MapGrid` tile they are standing on.

**DOD Strategy:**
We utilize zero-allocation caching (`employersCache` and `minesCache`) initialized directly within the `MiningSystem` struct. These caches are simply cleared with `clear(s.employersCache)` and `s.minesCache[:0]` at the top of each tick. This prevents Go's garbage collector from thrashing during the high-frequency query loop.

**The "Butterfly Effect" Integrations:**
1. **Geography -> Economy:** Draining the physical map grid of `Iron` directly injects it into a specific employer's `StorageComponent`.
2. **Economy Macro-Link:** Upon extracting `Iron` or `Stone`, the system artificially deflates `MarketComponent.IronPrice` and `StonePrice` slightly, ensuring that increased raw material supply naturally stimulates the `CraftingSystem` and `ConstructionSystem`.
3. **Biology -> Psychology:** Mining drains `VitalsComponent.Stamina`. If `Stamina` reaches 0, the miner takes `Pain` instead, which seamlessly feeds into Phase 62's `MentalBreakSystem` to trigger `BreakBerserk` or `BreakCatatonic`.
4. **Entropy -> Death:** Deeper mines have an RNG-driven chance to trigger a Cave-in. This directly damages the `VitalsComponent.Blood`, organically looping into the `DeathSystem` (Phase 24) and `SanitationSystem` (Phase 65) if the miner perishes.
