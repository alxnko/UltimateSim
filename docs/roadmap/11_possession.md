# Phase 11: The Possession Mechanics (The Graphics Orchestrator)

_Objective: Solve the aggressive Ebiten/Raylib runtime context clash to prevent OS GPU thread lock. Successfully bridge arbitrary 100k data nodes into an instanced 3D player perspective._

## 11.1 The Graphics Orchestrator Architecture

- **CRITICAL CONSTRAINT:** The `arche.World` pointer MUST remain static in the main pinned multi-core Goroutine.
- Render buffers (Ebitengine for 2D, Raylib-Go for 3D) act strictly as non-mutating functional "Observers."
- **The Switch Pattern:**
  1. If the player is in "Map Mode," pump the unrolled flat array coordinates to Ebiten.
  2. If the player clicks "Possess NPC," push an OS termination signal closing the `Ebitengine` window logic safely.
  3. Boot the `raylib-go` OpenGL context from the _same_ main executable, directly accessing the static ECS memory grid array without data serialization or copying.

## 11.2 Instanced 3D Control (raylib-go)

- **Avoiding VRAM Overflow:** Drawing 20,000 distinct village buildings will crash the renderer.
- **Instanced Rendering mapping:** Draw _one_ 3D generic "House Model", passing an array mapping positions across the `StorageComponent` bounds mathematically $N$ times.
- **3rd Person Controller:**
  - Bind WASD / Gamepad inputs directly updating the `VelocityComponent` delta of the target `Identity.ID`.
  - Override the standard `WanderSystem` AI state-processor for the Possessed target to ensure input cleanly controls the movement without system conflict jitter.
