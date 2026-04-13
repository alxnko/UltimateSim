# Phase 45: Genealogy, Mutation & Bloodline Curses

## Overview
Biology extends across centuries. Bloodlines degrade, mutate, and carry the sins of their ancestors.

## Architecture & DOD Implementation
- **Deep Genealogy Tree**: The engine tracks dominant and recessive alleles inside `GeneticsComponent` over thousands of generations.
- **Inbreeding Depression**: If nobles repeatedly marry within their own `FamilyID` to consolidate wealth, recessive genetic defects (hemophilia, madness) physically manifest in heirs.
- **Inheritable Curses**: Powerful esoteric magic (Phase 30) can attach a `CurseMarker` to an entity's DNA, meaning every descendant for 10 generations will be born with crippling `SanityComponent` debuffs or physical mutations (e.g., horns).