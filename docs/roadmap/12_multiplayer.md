# Phase 12: Network Delta Sync & Multiplayer

_Objective: Expose the robust ECS engine to internet concurrency, driving stable long-running strategy sessions over standard network protocols with minimum payload weight._

## 12.1 Network Sockets & Stream Binding

- **Protocol Limits:** Bind raw `net` UDP endpoints for real-time positional data, and TCP fallbacks mapping `SparseHookGraph` exchanges where dropped packets would corrupt session ledgers.
- Establish Server-Authoritative tick logic over all connected Clients.

## 12.2 Delta Extraction Queries

- **The Bandwidth Problem:** Sending 100,000 entity `Position` arrays every frame is absolutely impossible.
- **The Solution:** Query the `arche-go` ECS explicitly for entities marked with `Velocity != 0` flag statuses or specifically designated mapping updates executed within the current `TickManager` 60Hz window.
- Generate payload array structs mapping only these fractional data modifications.

## 12.3 Robust Client Prediction & Smoothing

- **Utilizing Phase 1 constraints:** Because the simulation logic utilizes a Seeded Global RNG exclusively, all clients possess identical mathematical rule-sets.
- Clients simulate abstract `WanderSystem` path logic identically to the Server locally.
- The network is only strictly required to synchronize unpredictable inputs (Player Actions, explicit `LegendComponent` deaths) rather than processing raw positional coordinate feeds for distant AI units.
