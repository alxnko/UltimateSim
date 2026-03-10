# Phase 4: Autonomous Nodes (HPA\* & Migration)

_Objective: Drive agency in the entities without stalling CPU constraints. Provide intelligence through optimized mathematical abstractions of the map grid._

## 4.1 Hierarchical Pathfinding (HPA\*) Implementation

- **Why HPA\*?** Raw A* executed by thousands of entities cross-map spikes O(n*log(n)) across vast node depths, choking single ticks.
- **Grid Abstractor (`/pkg/math/hpa`):** Divide `MapGrid` into 16x16 or 32x32 "Regions."
- Map portal nodes mapping Region borders dynamically mapping connected gateways.
- NPCs calculate strategic routes Region-to-Region first (very cheap), and then only compute tactical grid-level route data for the immediate region they are standing inside.

## 4.2 Async Path Queue Pool

- **Goroutine Load Distribution:** `arche-go` systems execute sequentially. Path computations must not block the tick render frame.
- **`PathRequestQueue` mechanism:** Implement worker pool (e.g., 4 dedicated persistent Goroutines). The ECS `WanderSystem` queues an entity ID + Target coordinate and immediately yields execution. Workers crack HPA\* math and return `[]Vec2` path data to the entity asynchronously over subsequent ticks.

## 4.3 Trait & Need Driven Targeting

- **`WanderSystem` Evaluator:** When an entity has NO active path vector:
  1. Read `NeedsComponent`. Is `Food` dominant missing need? Query map for `ResourceDepot(Food)`.
  2. Read `Identity.BaseTraits`. If `RiskTaker` bit is true, add arbitrary weight positive values to distant unknown regions. If `Cautious`, penalize distances moving far from rivers or other clusters.
- Generates the targeted coordinate mapping logic and queues to the Path Worker Pool.

## 4.4 Resolving Kinematics

- **`MovementSystem`:** Iterate entities containing active Path nodes and a `VelocityComponent`. Apply DeltaTime math updating `Velocity` toward specific targeted vector nodes. Transfer float values cleanly to update `PositionComponent`. Process bounds verifications so values do not map outside arrays.
