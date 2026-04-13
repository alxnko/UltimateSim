# Phase 27: Deep Conversation & Knowledge Brokerage

## Overview
To rival Dwarf Fortress's Adventure Mode, interacting with NPCs must go beyond simple menus. Information is power. The player must be able to interrogate, persuade, lie, and learn from every entity in the world.

## Architecture & DOD Implementation
- **Dialogue ECS State**: 
  - When the player engages an NPC, the game pauses macro-ticks and enters a `DialogueState`. 
  - Responses are procedurally generated based on the NPC's `BeliefComponent`, `MemoryComponent`, and current `Needs`.
- **Knowledge Transfer**: 
  - Players can ask "Where is [Entity/Item]?" NPCs will pathfind their memory arrays and provide directions if they have a `Secret` containing that location.
  - **Skill Tutoring**: Players can pay wealthy or highly skilled NPCs (e.g., Master Blacksmiths or ancient Scholars) to transfer `Genetics/Skill` components to the player over time.
- **Persuasion & Intimidation**: 
  - Tied directly to the `SparseHookGraph`. You can threaten an NPC with a weapon to extract a secret (generating a negative hook and a `CrimeMarker`), or bribe them (generating a positive hook).
- **Lying & Forgery**: Players can inject false `Secrets` into the `GossipDistributionSystem`, intentionally framing a rival noble for a crime to incite the `JusticeSystem` against them.