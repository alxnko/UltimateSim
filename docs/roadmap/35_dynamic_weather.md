# Phase 35: Dynamic Weather, Seasons & Ecological Shifts

## Overview
The environment must be an active, hostile participant in the simulation. Weather and seasons should drastically alter logistics, combat, and agriculture, forcing civilizations to adapt or perish.

## Architecture & DOD Implementation
- **Dynamic Weather Fronts**: 
  - Implement a meteorological simulation layer overlaying the `MapGrid`. Pressure and moisture cells move across the map, generating Rain, Snow, Fog, and Storms.
- **Terrain Alteration**: 
  - Rain turns dirt paths into Mud, halving the `Velocity` of caravans and armies.
  - Winter snow physically accumulates, blocking mountain passes and freezing rivers solid (allowing new temporary pathfinding routes across the ice).
- **Agricultural Cycles**: 
  - Crops only grow during specific temperature windows. A late frost or prolonged drought (driven by the weather simulation) will destroy crops in the fields, triggering massive regional famines and subsequent Resource Wars (Phase 29).
- **Combat Implications**: 
  - Fog reduces the `VisionComponent` radius drastically, enabling stealth ambushes. Rain extinguishes torches and reduces the effectiveness of ranged projectile entities.