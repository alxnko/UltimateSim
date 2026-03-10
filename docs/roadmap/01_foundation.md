# Phase 1: Initialization, Determinism, & ECS Bootstrapping

_Objective: Lay the Go and ECS foundations prior to any simulation math, focusing strictly on cache locality and determinism for large-scale operations._

## 1.1 Environment Setup

- **Go Module Initialize:** `go mod init github.com/yourname/boundless`
- **Directory Scaffolding:** Create standard Go layouts:
  - `/cmd/game` (entrypoint)
  - `/internal/engine` (arche world, tick orchestration)
  - `/internal/components` (all raw structs)
  - `/internal/systems` (ECS logic loops)
  - `/pkg/math` (HPA\*, grid conversions)

## 1.2 Deterministic Simulation Focus

- **Global Seeded RNG:** Implement a specialized PRNG instance (e.g., using `math/rand/v2` with `ChaCha8`).
- **No Goroutine Race Conditions:** ECS systems mutating shared states MUST sequence locally. If doing multithreading inside a system via `arche`'s builders, write determinism-safe iterators. All terrain generation, birth systems, plague spawns, and weather phenomena are generated strictly off this singleton seed.

## 1.3 ECS Core (`arche-go`) Setup

- **Initialize World:** Mount `arche.World` object.
- **TickManager (`internal/engine/tick_manager.go`):**
  - Fixed Time Step: Force precisely 60 TPS bounds. Sleep logic to cap max loops, preventing simulation "fast-forward" under varying hardware.
- **SystemRunner:** Use the `arche` provided or write a custom schedule. Sequence systems rigorously: Input -> AI -> Movement -> Resolution -> Cleanup.

## 1.4 Hardware Affinity & Rendering Bridging

- **Thread Pinning:** Use `runtime.LockOSThread()` within `/cmd/game/main.go` on the main Simulation Goroutine and the Window Context Goroutine. Prevents OS-level cache invalidations on multicore Ryzen CPUs.
- **Tick-Render Decoupling:** Build logic inside `TickManager` computing `alpha` value (fractional time between ticks). Expose this `alpha` variable to external renderer objects, enabling 144Hz screen painting over 60Hz deterministic math output.

## 1.5 Data-Oriented Design (DOD) Verification

- **Strict Memory Rules:** No pointer types inside Components (e.g., use `TargetID uint64`, not `*Target Entity`). Ensure flat struct packing. Use `uint8` and `uint16` where possible instead of `int` overhead.

## 1.6 Telemetry & Profiling

- Implement command-line readouts testing `Ticks Processing Time (ms)`.
- Boot `net/http/pprof` instance on `localhost:6060`. Standardize tracking Goroutine profiles and L1/L2 data access misses as complexity scales.
