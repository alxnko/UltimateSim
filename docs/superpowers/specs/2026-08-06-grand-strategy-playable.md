# Spec: Grand Strategy + Actually Playable (2026-08-06)

User vision: **EU5 + Victoria 3 + CK3 + RimWorld, but you play one character** —
and may play as ANY character. Fully working, playable, interesting, optimized,
polished. Language stays **Go** (decided after Rust discussion; evidence: tick
≈3ms, render GPU-bound, perf is fine).

## Probe findings (2026-08-06, headful play-test)

- F1 **CRITICAL**: "No body possessed" — `initPlayer()` runs before first tick;
  genesis NPCSpawner runs *in* the first tick → possession always fails → all
  input dead. This is the "fully unplayable" root cause.
- F2 Worldgen is fast (<4s for 1024×1024) — loading is not a problem.
- F3 Minimap reads as noise → map has no macro structure (no continents/oceans).
- F4 No NPCs near default camera (512,512); player sees an empty world.
- F5 No free camera pan — camera only follows the (nonexistent) player.
- F6 Perf: tick ≈3ms avg. DiseaseVectorSystem = 58% of sim time (top hog).
- F7 One silent process exit observed at menu (first run only, not reproduced).

## P0 — Playable core (BLOCKER, nothing ships without this)

- [ ] P0.1 Warmup: after load, run ~240 ticks so genesis/villages/jobs settle,
      with visible progress. Skip warmup when a possession already exists
      (loaded save).
- [ ] P0.2 Character Select screen: list real candidates (wealthy, young,
      stable villagers; name/age/job/city/stats) + "Surprise Me". Pick →
      possess, starter kit, camera snap. This IS "play as any character" at start.
- [ ] P0.3 "Play As" verb on any NPC (context menu): switch bodies anytime.
      No starter kit on switch — you inherit their life as-is.
- [ ] P0.4 Free camera: MMB-drag pan + arrow keys + minimap click-to-jump;
      F or moving (WASD) re-locks follow. Viewport rectangle on minimap.
- [ ] P0.5 Camera snaps to character on possess (no cross-map lerp).

## P1 — World that looks real

- [ ] P1.1 Worldgen v2 macro structure: continents (2–5), oceans between them,
      seas/bays, lakes, island chains, mountain ranges, coherent biome belts by
      latitude+moisture. Seed-variable (different seed → different world).
- [ ] P1.2 Minimap must read as geography (continents visible at 128px).
- [ ] P1.3 Deterministic: same seed → same world (existing law, keep).
- [ ] P1.4 Settlement placement respects new geography (coasts favored, no
      ocean spawns) — existing spawn rules must still work.

## P2 — Grand strategy layer

- [ ] P2.1 Time controls: speeds 1/2/4/8 ticks-per-frame (keys 1–4), pause
      (Space). HUD shows speed. TickManager.Speed added.
- [ ] P2.2 Diplomacy: country relations (opinion, alliance, war, truce),
      AI declares war/peace by opinion+power; diplomacy panel as sovereign
      (improve relations, ally, declare war, sue for peace).
- [ ] P2.3 Dynasty: marriage action (spouse link), dynasty panel (spouse,
      children, living kin, heirs in line), prestige surfaced.
- [ ] P2.4 Intrigue: plots (seize leadership / assassinate rival), progress
      over time, discovery risk vs plot power, resolution effects.
- [ ] P2.5 Council: sovereign appoints Steward/Marshal/Diplomat/Spymaster from
      city NPCs; each gives a periodic national bonus.
- [ ] P2.6 Economy: market panel (prices per good per city), tax rate control
      for rulers, treasury income surfaced; trade-route income for cities.
- [ ] P2.7 Strategic country view: below zoom threshold, click city →
      country info (ruler, treasury, relations at minimum).
- [ ] P2.8 Save/load covers every new component (existing law: full parity).

## P1.5 — UI/UX v2 (user verdict: current UI "not intuitive, looks weird")

- [ ] U1 Input works: mouse clicks MUST land on buttons (bug: clicks do
      nothing on character select — confirmed by user with real mouse).
- [ ] U2 Readable typography: crisp scalable font (gofont/goregular via
      text/v2, no asset files), size tiers (title/heading/body/small).
- [ ] U3 Design system: one theme (spacing grid, panel chrome with title bar +
      close X, button/hover/active states, consistent paddings).
- [ ] U4 Widget kit built once, reused everywhere: Panel, Tabs, Button,
      IconButton, Bar, SearchableList (text filter box + scroll), Table
      (sortable columns), Tooltip (hover, delayed), Modal stack (Esc closes
      top; one manager owns z-order).
- [ ] U5 Game chrome: bottom tab bar (RimWorld-style) — Character, Dynasty,
      Diplomacy, Economy, Laws, Military, Chronicle, Menu. HUD top: date/season,
      speed controls (clickable + keys), gold, ambition tracker.
- [ ] U6 Keyboard everywhere: arrows+Enter navigate lists, Esc closes, hotkey
      hints shown on buttons; number keys = speed.
- [ ] U7 Searches + filters on all big rosters (characters, cities, countries,
      laws, market goods) via SearchableList.
- [ ] U8 Tooltips explain every stat/bar/lens (teach the rules in-game).
- [ ] U9 Notifications: stacked toasts, non-overlapping, click-to-focus event.
- [ ] U10 Onboarding: first-run hint strip (move/attack/menu/build/goals).

## P4 — GAMEPLAY LOOP (user verdict on first playtest: "gameplay is kinda
## awful" — walking around is not a game; the sim must become a STORY)

- [ ] G1 Interactive events engine (the CK3 heart): sim-state-driven popup
      events with 2-3 choices and real consequences. Generators at minimum:
      marriage proposal, plot invitation, ruler tax demand, rival insult,
      bandit shakedown (pay/fight/flee), job offer, festival, ruler died /
      succession, war declared/peace. Effects hit gold, opinion hooks,
      legitimacy, jobs, health. Deterministic (seeded RNG + sim state).
- [ ] G2 Drama surfaced: wars, coups, deaths of rulers, plagues, famines pop
      as toasts + chronicle entries with city/actor names. The world must feel
      alive WITHOUT opening panels.
- [ ] G3 Progression ladder always visible: HUD shows current rank + concrete
      next step ("Win 3 more friends to claim leadership of Greenfork").
- [ ] G4 Villages LOOK like villages: genesis places houses/workshop/shrine
      structures around each center; town square feel.
- [ ] G5 Game-feel defaults: start at 2x, first ambition auto-granted and
      tracked on HUD, event popup pauses the sim (CK-style), attack/interact
      feedback (flash + floater already exist — verify wired).
- [ ] G6 An event every ~30-90s of play at 1x in a living city; never a dead
      stretch over 3 minutes.

## P5 — LIVING WORLD + BUILDING (user: "ai does almost nothing, building is
## not good, hard and almost impossible to even play")

- [ ] L1 Visible daily life: employed citizens walk to job anchors (farm
      structure/fields for farmers, forest edge for lumberjacks, workshop for
      artisans, patrol ring for guards), WORK there visibly, and return to the
      village center in the evening (Calendar phase). WanderSystem stays only
      for the unemployed/wilderness. NPCs must look busy at a glance.
- [ ] L2 Building that feels good: ghost footprint preview under cursor in
      build mode (green=valid, red=invalid, cost readout), sites show an
      overhead progress bar, village builders auto-work player sites (verify
      ConstructionSystem picks them up) with a toast when one starts/finishes.
      Player hammering (E) stays as optional speed-up.
- [ ] L3 Overhead info at action zoom: name plates on hover, health bar over
      damaged NPCs, job-colored outfits already exist — verify readable.
- [ ] L4 Onboarding sequence: scripted first-minute toasts (move → talk →
      goals → build → claim leadership) that each dismiss on completion.

## P3 — Polish + perf

- [ ] P3.1 DiseaseVectorSystem optimization (58% of tick) — target <20%.
- [ ] P3.2 UI/UX pass: readable fonts everywhere, consistent panels, hotkey
      help (F1 or ?), tooltips on HUD bars, notifications not overlapping.
- [ ] P3.3 FPS/TPS debug readout (toggle).
- [ ] P3.4 Window resize support (Layout follows outside size).
- [ ] P3.5 Delete stale raylib.dll from repo root.

## Laws (unchanged from repo standards)

- Deterministic sim: no map iteration order deps, no time/rand in logic paths.
- DOD: flat structs, explicit padding, size tests for new components.
- arche queries: iterate to exhaustion OR break+Close, never both.
- Every new system: deterministic test, -count=2 clean.
- Save schema parity for all new components.
- Macro/micro parity law per CODING_STANDARDS.md.

## Acceptance (the review gate checks THESE)

1. Fresh run: menu → Enter → warmup progress → character select → pick →
   you SEE your character among NPCs and WASD moves them. (Kills F1.)
2. Minimap shows continents and oceans, not noise. (Kills F3.)
3. MMB-drag pans the world; minimap click jumps. (Kills F5.)
4. Right-click any NPC → "Play As" works mid-game.
5. Keys 1–4 change sim speed visibly; Space pauses.
6. As a city ruler: diplomacy panel opens, war can be declared on another
   country, relations visible.
7. Marriage + dynasty panel show a real family; plots can be started and
   resolve.
8. Market panel shows real prices; ruler can set tax rate.
9. Full gate green: go vet, build, test ./... -count=2, soak+determinism.
10. Save → load → identical playable state including new components.
