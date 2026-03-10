# Phase 7: The Cognitive Engine (String Interning, Language, & Beliefs)

_Objective: Propagate ideas, language mutations, and gossip physically through the map, treating data as an infectious pathogen while strictly controlling RAM allocation via String Interning._

## 7.1 Secret Registry (String Interning)

- **The RAM Trap:** 10,000 duplicated strings of "The King is dead" will fragment memory and spike garbage collection.
- **The Registry:** Implement a global map `SecretRegistry {map[uint32]string}`. The actual string text is stored precisely once.
- **`SecretComponent`:** Entities hold arrays of `{SecretID uint32, OriginID uint64, Virality uint8}`.

## 7.2 Information Leakage (GossipDistributionSystem)

- **Physics of Gossip:** Runs on a slower tick execution (e.g., every 10 Ticks).
- **Vector check:** Iterates ECS seeking proximity overlaps. If Entity A overlaps Entity B:
  - Calculate mathematical chance: `Identity.Gossip` trait modifier \* `Secret.Virality`.
  - If RNG (via Seed) passes -> Inject `SecretID` into neighbor's `MemoryComponent` buffer.

## 7.3 Linguistic Drift

- **`CultureComponent`**: `{LanguageID uint16}`.
- **`LanguageDriftSystem`**: Iterates entities. If an entity logs no `MemoryEvent` interacting with its parent `LanguageID` origin for 10,000 consecutive ticks -> generate new `LanguageID` (Dialect formation).
- **Pidgin Creation:** Conversely, if two distinct `LanguageID`s interact at high volume for 50,000 ticks, mathematically assign both to a new, shared `PidginLanguageID`.

## 7.4 Translation Penalties & Silent Hooks

- **The Gossip Barrier:** When the `GossipDistributionSystem` attempts to pass a `SecretID`, it evaluates the `LanguageID` of both nodes.
- If mismatched, apply `TranslationPenalty` (drastic reduction to `Virality` or total failure).
- **Silent Hooks:** Even if language fails, physical trades can occur. Generate `SparseHookGraph` updates ("Silent Hooks" = owing a favor without understanding the language context).

## 7.5 Ideological Infection (The Memetic Engine)

- **`BeliefComponent`**: `map[BeliefID]int weight`. Tracks adherence to specific religions/cultural dogmas.
- **Idea Virus Logic:** Update `GossipDistributionSystem` to check if passed secrets contain ideological metadata flags.
- If a `SecretID` carrying a `BeliefID` successfully executes against a target NPC, linearly modify their `BeliefComponent` array, simulating forced or organic ideological conversion based on proximity.
