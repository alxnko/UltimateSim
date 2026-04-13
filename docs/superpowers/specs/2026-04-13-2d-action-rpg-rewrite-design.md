# 2D Action-RPG Simulation Rewrite Design

## 1. High-Level Vision
Transition Boundless Sovereigns from a 3D/2D hybrid macro-strategy game into a purely 2D Action-RPG (similar to Streets of Rogue, Rimworld, or Dwarf Fortress Adventure Mode). The player possesses a single NPC, acting as a localized agent of chaos or an ordinary citizen, physically existing within the massively scaled, fully deterministic ECS simulation.

## 2. Architecture & Rendering (The 2D Unification)
- **Engine Swap:** Completely remove `raylib-go` from the project. All rendering logic must be consolidated into `Ebitengine`.
- **Camera Pipeline:** The 2D view must scale dynamically. Zoomed out, it displays the macro-geopolitical state (country borders, trade routes). Zoomed in, it maps 1:1 to street-level interactions, rendering directional sprites for NPCs and tile-based physical architecture (houses, items).
- **Time Dilation:** The ECS continues strictly at 60 TPS (Ticks Per Second) regardless of the camera zoom level. The player lives in real-time continuous flow alongside the simulation.

## 3. Player Input & Possession (Data-Oriented Control)
- **Input System:** Create `PlayerInputSystem` mapping WASD and mouse actions directly to Ebitengine inputs.
- **Possession Component:** The active player entity receives a `Possessed` component. 
- **AI Bypass:** The `WanderSystem` and other autonomous pathfinding systems naturally skip any entity possessing this component.
- **Physical Interactions:** Mouse clicks and movement structurally mutate the `Velocity` and log physical events (e.g., `InteractionAssault`, `InteractionTheft`, `WealthTransfer`) into the target's `Memory` buffer, inherently triggering the `JusticeSystem` or `SparseHookGraph` systems natively.

## 4. Emergent Progression (PlayerDirectorSystem)
- **The "Agent of Chaos" Loop:** A new `PlayerDirectorSystem` will continuously evaluate the local entropy surrounding the `Possessed` entity. 
- **Dynamic Events/Quests:** If a local `Jurisdiction` is highly corrupt, or a wealthy NPC harbors a deep `-50` hook against a rival, the director suggests an "Opportunity" (e.g., a hit contract or theft). 
- **Citizen to Ruler:** The player begins as a random NPC with a `JobNone` or `JobFarmer`. By exploiting the physical simulation loops (e.g., buying all local food during a shortage to induce a famine), the player can organically destabilize regions, leading to structural revolutions where they can assume leadership (updating their own `Affiliation.CityID` or `RulerID`).

## 5. System Preservation (Zero Abstraction)
- All 49+ existing mechanics (Blood Feuds, Taxation, Disease Vectors, Maritime Migration, Scapegoating) remain functionally identical.
- The player is bound by the exact same biological and mathematical constraints as NPCs. They can die of starvation (`Needs.Food`), be banished by Guards for stealing, or die in a plague. Upon death, the game seamlessly transitions possession to their genetic heir via the existing `DeathSystem` legacy logic.

## 6. Implementation Scope
- `Delete:` `internal/render/raylib_app.go`
- `Add/Modify:` `internal/render/ebiten_app.go` (implementing zoom & local sprite logic).
- `Add:` `internal/systems/player_input.go` (WASD & Mouse parsing).
- `Add:` `internal/systems/player_director.go` (dynamic quest hooks).
- `Modify:` `internal/systems/wander.go` (skip `Possessed` components).
- `Add:` `internal/components/possessed.go` (if not already fully mapped for this new scale).
