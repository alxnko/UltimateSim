# Phase 13: Stability, Tooling, & Balance Loops

_Objective: Implement the final self-correcting negative feedback systems defining the Grand Strategy logic limits, and robust serialization for saving/modding._

## 13.1 Local Price Discovery (Market Logic)

- **`MarketComponent`:** `{map[ItemID]float32 price}` attached to `VillageEntity`.
- **`PriceDiscoveryLoop`:** Each city establishes dynamic local prices mathematically determined by its current `StorageComponent` metrics.
- Creates extreme local disparity signals (e.g., Grain hits max float value threshold due to immediate famine starvation stats), which cleanly instigate the mapping routines deploying `CaravanEntity` rescues (Phase 9).

## 13.2 Labor Rebalancing

- **`CareerChangeSystem`:** The engine needs algorithmic elasticity against collapse.
- If Price values mapping Grain/Wood severely cross bounds limits `X`, force specific lower-tier processing NPCs analyzing the `JobComponent` parameters to strictly adopt base `Farmer/Lumberjack` flags.

## 13.3 Jealousy Vulnerability

- **Dynamic Rumor Modifiers:** High-Prestige entities (kings, heroes) linearly spike their `Identity.RNG` vulnerability fields for surrounding node grids, dramatically elevating the likelihood that neutral adjacent networks autonomously generate negative `SecretID` leakages executing against them.

## 13.4 The Seasonal Pulse

- **`CalendarSystem`:** A global tick modifier affecting the primary bounds rules.
- Winter Boolean limit: Mutably scales the `NeedsComponent` decay matrices (e.g., `1.5x` calorie burn rates globally) while statically inflating active HPA\* tile traversal numerical costs per step.

## 13.5 Serialization & Modding hooks

- **`go-sqlite3` Backends:** Array state mapping execution to SQLite disk files, generating extremely minimal save logic strings by reading from `arche.World` directly.
- **`go.starlark.net` execution:** Externalize key variable thresholds (`Jealousy` weights, `Metabolism` limits, `Hook` spend math outputs) securely into external file structures ensuring pure data-driven modder controls independent of core Go logic loops.
