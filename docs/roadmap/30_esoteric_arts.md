# Phase 30: The Esoteric Arts (Alchemy & Rituals)

## Overview
Magic in this simulation isn't about throwing fireballs with a mana bar; it's about deep, systemic esoteric knowledge, alchemy, and cults. Magic is dangerous, highly illegal in most jurisdictions, and requires physical preparation.

## Architecture & DOD Implementation
- **Alchemy & Herbalism**: 
  - `FloraComponent` on map tiles allows gathering of specific, procedurally generated herbs.
  - At an Alchemy Workbench, these physical items can be combined into potions that directly manipulate ECS values (e.g., a potion that zeroes out `Vitals.Pain` or temporarily boosts `Genetics.Strength`).
- **Esoteric Rituals**: 
  - Magic requires a `RitualComponent`. Players and NPCs must physically draw runes on the ground (altering `TileStateComponent`), place specific offerings, and wait for planetary alignments (`CalendarSystem`).
  - Successful rituals can manipulate macro-systems (e.g., curing a regional `DiseaseEntity` or summoning a massive `Storm`).
- **Secret Societies (Cults)**: 
  - Tied to Phase 49 (Witch Hunts). Practitioners must form hidden `Affiliation.GuildID` groups. They use the `SecretRegistry` to share ritual knowledge in the shadows.
  - If a player joins a cult, they gain access to power but risk execution by the `JusticeSystem` if their `EsotericMarker` is discovered.