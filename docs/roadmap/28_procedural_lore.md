# Phase 28: Procedural Lore, Artifacts, & Myth-Making

## Overview
A medieval life simulation needs ancient history and terrifying unknowns. The procedural generation must extend to deep time, creating forgotten ruins, legendary beasts, and artifacts of power that players and ambitious NPCs seek out.

## Architecture & DOD Implementation
- **Deep Time Generation**: During world generation (Phase 02), the engine simulates 500 years of "headless" history before the player spawns, populating the world with ruined empires, buried ledgers, and established bloodlines.
- **`LegendaryComponent`**:
  - Attached to specific procedurally generated items (e.g., "The Ashen Blade of Kael").
  - These items passively emit `Aura` effects (e.g., increasing the wielder's `Charisma` or `Legitimacy` for ruling).
- **Procedural Lairs & Megabeasts**:
  - `EntityBeast`: High-stat, non-human entities that inhabit isolated map chunks (caves, deep forests). They hoard wealth and artifacts.
  - Eradicating a megabeast generates a massive positive `SparseHookGraph` connection with all nearby villages, instantly elevating the player (or an NPC) to hero/noble status.
- **Archaeology**: Players can use shovels or pickaxes to interact with `RuinComponent` tiles, unearthing buried `MaterialLedgers` that contain lost technologies or forgotten magic rituals.