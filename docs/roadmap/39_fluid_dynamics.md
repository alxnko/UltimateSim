# Phase 39: Fluid Dynamics & Hydrology

## Overview
Water is the lifeblood of civilization. To rival Dwarf Fortress, water cannot just be static blue tiles. It must flow, exert pressure, and be manipulated by physical infrastructure.

## Architecture & DOD Implementation
- **Volumetric Water Arrays**: Map tiles track a `FluidVolume` and `FlowDirection`. Water naturally equalizes across adjacent tiles and spills into Z-levels (Phase 34).
- **Engineering & Infrastructure**: Players and NPCs can build Dams, Aqueducts, and Watermills. Watermills convert physical river flow into rotational energy, powering adjacent workbenches (like grain mills or automated hammers).
- **Floods & Catastrophes**: Heavy rain (Phase 35) or breached dams can flood subterranean mines or low-lying villages, drowning entities and destroying organic goods.