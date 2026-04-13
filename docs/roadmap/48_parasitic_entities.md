# Phase 48: Symbiotic & Parasitic Entities (Vampirism)

## Overview
Diseases (Phase 10) evolve into complex, transformative entities that fundamentally alter how an NPC interacts with the simulation.

## Architecture & DOD Implementation
- **The Parasite Component**: Fungal spores or viral curses (Lycanthropy, Vampirism) that alter an entity's `MetabolismSystem` (e.g., needing `Vitals.Blood` instead of Food, or burning in sunlight).
- **Hidden Infections**: Vampiric NPCs must hide their nature, utilizing Subterfuge (Phase 32) to blend into society, hunting at night while maintaining a normal `JobComponent` by day.
- **The Witch Hunter AI**: Unique investigator NPCs who parse the `GossipDistributionSystem` for anomalies (e.g., missing blood, bodies drained) to track down and execute parasitic entities.