# Phase 26: Physical Labor, Crafting & Job Contracts

## Overview
To solidify the simulation as a *Rimworld*-style microscopic economy, goods and architecture cannot simply appear out of thin air. They must be manually created through physical labor. This phase introduces the labor economy, where time, raw materials, and physical actions are monetized. The player and NPCs must accept jobs, earn wages, and physically craft or build the world.

## Architecture & DOD Implementation
- **The `JobContractComponent`**:
  - A new DOD struct defining a requested task. Fields include `EmployerID`, `TaskType` (Build, Craft, Haul, Harvest), `TargetEntity`, `RequiredTicks`, and `Wage` (float32).
  - Can be attached to a global `ContractBoard` entity within a city, functioning as a localized job market that NPCs and players can query.
- **Physical Construction & Hauling**:
  - When an NPC accepts a "Build Wall" `JobContract`, they must first physically walk to a material source (e.g., a chest holding Wood), acquire the material, haul it to the construction site, and spend `X` ticks iterating their `Action` state.
  - The macroscopic `CityBinderSystem` now only generates blueprints (`BlueprintComponent`), which remain impassable and transparent until fully constructed by laborers.
- **Workbenches & Crafting (`CraftingSystem`)**:
  - Advanced goods (Weapons, Armor, Tools) require a `WorkbenchComponent` attached to a specific `FurnitureEntity` (e.g., an Anvil or a Loom).
  - An NPC with `JobArtisan` must actively occupy the `Position` of the Anvil and spend time processing `ItemIron` into `ItemSword`. If no workbench exists, no swords can be created, directly tying macro-economic supply to physical micro-infrastructure.
- **Dynamic Employment**:
  - Unemployed NPCs naturally query the local `ContractBoard` based on their `Needs.Wealth`.
  - Players can interact with the board to take jobs (e.g., hauling stone for wages to afford food) or post jobs (e.g., spending their inherited wealth to hire 10 NPCs to construct a large player estate).
- **Player Freedom**:
  - The physical economy provides the foundation for total player freedom. You can be a regular lumberjack fulfilling contracts, or a guild master who owns the town's only Anvil, manipulating the local economy by controlling access to crafting infrastructure.