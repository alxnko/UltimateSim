# Phase 47: Siege Defense & Trap Engineering

## Overview
Defending a castle requires as much engineering as attacking it. The player can build lethal, automated defense networks.

## Architecture & DOD Implementation
- **Trap Entities**: Pressure plates linked via logical wires to automated crossbow turrets or trapdoors.
- **Mechanisms & Logic Gates**: A rudimentary logic system (AND, OR, NOT gates) allowing players to build complex chain reactions (e.g., pulling a lever opens a dam, flooding the moat, and dropping the drawbridge).
- **Boiling Oil & Physics Hazards**: Pouring heated fluids (Phase 43) from Z-levels above (Phase 41) onto attackers below, calculating catastrophic burn damage across multiple entities.