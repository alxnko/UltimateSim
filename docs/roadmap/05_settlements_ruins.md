# Phase 5: Settlement Birth, Genetics, & The Ruin Lifecycle

_Objective: Transform wandering entity clusters into stationary, physical settlements. Manage biological inheritance and ensure cities die gracefully without leaving ghost entities draining CPU._

## 5.1 Settlement Conversion System

- **`SettlementRuleSystem`:** An active ECS iterator tracking `FamilyCluster` tags.
- **Trigger Logic:** If `Velocity == 0` for 1000 consecutive ticks AND the current `Position` overlaps a `TileData` grid index where `ResourceDepot` values are high (e.g., Water + Wood > 50).
- **Execution:**
  1. Despawn the migrating `FamilyCluster` entity.
  2. Read origin coordinates, spawn a new `VillageEntity` at that exact float vector.
  3. Attach `StorageComponent` (Int arrays mapping inventory).
  4. Attach `PopulationComponent` (Integer scalar tracking raw headcount, abstracting individual AI nodes inside the city limits).

## 5.2 The Ruin Transformation ("Zombie Entity" Prevention)

- **The Problem:** Traditional games delete cities when population hits 0, erasing history. Keeping them active drains CPU as the engine checks if the empty city is hungry.
- **The Solution (`Ruin Transformation`):**
  - Iterate `PopulationComponent`. If count drops to 0 (due to famine/war/disease).
  - Do NOT call `Despawn()`.
  - Remove `PopulationComponent` and `NeedsComponent` from the entity.
  - Add `RuinComponent {Decay int, FormerName string}`.

## 5.3 Arche-Go Component Filters

- **Optimization Requirement:** Within `arche-go` construction of `MetabolismSystem` and `GossipDistributionSystem`, explicitly build `filter.Not(RuinComponentID)`.
- This absolutely ensures the ECS query iterators skip over ruins, instantly salvaging tens of thousands of wasted CPU cycles every frame.

## 5.4 Birth & Genetics Math

- **`BirthSystem`:** When `StorageComponent` hits arbitrary surplus, increment `PopulationComponent`.
- **Genetic Crossover Simulation:** Create internal `CitizenData` arrays dynamically representing generated births.
- `Child.GeneticsComponent` = Mathematical average of (Parent1 + Parent2) `GeneticsComponent` values (+/- 5 points RNG mutation via the Global Seed).
- Inherit `Identity.BaseTraits` via a 50% chance bitmask evaluation.
