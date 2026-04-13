# Phase 25: Combat, Stealth, & Chaos (The Streets of Rogue Layer)

## Overview
The culmination of the Action-RPG transition. The deep economic and social systems must violently crash into real-time gameplay. Assassinations, riots, and stealth infiltrations must be fully systemic.

## Architecture & DOD Implementation
- **Hitbox Combat System**: 
  - `AttackComponent` tracking swing arcs, windup, and recovery frames.
  - Projectile entities with high velocity that evaluate collisions every sub-tick.
- **Stealth & Line of Sight (FOV)**:
  - Implement a shadowcasting FOV algorithm. Entities blocked by walls or behind the player's vision cone are not rendered.
  - `VisionComponent`: Defines an NPC's view radius. If the player commits a crime (e.g., hitting someone) outside of any Guard's FOV, no `CrimeMarker` is generated.
- **Noise Emission**: 
  - Sound waves are mapped as expanding radii. Breaking a door emits a noise radius of 15 tiles; stealthy lockpicking emits 2.
  - `HearingComponent` allows NPCs to investigate suspicious noises, abandoning their default `JobComponent` pathfinding temporarily.
- **Emergent Chaos**: When the player causes chaos (e.g., tossing a spoiled food item to make an NPC sick, or causing a massive explosion), the `WanderSystem` AI must dynamically flip to fleeing or combat states, forcing the macro-simulation to reconcile the sudden entropy.