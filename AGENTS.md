# AI Agents Master Document (Antigravity, Gemini, Jules)

This document replaces the centralized NotebookLM and Cursor knowledge base for this project. All project vision and technical decisions are now managed directly in the repository so that local AI agents—Antigravity, Gemini, and Jules—have direct, synchronized access.

## The Single Source of Truth

The definitive vision and technical blueprint for **Boundless Sovereigns** has been flawlessly consolidated into a comprehensive set of documentation files. All AI agents must reference these files to maintain context and ensure structural priorities:

1. **`docs/vision.md`**: The High Concept and the 6 Pillars of the Total Simulation Closed Loop.
2. **`docs/mechanics.md`**: The granular, unabstracted systems powering the 6 Pillars (Geography, Infrastructure, Social Fabric, Cognitive/Linguistic Engine, Justice/Power, and Logistics/Entropy).
3. **`docs/architecture.md`**: The specific Go, ECS (`arche-go`), and Hybrid Rendering (`Ebitengine` / `raylib-go`) logic required to make the simulation run at 60+ TPS.
4. **`docs/roadmap.md`**: The definitive 17-phase development pipeline.
5. **`docs/roadmap/*.md`**: The individual phase-by-phase implementation blueprints (e.g., `01_foundation.md`, `02_geography.md`).
6. **`docs/internal/DEV_LOG.md`**: The active developer log tracking current tasks, ECS Component IDs, and the singleton Seed strategy.
7. **`docs/internal/CODING_STANDARDS.md`**: The project's rules for mandatory E2E testing, Arche-Go filter query constraints, and PR profiling checks.

These files must be treated as the ultimate instruction set and single source of truth.

## Core Agents

1. **Antigravity:** The powerful agentic AI coding assistant meant for executing complex technical plans, deeply analyzing code, and implementing architectural patterns like Go's ECS and Data-Oriented Design.
2. **Gemini:** The underlying AI model and orchestrator providing core reasoning, logic, and adherence to the frameworks established in `GEMINI.md`.
3. **Jules:** Partner AI agent integrated into the project workflow, working alongside Gemini and Antigravity.

## Expected Workflow (Vibecoding)

- **Single Source of Truth:** As the project evolves, the three `docs/` files MUST be updated. They serve as the definitive representation of the project, replacing external knowledge bases.
- **Continuous Synchronization:** AI agents will actively read `mechanics.md` and `architecture.md` to refactor code and generate logic with those specific guardrails in mind (Physicality, Culture, Justice, Entropy).
- **Adherence:** All actions conform to the project's Go and ECS-driven architecture designed to build a bottom-up grand strategy game. Agents must maintain the structural priorities (parallel processing, procedural generation, cache efficiency) defined in the architecture.
