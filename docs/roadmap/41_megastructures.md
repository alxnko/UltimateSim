# Phase 41: Megastructures & Z-Level Architecture

## Overview
Expanding Phase 24 (Micro-Architecture) upwards. Castles shouldn't just be flat boxes; they should be multi-story fortresses towering over the landscape.

## Architecture & DOD Implementation
- **Z-Level Pathfinding (Stairs & Ladders)**: `HPA*` is expanded to handle vertical transitions. `StructureComponent` now supports multiple layers of roofs and floors.
- **Megastructure Construction**: Massive generational projects (e.g., Cathedrals, Watchtowers) require thousands of `ConstructionContracts` and complex scaffolding logic.
- **Physics of Collapse**: If the ground floor of a watchtower is destroyed by artillery (Phase 36), the physics engine evaluates structural integrity, collapsing all Z-levels above it and crushing entities below.