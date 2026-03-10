# Developer Knowledge Base: Internal Activity Log

## Current Phase / Task
- **Phase 1: Initialization, Determinism, & ECS Bootstrapping** (from `docs/roadmap/01_foundation.md`)

## Active Component IDs & Data Structures
*Note: All structs must follow strict flat memory rules for Data-Oriented Design (DOD) to ensure cache alignment.*
- Structs use integer IDs instead of pointers (e.g., `TargetID uint64`).
- Use `uint8` and `uint16` where possible to minimize memory overhead instead of `int`.

**Implemented Structures (`internal/components/basic.go`):**
- `Identity`: `ID uint64`
- `Position`: `X, Y float32`
- `Velocity`: `X, Y float32`

**Design Decision Log (Phase 01):**
- **Data Types & CPU Cache**: `float32` was deliberately chosen over Go's default `float64` for `Position` and `Velocity` to strictly adhere to Data-Oriented Design (DOD) constraints. A `float32` takes 4 bytes instead of 8, doubling the density of our flat arrays. This tightly packed memory ensures significantly higher L1/L2 cache hit rates when the ECS iterates sequentially over 100,000+ entities, guaranteeing our 60 TPS performance goal is met.

## Global RNG Seeding Strategy
- **Seed Methodology**: A single, global singleton seed handles all stochastic events (terrain generation, birth systems, plague spawns, weather phenomena) to maintain absolute determinism across all simulation components.
- **Implementation**: Utilizes Go's `math/rand/v2` with `ChaCha8` engine for deterministic pseudorandom number generation.
- **Thread Safety**: Implemented behind a `sync.Mutex` in `internal/engine/rng.go` to prevent race conditions during highly parallelized ECS operations while maintaining strict sequencing.
