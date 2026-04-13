# Phase 36: Siege Warfare, Artillery & Fortification

## Overview
When geopolitical wars (Phase 29) escalate, they shouldn't just be blobs of infantry hitting each other. Sieges should be protracted, brutal affairs involving starvation, artillery, and structural destruction.

## Architecture & DOD Implementation
- **Siege Engine Crafting**: 
  - Armies can pool resources to build massive `ArtilleryEntities` (Trebuchets, Catapults, Battering Rams) on the battlefield via `ConstructionContracts`.
- **Structural Destruction**: 
  - Projectiles launched by artillery evaluate collisions against `StructureComponent` walls and `DoorComponent` gates. High-damage impacts permanently destroy the structures, opening breaches for infantry.
- **Sappers & Undermining**: 
  - Utilizing the Z-Level system (Phase 34), attackers can assign `JobMiner` units to dig tunnels underneath city walls. If the structural integrity fails, the walls above collapse into the tunnel.
- **The Starvation Tactic**: 
  - An attacking army can encircle a city, physically blocking all `CaravanEntities` from entering. The simulation naturally starves the city out as their `StorageComponent.Food` dwindles to zero, bypassing combat entirely.