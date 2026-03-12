# Phase 18: The Justice Engine & Legal Logic

_Objective: Establish abstract "Laws" purely as mathematical rule-sets enforced by regional authorities, simulating crime and punishment as data._

## 18.1 Jurisdiction & Law Definitions

- **`JurisdictionComponent`:** Attached to `VillageEntity` or Capital arrays. Defines geometric boundary arrays and a list of `IllegalActionIDs` (e.g., executing an assault EventID, or bypassing a Guild Tax variable).
- **Contraband Logic:** Expand `MarketComponent` to flag certain `ItemIDs` dynamically as illegal within the `Jurisdiction` bounds.

## 18.2 Detection & The Guard System

- **`JusticeSystem`:** Iterates local `MemoryEvents`. If an NPC inside a Jurisdiction logs an event matching an `IllegalActionID`, they generate a `CrimeMarker` component.
- **Enforcement Nodes:** NPCs holding specific `JobComponent(Guard)` path towards entities bearing `CrimeMarker` tags.

## 17.3 Sentencing & Prisons

- **Punishment Math:** When a Guard successfully interacts with a marked entity, initiate punishment arrays.
- _Banishment:_ Remove `AffiliationComponent.CityID` and force positive velocity outward.
- _Execution:_ Run `DeathSystem` override.
- _Fines:_ Transfer `TreasuryComponent` wealth to the enforcing `CityID`.
