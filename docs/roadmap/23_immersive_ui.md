# Phase 23: The Immersive UI/UX & Inventory

## Overview
To make the game playable and intuitive, the player needs actionable feedback. Floating health bars, a clear inventory system, and diegetic pop-ups for dialogue or interaction are required to bridge the deep simulation with the player's immediate experience.

## Architecture & DOD Implementation
- **The UI Overlay System**: Separate the UI rendering pass from the world rendering pass in `EbitenApp`.
- **Diegetic UI (World-Space)**:
  - Floating damage numbers or status icons (e.g., a "!" over an NPC that has a rumor to trade).
  - Health/Stamina bars drawn slightly above the entity's Y-coordinate, rendered only if `Needs.Food` or `Vitals.Pain` are actively shifting.
- **Non-Diegetic UI (Screen-Space)**:
  - A persistent Hotbar for equipping weapons, tools, or consumable items.
  - A dedicated Inventory Screen mapping the possessed entity's internal item arrays to visual slots.
- **Contextual Interaction Menus**: When right-clicking an NPC, a radial or list menu should populate dynamically based on the target's ECS components (e.g., "Trade", "Extort", "Attack", "Bribe"), integrating natively with the `SparseHookGraph`.