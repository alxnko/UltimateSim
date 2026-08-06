# Plan: Grand Strategy build waves (2026-08-06)

Governing spec: `docs/superpowers/specs/2026-08-06-grand-strategy-playable.md`.
Branch: `feature/grand-strategy` (worktree `.claude/worktrees/grand-strategy`).
Base: `main` @ 1f15178. P0 (input fix + character select + free camera +
speed) landed in 434566f, verified by headful play-probe screenshots.

## Delivery model

Parallel subagents build IN this worktree on **disjoint files**; the
orchestrator (main session) owns all shared files, wiring, commits, and the
merge gate. Agents never run git. Each agent builds + tests its own package
before reporting, and reports its exported API surface.

## Wave 1 — foundations (parallel, disjoint)

| # | Agent | Owns (only these) | Delivers |
|---|-------|-------------------|----------|
| 1 | worldgen-v2 | `internal/engine/map_generation*.go` + its tests | Continents/oceans/seas/lakes/islands/ranges; latitude+moisture biomes; seed-variable; deterministic |
| 2 | ui-font | `internal/ui/theme.go` | gofont/goregular via text/v2; size tiers Title/Heading/Body/Small; MeasureText accurate; DrawText back-compat |
| 3 | ui-widgets | `internal/ui/widgets.go`, new `internal/ui/widgets_kit.go` | Tabs, SearchableList (filter box), Table (sortable), Tooltip, Modal frame; keyboard nav; input via BeginUIFrame snapshot |
| 4 | diplomacy | new `internal/components/diplomacy.go`, `internal/systems/diplomacy.go` + tests | CountryRelation (opinion/alliance/war/truce), DiplomacySystem AI, player actions |
| 5 | dynasty+intrigue | new `internal/components/dynasty.go`, `internal/systems/dynasty.go`, `internal/systems/plots.go`, `internal/systems/council.go` + tests, `internal/systems/names.go` | Marriage, dynasty queries, plots, council seats + bonuses, deterministic name generator (kills "NPC-1") |
| 6 | economy | new `internal/components/economy.go`, `internal/systems/economy_actions.go` + tests | Tax policy + collection into treasury, trade-route income, market price snapshot API for UI |

## Wave 2 — surface (after wave 1)

| # | Agent | Owns | Delivers |
|---|-------|------|----------|
| 7 | ui-panels | new `internal/ui/panel_diplomacy.go`, `panel_dynasty.go`, `panel_council.go`, `panel_market.go`, `chrome.go` | Bottom tab bar, all strategy panels, searches/filters, tooltips, hint strip |
| 8 | integration (orchestrator) | `cmd/game/main.go`, `internal/engine/save_load.go`, `internal/ui/state_playing.go`, `internal/ui/player_context.go` | Registration, save parity, hotkeys, tab-bar wiring, warmup tuning |
| 9 | e2e-tests | `cmd/game/soak_test.go` + new e2e | Soak covers new systems; determinism -count=2 |

## Wave 3 — gate

Adversarial review workflow (find→verify) against the SPEC, fixes, full gate
(vet/build/test -count=2/soak), headful play-probe screenshots re-run, then
shipping-to-merge steps 4–8.

## Shared-file law

`basic.go`, `main.go`, `save_load.go`, `state_playing.go`, `player_context.go`,
`context_menu.go`, `hud.go` belong to the orchestrator. Agents needing a hook
there RETURN the needed snippet in their report instead of editing.

## Standing laws for every agent

- Deterministic: no wall-clock/map-order/randomness outside seeded RNG.
- arche queries: iterate to exhaustion OR break+Close — never Close after
  exhaustion (double-unlock panic).
- DOD structs, explicit padding, unsafe.Sizeof size test for new components.
- Tests must pass `-count=2`; system tests build a real World and tick it.
- UI click handling ONLY through BeginUIFrame snapshot (consumeClickIn /
  TakeWorldClick) — never inpututil in Draw.
- Reused mapping data: reader results at
  `...\workflows\wf_92d34406-d22\res1.json` (dynasty), `res2.json` (UI),
  `res3.json` (economy).
