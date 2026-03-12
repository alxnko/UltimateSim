# Phase 9: Logistics, Caravans, Infrastructure, & Artifacts

_Objective: Make the ECS economy deeply physical. Move physical goods across the grid via pathfinding, calculate organic structural wear (roads), and spawn distinct historical Items._

## 9.1 The Caravan Entity

- **Demand Calculus:** If a `VillageEntity` processes a negative delta inside its `StorageComponent` against localized need requirements, instantiate a `CaravanEntity`.
- **The Entity Bind:** `CaravanEntity` possesses `Component.Payload` containing specified integer limits of trade goods.
- **Labor & Personnel Requirement:** Caravans are not autonomous. They MUST be crewed by individual `NPC` entities functioning as Merchants, Teamsters, and Guards. If a village cannot hire enough NPCs, the caravan cannot depart.
- **Passenger Economy:** Individual `NPC` entities with high `Wealth` and a desire to migrate can pay the Caravan owner to travel safely as passengers, generating an active travel economy.
- **Routing Integration:** Pushes destination vectors to the Phase 4 HPA\* `RoutingSystem`, physically traversing the array map towards the supplier or market. The speed of the caravan is dictated by the slowest NPC in the group.

## 9.2 Dynamic Attrition

- **Physical Goods Rot:** Goods cannot sit infinitely in arrays.
- **`SpoilageSystem`**: Evaluates biological `StorageComponent` trackers (Grain, Meat). Decrements limits by constant fractional values across tick times.
- **`RustSystem`**: Evaluates non-biological parameters (Iron Tools).

## 9.3 Infrastructure Wear System (Desire Paths)

- **Map Mutability:** The terrain degrades dynamically.
- **Execution Loop:** Every time a `VelocityComponent` carrying entity transitions `Position` vectors across a specific `TileData` grid index:
  1. Increment `TileStateComponent.FootTraffic` for that specific tile.
  2. Continual increments mathematically reduce extreme `MovementCost` constants on that tile.
  3. Forests become cleared paths, mountains generate safe passes over generations—organically creating an interconnected road grid devoid of player instruction.

## 9.4 Physical Legend Components

- **`LegendComponent`**: `{Name String, History []EventID, Prestige int}`. Represents physical items (crowns, ledgers, ancient swords).
- Artifacts are completely distinct entities tracking their own isolated history strings mapped to the Global Registry.

## 9.5 Item Inheritance

- **Death Hook Integration:** Link into Phase 3's `DeathSystem`.
- If an entity possessing extreme `LegacyComponent.Prestige` despawns:
  - Spawn an independent `ItemEntity` equipped with the `LegendComponent` matching former status.
  - This physical entity generates massive `Legitimacy Hooks` in the AI evaluations, pushing emergent wars as Clans target possession of the physical data struct.
