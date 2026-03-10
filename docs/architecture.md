# Boundless Sovereigns: Technical Architecture (Go & ECS)

The extreme depth of the "Total Simulation" outlined in the `vision.md` and `mechanics.md` documents is only possible through specific, highly optimized technical choices. Boundless Sovereigns relies on a modern concurrent architecture to process massive agent volumes without object-oriented lag.

## 1. Language & Concurrency

- **Go (Golang):** Chosen as the primary language for its performance-first design, memory safety, and native, lightweight concurrency model.
- **Parallel Simulation (Goroutines):** Every major organization, trade caravan routing protocol, or geographical region runs its logic on independent goroutines. This allows the simulation to saturate all available CPU cores, keeping the game ticking smoothly even with 100,000+ active agents.

## 2. Data Design (ECS & DOD)

- **Entity Component System (ECS):** Powered by the highly optimized `arche-go` library. All NPCs, Cities, and physical Items are simply integer IDs. Their attributes (Health, Hunger, Hooks, Beliefs, Language Profile) are "Components," and the active logic linking them are entirely standalone "Systems" (e.g., `InformationLeakageSystem`, `CaravanMovementSystem`, `SpoilageSystem`, `LinguisticDriftSystem`).
- **Data-Oriented Design (DOD):** Traditional OOP classes (e.g., `NPC.Eat()`) are strictly banned. Data must be tightly packed in flat memory arrays within the ECS to maximize CPU L1/L2 Cache hits. This drastically boosts sequential iteration speed, guaranteeing 60+ simulation ticks per second.

## 3. Data-Driven Scalability (The "Lego" Principle)

Because the engine is strictly constructed on ECS `arche-go` foundations, the codebase is infinitely scalable without risking "spaghetti code."

- **Infinite Feature Plugs:** Adding a new feature never requires rewriting the core loop. To add "Disease," we simply create a new `ContagionComponent` and a `DiseaseSystem`. The existing `MetabolismSystem` remains entirely ignorant of it.
- **Decoupled Logic:** The Headless Simulation (Data) is completely severed from Rendering (View). We can overhaul the entire 3D camera or add new visual layers without ever touching the logic for how a Clan tracks wealth or gossip. The architecture guarantees clean expansion slots.

## 4. Rendering & Graphics

The engine must handle shifting instantly between the grand strategic view and the granular individual view without dropping simulation state.

- **Hybrid Map View (Ebitengine):** Uses `Ebitengine` for high-performance 2D strategic map overlays. This acts as the geopolitical "paint" showing emergent borders, active physical trade routes (Desire Paths), language heatmaps, and town density.
- **Character Possession View (raylib-go):** Uses `raylib-go` the moment a player "possesses" a specific Clan member, shifting the camera into the city streets. The 3D render engine handles direct 3rd-person movement, object interaction, and visual representation of the ECS data (e.g., visually seeing a blacksmith hammering a rusted piece of iron).

## 4. Persistence & Modding

- **State Management:** The complete ECS state is periodically snapshotted to SQLite for local save-game persistence. Fast-changing, highly volatile data (like gossip ping-pongs) is handled exclusively in memory.
- **Modding (Scripting Engine):** **Starlark** is embedded directly into the Go engine to allow external behavior scripts (e.g., modifying the precise probability formula of the Gossip Infection algorithm or creating new Traumatic Tradition triggers) without ever needing to recompile the core Go engine.
