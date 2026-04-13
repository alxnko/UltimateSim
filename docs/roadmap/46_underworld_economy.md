# Phase 46: The Underworld Economy & Crime Syndicates

## Overview
If a city is heavily taxed or corrupt, a shadow economy naturally forms parallel to the legitimate one.

## Architecture & DOD Implementation
- **Thieves Guilds**: Criminals naturally flock together into `Affiliation.GuildID` syndicates. They operate out of hidden subterranean bases (Phase 34) or disguised taverns.
- **The Black Market**: Syndicates maintain their own `MarketComponent` networks for contraband (poisons, stolen artifacts).
- **Turf Wars**: Rival syndicates utilize the `BloodFeudSystem` natively to fight over control of specific city blocks, engaging in stealthy back-alley assassinations to secure monopolies on illicit goods.