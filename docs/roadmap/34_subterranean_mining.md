# Phase 34: Subterranean Layers & Deep Mining (Z-Levels)

## Overview
A true Dwarf Fortress inspired simulation must go underground. The map cannot be strictly 2D. We must introduce Z-levels to allow for deep mining, underground civilizations, and structural collapse.

## Architecture & DOD Implementation
- **3D Chunk Architecture**: 
  - Transition the `MapGrid` from a flat 1D array to a 3D Chunk array `[Z][Y][X]TileData`.
  - Z=0 is the surface. Z=-1 through Z=-10 represent deep subterranean layers containing valuable ore veins, hidden caverns, and extreme dangers.
- **Mining & Excavation**: 
  - `JobMiner` entities use pickaxes to physically alter `TileStateComponent.IsSolid` from true to false, converting rock tiles into walkable floor tiles and spawning `ItemIron` or `ItemGold` entities.
- **Structural Integrity**: 
  - A dynamic cellular automata algorithm checks the stability of mined out areas. If too many subterranean tiles are excavated without leaving `PillarEntities` for support, a `CaveInEvent` is triggered, instantly crushing entities and destroying infrastructure above and below.
- **Subterranean Ecosystems**: 
  - Deep layers have zero light, requiring torches or lanterns (consuming physical fuel). They spawn unique blind fauna, glowing flora, and procedurally generated ancient horrors (Phase 28).