# UltimateSim: Boundless Sovereigns Coding Standards

## 1. Mandatory E2E Testing
Every new ECS System must be accompanied by end-to-end (E2E) testing using Go's built-in `testing` package. This ensures the system functions correctly in isolation and alongside other systems within the `arche-go` ECS environment.

## 2. ECS Filter Usage
All logic processing entities must enforce strict **Arche-Go filter usage**.
- Ensure filters accurately query for only the necessary components.
- This prevents the processing of "Zombie Entities" or applying logic to objects missing critical dependencies, avoiding panics and maintaining simulation integrity.

## 3. Pull Request Rules
Every PR submitted must satisfy the following criteria:
- **Updated Test Suite**: Includes tests covering new or updated logic. Tests must pass locally.
- **Performance Profiling**: If the change involves core simulation loops, data structure adjustments, or complex math, a performance profile (pprof) check must be included to verify no regressions in cache efficiency or execution time occurred.
