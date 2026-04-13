# Action-RPG Complete Rewrite Design (Approach 1)

## 1. High-Level Vision
Transform "Boundless Sovereigns" into a fully-fledged 2D Action-RPG. The game will shift from a raw simulation viewer into a single-character experience (similar to Streets of Rogue, RimWorld, or Dwarf Fortress Adventure Mode) with a dedicated UI, robust state machine, and twin-stick/mouse-aim controls.

## 2. Architecture & State Management (The "Approach 1" Model)
- **Game State Machine:** Implement a `GameState` interface and a `StateManager` to handle the core loop:
  - `StateMainMenu`: Options to start as a random citizen or create a custom character.
  - `StateCharacterCreation`: Allocate points to `Genetics` (Strength, Intellect), select a `Trait` (e.g., RiskTaker, Jealous), and pick a starting `Biome`/`CityID`.
  - `StatePlaying`: The core simulation. Wraps the `TickManager`, `MapGrid`, and `Camera`.
  - `StateGameOver`: Triggered on player death. Offers options: "Possess Heir" (if applicable), "Possess Random Citizen", or "Return to Main Menu".
- **Rendering Pipeline:** Split rendering into two distinct phases:
  1. `World Space`: Rendered via the camera, handling tiles, NPCs, and dynamic elements. Uses a virtual resolution scaled up for a crisp pixel-art aesthetic.
  2. `Screen Space`: Rendered directly to the screen for crisp, unscaled UI elements (Clean HUD).

## 3. Player Input & Possession (Twin-Stick/Mouse Aim)
- **Movement:** `PlayerInputSystem` processes WASD and updates the `Velocity` of the entity bearing the `Possessed` component.
- **Aiming:** The mouse cursor position translates to world coordinates. We calculate the angle from the player to the cursor to determine the facing direction.
- **Combat & Interactions:**
  - `Left-Click`: Spawns an `InteractionAssault` (melee or ranged, depending on the active equipment) in the direction of the cursor.
  - `Right-Click/E`: Spawns an `InteractionTalk` or `InteractionTrade` if an NPC or interactive object (like a door or merchant) is under the cursor.

## 4. UI & The "Clean HUD"
- **Vitals:** Display `Vitals.Blood` (Health), `Vitals.Stamina`, and `Needs.Food` in the corners of the screen using clean, minimal bars.
- **Contextual Prompts:** The `PlayerDirectorSystem` will push "Emergent Quests" (e.g., "Bounty: Local Criminal", "Grudge: Clan 4 needs a mercenary") to a small, non-intrusive notification feed on the side.
- **Inventory/Log Modal:** A toggleable full-screen or modal window (Tab/I) to view detailed stats, inventory (`StorageComponent`), and local rumors (`Secrets`).

## 5. Progression & Death (Persistent Lineage)
- **Living:** The player exists within the 60 TPS simulation. Buying all food causes local famines; murdering a guard spawns a Blood Feud.
- **Dying:** When `Vitals.Blood <= 0` or `Needs.Food <= 0`, the `DeathSystem` intercepts the despawn logic if the entity is `Possessed`. It caches the `heirData` and pauses the simulation, pushing the `StateManager` to `StateGameOver`.
- **Rebirth:** The player can seamlessly take control of their genetic heir (inheriting debts, hooks, and prestige) or start over.

## 6. Implementation Scope (Sub-Projects)
Because this is a complete rewrite, it will be broken down into focused tasks:
1. `internal/ui`: Implement `StateManager`, `GameState` interface, and the core states (`MainMenu`, `Playing`, `GameOver`).
2. `internal/render`: Update `EbitenApp` to utilize the `StateManager` and implement the virtual resolution / Split-Space rendering.
3. `internal/systems/player_input.go`: Overhaul to support Mouse Aim, Left-Click (Attack), and Right-Click (Interact).
4. `internal/systems/death.go`: Intercept player death to trigger `StateGameOver`.
5. `cmd/game/main.go`: Wire up the new state machine.
