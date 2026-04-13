# Phase 21: The Micro-Scale Visual Overhaul (Tilesets & Sprites)

## Overview
To transition from a purely macro-scale geopolitical simulator into a "Streets of Rogue" or "Rimworld" style Action-RPG, the visual representation must evolve from abstract colored squares to a recognizable, immersive 2D environment. The player must be able to zoom in and clearly identify individual NPCs, terrain types, and items.

## Architecture & DOD Implementation
- **Sprite Rendering Engine**: Implement a robust texture atlas loading system in `internal/render/sprite_manager.go`. 
- **`RenderComponent`**: Add a new 16-byte DOD struct to the ECS containing a `TextureID` (uint32), `Layer` (uint8 for Y-sorting / Z-depth), and `Direction` (uint8).
- **Y-Sorting (Z-Depth)**: Update the Ebitengine draw loop in `internal/ui/state_playing.go` to sort entities by their `Y` position before drawing, ensuring entities properly overlap each other in 2D space.
- **Animation System**: Implement a lightweight state machine for animations (Idle, Walk, Attack) tied directly to the ECS `Velocity` and `Action` components, without polluting the core logic loop.
- **Camera Zoom & Culling**: Refine the camera logic to smoothly interpolate zoom levels. Only render entities within the scaled viewport (Frustum Culling) to maintain 60+ TPS even with complex sprite batches.