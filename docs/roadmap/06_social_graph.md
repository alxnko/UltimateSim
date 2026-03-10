# Phase 6: The Social Graph & Sparse Hook Matrix

_Objective: Create the multi-vector relational web that ties tens of thousands of independent entities together, utilizing mathematically clean sparse arrays to eliminate the multi-gigabyte RAM bomb._

## 6.1 Societal Hierarchies (`AffiliationComponent`)

- **Struct Design:** `{ClanID uint32, GuildID uint32, CityID uint32, CountryID uint32}`.
- Provides immediate array-index O(1) lookups for political mapping without traversing pointers.
- **`CityBinderSystem`:** Runs every 10,000 Ticks. Calculates spatial radii for wandering clusters. Assigns nearest active `VillageEntity` index to their `CityID` field dynamically, mimicking refugee or local vassal ingestion.

## 6.2 Interaction Telemetry (`MemoryComponent`)

- **Event Log:** An entity cannot comprehend global history, only what it experiences.
- **`MemoryEvent Struct`**: `{TargetID uint64, InteractionType uint8, Value int8, TickStamp uint64}`.
- **`MemoryComponent`**: A circular Ring Buffer (`[50]MemoryEvent`). Oldest memories are physically overwritten (forgotten) unless transferred to material artifacts (Phase 9).

## 6.3 The Sparse Hook Graph implementation

- **The Matrix Problem:** A standard 2D array matrix tracking every entity against every other entity ($100,000 \times 100,000$) requires $10^{10}$ integers, immediately crashing standard RAM caches.
- **The Sparse Solution:**
  - Implement `SparseHookGraph` at the `/internal/engine` level.
  - Type mapping: `map[EntityID]map[TargetID]int`.
  - Only consumes RAM bytes when a relationship explicitly exists.
- **Hook Writers:**
  - `AddHook(EntityA, EntityB, Points int)` function appended dynamically resulting from positive `InteractionType` evaluations inside `MemoryEvent` logic.
  - `SpendHook(EntityA, EntityB, Points int)` decrement logic attached to request execution systems (Debts, Trades, Actions).
