# UltimateSim: Boundless Sovereigns Coding Standards

## 1. Mandatory E2E Testing
Every new ECS System must be accompanied by end-to-end (E2E) testing using Go's built-in `testing` package. This ensures the system functions correctly in isolation and alongside other systems within the `arche-go` ECS environment.

## 2. ECS Filter Usage
All logic processing entities must enforce strict **Arche-Go filter usage**.
- Ensure filters accurately query for only the necessary components.
- This prevents the processing of "Zombie Entities" or applying logic to objects missing critical dependencies, avoiding panics and maintaining simulation integrity.

## 3. The Law of Macro/Micro Parity (No Background Math)
To ensure the game feels like a living world and not a spreadsheet, **every single feature must exist on both the Macro (Strategic) and Micro (Action-RPG) levels.** 
- **No Background Math:** If a system calculates a value (e.g., a city is starving, a religion is spreading, a faction is at war), the player MUST be able to see it and interact with it physically. 
- **Macro Lenses:** Every systemic data point must have a corresponding visual "Lens" on the strategic map (e.g., Political Borders, Cultural Heatmaps, Religious Influence, Trade Route Density, Crime Rates).
- **Micro Interactions:** Every systemic data point must have a physical representation in the 3D/2D Action-RPG layer. If a town is wealthy, there must be physical coins in chests. If a religion spreads, there must be physical shrines or preachers. If an item is crafted, there must be a physical workbench with a texture.

## 4. Pull Request Rules
Every PR submitted must satisfy the following criteria:
- **Updated Test Suite**: Includes tests covering new or updated logic. Tests must pass locally.
- **Performance Profiling**: If the change involves core simulation loops, data structure adjustments, or complex math, a performance profile (pprof) check must be included to verify no regressions in cache efficiency or execution time occurred.
- **Parity Check**: The PR must explicitly state how the new feature is visualized on the Macro Map (Lenses) and how the player interacts with it on the Micro Map (Physical objects, sprites, dialogue).
