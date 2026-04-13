# Phase 37: Naval Combat, Shipboarding & Privateering

## Overview
Expanding on Maritime Logistics (Phase 17), the oceans must become arenas for systemic conflict, piracy, and naval supremacy.

## Architecture & DOD Implementation
- **Ship Entities as Mobile Grids**: 
  - A `ShipEntity` is not just a point on a map; it contains a miniature, mobile sub-grid. Crew members (`NPC`s) physically walk around the deck of the ship while it moves across the macro-ocean.
- **Naval Artillery & Sinking**: 
  - Ships can be equipped with ballistas or cannons. Projectiles damage the ship's `HullComponent`. If `HullIntegrity` reaches zero, the ship sinks, instantly drowning all crew lacking `Floatation` gear and destroying all cargo.
- **Boarding Actions**: 
  - When two ships collide, their sub-grids lock together via a `BoardingComponent`. Crew members pathfind across grappling hooks and engage in brutal, close-quarters melee combat on the decks.
- **Privateering & Letters of Marque**: 
  - The `JusticeSystem` (Phase 18) applies to the high seas. Countries can issue `LetterOfMarque` items to captains, legalizing their attacks on rival nations' shipping lines without incurring global `CrimeMarkers`.