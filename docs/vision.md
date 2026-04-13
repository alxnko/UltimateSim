# Boundless Sovereigns: Master Vision

**The High Concept:**
Boundless Sovereigns is a "bottom-up" medieval society engine where grand strategy emerges purely from individual, microscopic agency. The simulation accounts for Physical, Cultural, and Systemic forces that bind independent AI agents (Pops) to the map.
Players inhabit a physical character in a living ecosystem, blending the deep, granular colony-sim interactions of *Rimworld* and the systemic action-RPG chaos of *Streets of Rogue* with the grand-strategy arcs of *Kenshi* and *Europa Universalis*. You directly control your current character in real-time. You can choose to be an agent of chaos, a dynasty-building noble, or simply a regular person working a mundane job. 

**Core Philosophy:**

- **Micro-Foundations for Macro-Emergence:** The macroscopic world (wars, economies, nations) does not operate on hidden, abstract formulas. It is entirely dependent on the physical, micro-level actions of NPCs and the player. A city grows because individual NPCs physically hauled wood, got hired via contracts, and physically hammered walls and workbenches into existence. If a sword exists, someone mined the iron, smelted it at a forge, and sold it.
- **Total Simulation:** There is no hardcoded "End Game." The world is self-sustaining through endless cycles of growth, decay, and rebirth generated entirely by the physical actions, labor, memories, and needs of the NPCs.
- **Total Freedom & Physical Constraints:** The player has absolute freedom to interact with the world, but is strictly bound by their character's physical and social reality. You are limited by your wealth, your family's nobility (or lack thereof), your physical stamina, and your relationships. You can live a quiet life as a local baker or scheme to overthrow an empire, but both require interacting with the physical and social systems.
- **Inherited Legacy:** When the player character dies, they continue the legacy as an heir, inheriting not just physical items and real estate, but Social Standing, Debts, and Blood Feuds.

**The Golden Rules of the Simulation (The Closed Loop):**
To achieve absolute simulation stability, the engine operates on a massive closed loop of negative and positive feedback involving six final "Total Simulation" layers:

1.  **Geography:** Biomes and resources dictate where people move.
2.  **Biology:** Genetics, seasonal pulses, aging, and immunity dictate survival pressures.
3.  **Economy:** Supply, demand, and debt dictate material motivation.
4.  **Social Hierarchy:** Families, Clans, and Guilds dictate the power structure.
5.  **Information:** Gossip, Secrets, and "Hooks" dictate uncertainty and "Vibe".
6.  **Culture:** Languages, Dialects, and Beliefs dictate trust and unity.

This document serves as the high-level intent. For the exact systemic breakdown of how the simulation handles these pillars, refer to `mechanics.md`. For the technical implementation in Go and ECS, refer to `architecture.md`.
