# Phase 24: Micro-Architecture (Interiors & Urban Planning)

## Overview
For an immersive action-RPG to function, villages cannot just be abstract resource pools on a single tile. They must be physical layouts of walls, doors, floors, and furniture that the player can navigate, break into, or defend.

## Architecture & DOD Implementation
- **Procedural Urban Generation**: When a `VillageEntity` is born (Phase 05), the system triggers a localized cellular automata or BSP (Binary Space Partitioning) algorithm to generate actual building footprints on the micro-grid.
- **`StructureComponent`**: Differentiates standard map tiles from built architecture. Defines walls that block both movement (`ColliderComponent`) and vision.
- **`DoorComponent`**: A functional barrier that can be locked. Tied to `KeyID` or faction alignment. NPCs with the correct `AffiliationID` can pass freely; others must lockpick or break it down.
- **Furniture Entities**: Beds, Chests, Anvils. `StorageComponent` wealth is distributed physically into these containers rather than remaining an abstract integer on the Village entity. The player must physically walk to a chest to steal from the village.