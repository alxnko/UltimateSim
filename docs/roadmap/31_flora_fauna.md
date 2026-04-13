# Phase 31: Flora, Fauna, & Animal Husbandry

## Overview
The world must feel alive beyond human settlements. Dynamic ecosystems of predators and prey, alongside the ability for the player to tame, ride, and breed animals, adds a massive layer of life-simulation depth.

## Architecture & DOD Implementation
- **`AnimalComponent`**: 
  - Defines an entity as flora/fauna rather than a humanoid Pop. Follows unique AI rules (e.g., wolves hunt deer; deer flee from noise).
  - Ecosystems naturally balance based on the `MetabolismSystem`—if wolves eat all the deer, the wolves starve.
- **Taming & Husbandry**: 
  - Players (and NPCs with the `JobHerder` tag) can use food to attempt to add a `TamedMarker` to an animal, adding it to their `AffiliationID`.
  - Tamed animals can be sheared for wool, milked, or slaughtered for meat, tying into the physical crafting economy (Phase 26).
- **Mounts & Cavalry**: 
  - A tamed horse can be interacted with to merge the player's `Position` with the mount, drastically increasing the `Velocity` vector and altering `AttackComponent` hitboxes for cavalry combat.