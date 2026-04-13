# Phase 40: Advanced Agriculture & Soil Mechanics

## Overview
Farming must be a science. It grounds the macro-economy in the physical reality of the dirt.

## Architecture & DOD Implementation
- **`SoilComponent`**: Tracks `Nitrogen`, `Phosphorus`, and `Moisture`.
- **Crop Rotation & Depletion**: Planting wheat continuously drains nitrogen, eventually causing the soil to become barren. NPCs must dynamically switch to planting legumes to restore nutrients or use `Fertilizer` (crafted from animal husbandry byproducts).
- **Irrigation**: Linked to Phase 39 (Hydrology). Players must dig physical trenches from rivers to their farms to ensure `Moisture` stays within optimal bounds during droughts.