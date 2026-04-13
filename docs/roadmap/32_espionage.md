# Phase 32: Espionage, Disguises & Subterfuge

## Overview
With a rigorous `JusticeSystem` and `SocialGraph` in place, the ultimate form of player freedom is the ability to bypass them through systemic trickery, espionage, and stolen identities.

## Architecture & DOD Implementation
- **`IdentityComponent` Spoofing (Disguises)**: 
  - Clothing is physicalized. If a player kills a Guard and equips their uniform (`EquipmentComponent`), the `JusticeSystem` query fails to recognize the player as a criminal, seeing the `FactionTag` of the uniform instead.
  - High `Perception` NPCs can roll to see through the disguise based on proximity and line-of-sight.
- **Forgery**: 
  - Using the Crafting system (Phase 26), players can craft fake `JobContracts` or false `Ledger` entries, altering the reality of the macro-simulation (e.g., forging a deed to a house, tricking the city administration into recognizing the player as the owner).
- **Systemic Poisoning**: 
  - Players can combine Alchemy (Phase 30) with Subterfuge. Adding a `ToxinComponent` to a physical food item and dropping it in a King's chest. When the King's `MetabolismSystem` forces him to eat, the toxin executes, assassinating the leader without a combat roll and leaving no immediate `CrimeMarker` trace.