# Phase 3: The Genesis Entities (Spawning & Genes)

_Objective: Introduce the first wave of simulation nodes (Pops) into the environment with basic survival math and biological inheritance limits._

## 3.1 Base Arche-Go Components

- **`PositionComponent`**: `{X, Y float64}`. Fractional precision required for sub-tick interpolation drawing.
- **`VelocityComponent`**: `{DX, DY float64}`. Immediate intended directional speed.
- **`IdentityComponent`**: `{ID uint64, Name string, BaseTraits uint32}`. Traits packed as bitmasks (e.g., `1<<1` for Gossip, `1<<2` for Lazy) for extreme query speed.
- **`GeneticsComponent`**: `{Strength uint8, Beauty uint8, Health uint8, Intellect uint8}`. Bounded 0-100 values dictating biological luck vectors.
- **`LegacyComponent`**: `{Prestige uint32, InheritedDebt uint32}`. Tracks social mass explicitly distinct from personal traits.

## 3.2 The Genesis Spawner

- **`FamilySpawner` Logic:** Run once at Tick 0. Queries `MapGrid` for walkable/habitable tiles.
- Uses `GlobalRNG` to select 100 starting locations.
- Instantiates `arche.Entity` groups representing "Family Clusters". Populates raw values for `Identities`, randomizing `Genetics` via Bell Curve algorithms.

## 3.3 The Metabolic Engine

- **`NeedsComponent`**: `{Food float32, Rest float32, Safety float32, Wealth float32}`. Starting values 100.0.
- **`MetabolismSystem`**: Standard ECS iterator. Evaluates all valid `NeedsComponent` payloads. Subtracts dynamic rate variables: `Food -= 0.05 * GeneticHealthModifier`.
- **`DeathSystem`**: Secondary ECS iterator. Scans for any Entity where `Needs.Food <= 0`. If found, trigger the `Despawn` pipeline, logging root causes to standard output.
