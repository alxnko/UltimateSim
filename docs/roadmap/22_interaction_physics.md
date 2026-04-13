# Phase 22: The Interaction & Physics Engine

## Overview
A true immersive sim requires physics and precise interaction. The player must be able to select specific entities, collide with walls, and manipulate the environment rather than just floating over a grid.

## Architecture & DOD Implementation
- **`ColliderComponent`**: A new DOD struct defining Axis-Aligned Bounding Boxes (AABB).
  - `Width float32`, `Height float32`, `IsSolid bool`.
- **Kinematic Resolution**: Update `MovementSystem` (Phase 04) to evaluate `ColliderComponent` overlaps. Entities must slide against walls rather than getting stuck or phasing through them.
- **Interaction Raycasting**: 
  - Implement cursor-based entity selection. The game must cast a mathematical point from the screen coordinates to the world coordinates to identify the top-most entity under the mouse.
  - Implement forward-facing interaction ("Press E to Use"). The possessed entity casts a short vector in its facing direction to identify interactable `DoorComponent`, `NPC`, or `StorageComponent` entities.
- **Physicalized Items**: Items dropped on the ground must be represented as true ECS entities with `Position` and `ItemComponent`, rather than abstract numbers in a village's storage.