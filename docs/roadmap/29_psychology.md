# Phase 29: Psychology, Moods & Mental Breaks

## Overview
Drawing heavy inspiration from Dwarf Fortress, physical needs are not enough. Entities must have psychological limits. Trauma, stress, addictions, and mental breaks create unparalleled emergent stories.

## Architecture & DOD Implementation
- **`SanityComponent`**: 
  - A new DOD struct tracking `Stress`, `Mood`, and `AddictionLevel`.
  - Witnessing death, starving, or being ostracized (Phase 41) increases `Stress`.
- **Catharsis & Vices**: 
  - NPCs and players can reduce `Stress` by consuming specific goods (e.g., Ale at a Tavern) or attending religious gatherings.
  - Overconsumption leads to `AddictionLevel` increases, creating a new, overriding physiological `Need`.
- **Mental Breaks**: 
  - If `Sanity.Stress` reaches maximum, the `WanderSystem` AI state flips to a `MentalBreak` state (e.g., Berserk, Catatonic, or Melancholy).
  - A Berserk NPC will attack random entities, generating chaos, while a Catatonic NPC will freeze, potentially starving to death.
- **Phobias & Quirks**: Accumulated trauma can mutate `Genetics.BaseTraits`, giving an NPC a permanent phobia of the dark, or a specific animal, physically altering their pathfinding weights (`HPA*`) to avoid certain map tiles.