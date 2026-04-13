# Phase 43: Metallurgy, Alloys & Thermodynamics

## Overview
Crafting a sword is not just inputting iron. It is a process of heat, carbon, and physical chemistry.

## Architecture & DOD Implementation
- **`ThermodynamicComponent`**: Items and tiles track physical temperature. Forges must be fueled (Wood/Coal) to reach specific temperature bounds.
- **Alloy Crafting**: Combining Iron and Carbon (from burnt wood) at a specific heat threshold produces `Steel`, which dramatically alters the physical stats of the resulting weapon.
- **Quenching & Tempering**: Plunging a heated blade into water (Phase 39) rapidly cools it, locking in its structural integrity but risking a shatter if the temperature differential is too high.