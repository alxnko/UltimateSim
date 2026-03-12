# Phase 14: True Individual NPCs & Dynamic Villages

This phase represents a critical architectural shift from the early prototype logic. It migrates the simulation's atomic moving unit from an abstracted `FamilyCluster` to true individual `NPC` entities with dual affiliations (Family and Clan).

## 1. Goal
To allow players to possess and interact with a truly individual character who belongs to a dynamic family/clan structure, capable of independently leaving, migrating, and settling based on their own Needs and Traits.

## 2. Core ECS Components Added/Modified

- **`NPC` (Tag Component)**
    - Replaces `FamilyCluster`. Identifies a single human actor in the simulation.
- **`Affiliation` (Component Update)**
    - Add `FamilyID uint32` alongside the existing `ClanID`, `CityID`, etc.
    - Enables rendering groups of individuals that belong to the same immediate family.
- **`Village` (Refined Logic)**
    - Villages no longer "delete" NPCs into abstract arrays. They become stationary "Hub" entities. NPCs physically stand at the Village coordinates and update their `CityID` to effectively "live" there.

## 3. Systems Refactored

- **`FamilySpawnerSystem` -> `NPCSpawnerSystem`**
    - Spawns 5-10 individual `NPC` entities per habitable tile.
    - Assigns identical `FamilyID` and `ClanID` to NPCs spawned on the same tile to establish the initial family network.
- **`WanderSystem` / `MovementSystem`**
    - Pathfinding resolves per-NPC. While family members share similar starting Needs (and thus travel together initially), diverging Traits (e.g., `TraitRiskTaker`) allow individuals to split from the group and create new Desire Paths or found new Villages.

## 4. Rendering & Possession Impacts

- Visualizing single NPCs instead of clusters.
- Possession exclusively targets single `NPC` entities, not stationary `Village` entities.
- Selection UI displays specific NPC names (e.g., "John Doe") rather than numerical clusters.
