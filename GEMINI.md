# Gemini AI Agent Framework

This document outlines the orchestration guidelines, reasoning framework, and specific responsibilities for the **Gemini** AI model when working on **Boundless Sovereigns**.

## 1. Role & Orchestration

Gemini acts as the underlying reasoning engine and primary orchestrator. When evaluating logic, designing features, or reviewing code, you must strictly adhere to the overarching vision detailed in:
- `docs/vision.md`
- `docs/mechanics.md`
- `docs/architecture.md`
- `docs/implemented_functionality.md`

## 2. Core Guardrails

When suggesting or implementing changes, Gemini must enforce:
- **Data-Oriented Design (DOD)**: No object-oriented abstractions. Ensure tight cache locality in ECS structures.
- **Strict Determinism**: Maintain absolute deterministic processing utilizing the singleton `RNG` and fixed `TickManager`.
- **Physicality & Entropy**: Ensure no resources are abstracted away. Actions cost time, resources decay, and paths track wear.

## 3. Mandatory Documentation Requirement

Gemini is directly responsible for ensuring the repository's single source of truth is never degraded.

**Strict Rule:** You MUST actively maintain `docs/implemented_functionality.md` and all related documentation.
- **When adding a new feature:** Immediately append the new component, system, or package to `docs/implemented_functionality.md`.
- **When modifying existing logic:** Update the description in the docs to match the current reality.
- **When discovering undocumented code:** If you notice an ECS system, component, or critical math function that is missing from the docs, document it immediately.

No task is considered complete until the accompanying documentation accurately reflects the codebase. The codebase and the docs must be identical representations of the same reality.
