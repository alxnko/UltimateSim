# Phase 19: Advanced Biology & Ecology

_Objective: Extend the simulated depth of biology, moving beyond simple trait arrays to track long-term recessive genetics, immunities, and macro-climate shifts._

## 19.1 Deep Genetics (`GenomeComponent`)

- **Expansion:** Replace static `GeneticsComponent` integer bounds with a `GenomeComponent` holding distinct dominant/recessive bitmasks for traits like Height, Disease Resistance, and Agility.
- **Inbreeding Penalties:** If `BirthSystem` detects identical or overly similar `GenomeComponent` arrays (simulating Clan isolation), drastically penalize Health or attach mutation flaws.

## 19.2 Ecological Drift (Climate Change)

- **The Problem:** The `MapGrid` generated in Phase 2 remains static forever.
- **`GlobalWeatherSystem`:** A macro-system executing exactly once per 100,000 ticks.
- **Execution:** Shifts global Temperature mapping. Forests may dry into Plains, rendering localized `WoodValue` drops. Cities must dynamically spawn `CaravanEntity` fleets (Phase 9) to import newly scarce lumber, forcing historic geopolitical shifts based purely on ecological math.
