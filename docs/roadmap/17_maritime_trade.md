# Phase 17: Maritime Reach & Naval Logistics

_Objective: Expand the economy to the oceans without rewriting ground pathfinding logic, creating massive trans-continental supply chains._

## 17.1 Coastal Ports & Ship Construction

- **`PortComponent`:** Added to `VillageEntity` structures resting adjacent to `TileData(Ocean)`.
- **Naval Spawning:** When an overland `CaravanEntity` reaches a `PortComponent`, instantiate a `ShipComponent` attached to a new entity, transferring the `StorageComponent` payload.
- **Maritime Labor Market:** Ships require a massive crew of individual `NPC` entities holding the `JobComponent(Sailor)` and a `JobComponent(Captain)`. Ship owners must pay high wages, drawing laborers away from farming/mining in coastal cities.
- **Trans-oceanic Migration:** Just like caravans, ships hold `Passenger` slots. NPCs fleeing famine or seeking new lands can purchase passage across oceans.

## 17.2 Oceanic Pathfinding

- **HPA\* Extension:** Generate a secondary nav-mesh array mapping specifically over `BiomeID(Ocean)`. Water tiles have a baseline `MovementCost` of 1 for ships, dramatically speeding up mass transit compared to overland arrays.
- **`NavalRoutingSystem`:** A specialized pathfinder that traces vectors across deep water grids, avoiding shallows or seasonal ice caps calculated by the `CalendarSystem`.

## 15.3 Maritime Attrition & Piracy

- **`StormSystem`:** Utilize the `GlobalWeatherSystem` to spawn dynamic hurricane vectors over water arrays, executing massive `DecaySystem` damage to traversing `ShipComponent` hulls.
- **Naval Piracy:** Rogue entities operating outside `CityID` bounds aggressively path towards high-wealth `ShipComponents`, creating emergent Pirate havens.
