# Developer Knowledge Base: Internal Activity Log

## Current Phase / Task
- **Phase 1: Initialization, Determinism, & ECS Bootstrapping** (from `docs/roadmap/01_foundation.md`)

## Active Component IDs & Data Structures
*Note: All structs must follow strict flat memory rules for Data-Oriented Design (DOD) to ensure cache alignment.*
- Structs use integer IDs instead of pointers (e.g., `TargetID uint64`).
- Use `uint8` and `uint16` where possible to minimize memory overhead instead of `int`.
- Example basic structures to be implemented:
  - `Position`
  - `Velocity`
  - `Identity`

## Global RNG Seeding Strategy
- **Seed Methodology**: A single, global singleton seed handles all stochastic events (terrain generation, birth systems, plague spawns, weather phenomena) to maintain absolute determinism across all simulation components.
- **Implementation**: Utilizes Go's `math/rand/v2` with `ChaCha8` engine for deterministic pseudorandom number generation.
