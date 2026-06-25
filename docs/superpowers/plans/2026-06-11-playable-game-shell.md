# Playable Game Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete presentation + interaction layer turning UltimateSim's simulation backend into a playable Streets of Rogue / RimWorld-style game per `docs/superpowers/specs/2026-06-11-playable-game-design.md`.

**Architecture:** All UI/world mutation happens on the single Ebiten goroutine between ticks (StatePlaying.Update: input → mutations → Tick). New gameplay logic lives in DOD ECS systems (PlayerOrderSystem, AmbitionSystem, StructureEffectSystem) registered in main.go. Rendering uses a procedural SpriteCache; UI uses an immediate-mode widget toolkit. 39 dormant simulation systems get registered.

**Tech Stack:** Go 1.21+, Ebiten v2, arche-go ECS, golang.org/x/image basicfont, modernc.org/sqlite.

**Conventions (apply to every task):**
- DOD: flat structs, integer IDs, padding comments, no pointers in components.
- ECS mutations deferred outside `query.Next()` loops via pre-allocated slices.
- Every new system gets a deterministic E2E test (`-count=2` safe).
- Build check: `$env:CGO_ENABLED=0; go build ./...` then `go test ./...`.
- Commit after each task: `feat(shell): <task>`.

---

### Task 1: Camera & coordinate transforms
**Files:** Create `internal/ui/camera.go`, `internal/ui/camera_test.go`. Modify `internal/ui/state_playing.go` (use Camera struct).
- `type Camera struct { X, Y float64 // world tile coords center; Zoom float64 }`
- Methods: `WorldToScreen(wx, wy float32, sw, sh int) (float64, float64)`, `ScreenToWorld(sx, sy int, sw, sh int) (float32, float32)`, `TileSize() float64` (=16*Zoom), `ClampZoom()` (0.25..4.0), `HandleWheel()` (ebiten wheel → zoom step ×1.15, isolated from tests).
- Replace StatePlaying CamX/CamY/Zoom fields with `Cam Camera`; wheel zoom in Update.
- Tests: round-trip WorldToScreen∘ScreenToWorld within epsilon at zooms 0.25/1/4; clamp bounds.
- Steps: write tests → fail → implement → pass → wire into state_playing → build → commit.

### Task 2: UI widget toolkit
**Files:** Create `internal/ui/widgets.go`, `internal/ui/theme.go`, `internal/ui/widgets_test.go`.
- `theme.go`: color constants (PanelBG RGBA{25,28,36,230}, PanelBorder {90,95,110,255}, TextCol {220,220,225,255}, AccentCol {230,180,60,255}, BarRed/Green/Blue/Purple), `FontFace` = basicfont.Face7x13, helpers `DrawText(dst, s, x, y int, clr color.Color)`, `MeasureText(s string) int`.
- `widgets.go` immediate-mode (no retained state; hover/click computed from `ebiten.CursorPosition` + `inpututil.IsMouseButtonJustPressed`):
  - `DrawPanel(dst, x, y, w, h)` (bg + 1px border)
  - `Button(dst, label string, x, y, w, h int) bool` (returns clicked; hover highlight)
  - `Bar(dst, x, y, w, h int, frac float32, fg color.Color, label string)`
  - `type ContextMenu struct { X, Y int; Items []string; Visible bool }` with `Update() (int, bool)` (returns chosen index, closed) and `Draw(dst)`
  - `type ScrollList struct { X,Y,W,H, Offset int }` with `Visible(rowH, n int) (from, to int)` + wheel scroll when hovered
  - `PointIn(px, py, x, y, w, h int) bool`
- Tests: PointIn, ScrollList.Visible math, ContextMenu index math (pure logic only).

### Task 3: Procedural sprite cache
**Files:** Create `internal/render/sprites.go`, `internal/render/sprites_test.go`.
- `type SpriteCache struct { chars map[uint64]*ebiten.Image; statics map[string]*ebiten.Image }`, `NewSpriteCache()`.
- `CharSprite(id uint64, genome *components.GenomeComponent, jobID uint8, frame int) *ebiten.Image` — 16x16: 6x8 body rect (clothing color = job palette table[15 jobs]), 6x5 head (skin tone = 3-entry table indexed by genome.Beauty%3), 6x2 hair (color from genome.Dominant bits), 2px feet offset on frame 1 (walk bob). Cache key = id^(frame<<63 style bit mix); deterministic — derive all colors from (id, genome, jobID), never RNG.
- `StaticSprite(kind string) *ebiten.Image` for kinds: `house, workshop, storehouse, shrine, site, village, village_rich, capital, ruin, ship, caravan, item, coin, corpse, ledger, workbench, ghost_ok, ghost_bad` — each a distinct 16x16 (buildings 24x24 for village/capital) drawn with rect fills (roof triangle approximated by stacked rows).
- Tests: same inputs → identical pixels (compare `image.At` samples); all kinds non-nil; distinct kinds differ.

### Task 4: World rendering overhaul + real HUD + minimap + notifications
**Files:** Create `internal/ui/render_world.go`, `internal/ui/hud.go`, `internal/ui/notifications.go`, `internal/ui/minimap.go`, `internal/ui/notifications_test.go`. Modify `internal/ui/state_playing.go` (Draw delegates), `cmd/game/main.go` (window 1280x720, Layout).
- `render_world.go`: passes — tiles (existing loop, camera-based); entity pass querying once each: structures (`StructureComponent`+Position → StaticSprite by type), construction sites (site sprite + mini progress bar), workbenches, villages (rich variant when Storage total>2000; ruin overlay), capitals, ships, caravans, items, coins, ledgers, corpses; then characters via `CharSprite` (frame = (tick/20+id)%2 when |vel|>0), possessed gets 18x18 accent ring; selection ring on `pc.Selected`.
- `hud.go`: `DrawHUD(screen, pc *PlayerContext)` bottom bar 64px: Blood/Stamina/Pain/Consciousness bars + Food/Rest/Wealth + stress bar (read possessed entity components, tolerate missing), name/job/city/rank text, season+day (from Calendar via exposed getter), speed buttons `⏸ ▶ ⏩` toggling TickManager.IsPaused/IsFastForward, tool buttons Build/Orders/Ambitions/Char.
- `notifications.go`: `type Notifier struct{ items []Note }`, `Push(text string, tick uint64)`, `Draw` top-right stack, fade after 600 ticks, max 8. Test Push/expiry logic.
- `minimap.go`: 128x128 image built once from biome colors at grid/8 resolution; player dot; village dots (refresh every 300 ticks).
- Manual check: `go build` + screenshot run.

### Task 5: PlayerContext, selection & inspector panel
**Files:** Create `internal/ui/player_context.go`, `internal/ui/inspector.go`. Modify `internal/ui/state_playing.go`.
- `PlayerContext`: `World/TM/Grid/HookGraph` refs, `Cam`, `Selected ecs.Entity`, `SelectedValid bool`, `Mode uint8` (ModeNormal/ModeBuild/ModeOrder consts), `BuildType uint8`, `Notifier`, `Sprites`, `PlayerEnt ecs.Entity`, helper `PossessedEntity() (ecs.Entity, bool)` (query Possessed), `EntityAt(wx, wy float32, radius float32) (ecs.Entity, bool)` (nearest Position entity within radius; prefer NPCs over structures).
- Left-click (ModeNormal) on entity within 0.8 tile of cursor → select (not attack — attack only on NPC targets via A-key… **No**: per spec left-click attack). Resolution: left-click on an NPC = attack; left-click on non-NPC entity/tile = select; selection of NPCs via right-click→Inspect or holding Shift+click. Keep: Shift+LClick always select.
- `inspector.go`: right panel 300px when SelectedValid. Tab bar: NPC → Bio (name/age/traits decode/genome bars), Needs (needs+vitals+sanity bars), Social (last 8 Memory events decoded to text, secret count, top hooks in/out via HookGraph), Politics (affiliation IDs, crime marker, legitimacy/loyalty if present). Village/structure → Overview (population, storage, treasury, integrity), Market (prices/wage), Laws (read-only here). Tile fallback: biome name, resources, foot traffic.
- Trait/interaction decode helpers in `internal/ui/format.go` (`TraitNames(bitmask) []string`, `InteractionName(uint8) string`, `JobName(uint8) string`, `StructureName(uint32) string`) with unit tests.

### Task 6: Attack verb + combat feedback
**Files:** Modify `internal/systems/player_input.go` (full rewrite), create `internal/systems/player_input_test.go`, `internal/ui/effects.go`. Modify `cmd/game/main.go` (constructor args).
- Rewrite PlayerInputSystem: `NewPlayerInputSystem(world, pc *InputBridge)` where `InputBridge` (in `internal/systems/input_bridge.go`) is a plain struct the UI fills each frame: `AttackAt (x,y float32, valid bool)`, `MoveDir (x,y float32)`, cleared after consumption. UI computes world coords (camera lives in UI layer; systems stay UI-agnostic). WASD remains read directly from ebiten keys (acceptable; bridge carries mouse intents).
- Attack: find nearest entity with VitalsComponent+Identity within 1.5 of AttackAt (excluding self); if found and player stamina > 5 → add `CombatMarker{TargetID}` to player (replace existing), push feedback event to bridge (`Events []BridgeEvent` — {Kind, X, Y, Value}).
- Attack cooldown: `LastAttackTick`, 30-tick cooldown.
- `effects.go`: floating damage texts + hit flashes consumed from bridge events; CombatSystem unchanged (damage numbers approximated from event).
- E2E test: world with 2 NPCs; bridge AttackAt near target; Update; assert CombatMarker on player with correct TargetID; assert cooldown blocks immediate second marker; deterministic ×2.

### Task 7: Context menu interactions — Talk, Gift, Threaten, Rumor
**Files:** Create `internal/ui/dialog.go`, `internal/systems/social_actions.go`, `internal/systems/social_actions_test.go`. Modify `internal/ui/state_playing.go` (right-click → ContextMenu).
- Right-click: EntityAt → menu items by archetype: NPC [Talk, Trade, Attack, Order…(ruler+subordinate only), Inspect], Village [Inspect, Laws…(ruler)], Item/Coin [Pick up], ground [Move here (sets player Path via direct velocity target — simple: walk toward), Build here…].
- `social_actions.go` pure functions (called by UI between ticks, world-mutating, unit-testable):
  - `Chat(world, hooks, playerEnt, targetEnt, tick)` → +1 hook both directions, MemoryEvent InteractionGossip both, returns target reply summary (name + random-free: top need name).
  - `GiveGift(world, hooks, player, target, amount float32) error` → wealth check, transfer Needs.Wealth, +3 hook target→player.
  - `Threaten(world, hooks, player, target, tick)` → −2 hook, target Needs.Safety −10; if inside jurisdiction banning Assault → add CrimeMarker to player (reuse justice helpers by replicating its jurisdiction scan inline).
  - `ShareRumor(world, player, target)` → copy random secret w/ existing language-penalty rule (10% on mismatch), returns success.
- `dialog.go`: Talk window — portrait sprite, hook standing, buttons Chat/Gift 10/Rumor/Threaten/Close, response line.
- E2E tests for all four functions (hook deltas, wealth transfer, crime marker in/out of jurisdiction, secret copy determinism with seeded RNG).

### Task 8: Trade & pickup
**Files:** Create `internal/ui/trade.go`, `internal/systems/trade_actions.go`, `internal/systems/trade_actions_test.go`.
- `EnsurePlayerStorage(world, ent)` adds zeroed StorageComponent if missing.
- `TradeBuy/TradeSell(world, playerEnt, villageEnt, item uint8, qty uint32) error` — price from village MarketComponent (wood/stone/iron/food), wealth from player Needs.Wealth, stock checks both sides.
- `PickUp(world, playerEnt, itemEnt) (string, error)` — CoinEntity → wealth += CurrencyComponent.Value; ItemEntity+LegendComponent → EquipmentComponent (equip, replace lesser prestige); resource piles n/a. Defer entity removal to caller list (UI removes after call — between ticks, safe).
- `trade.go`: window listing 4 goods, price, village stock, player stock, +/- buttons buy/sell 1/10.
- E2E tests: buy/sell math incl. insufficient funds/stock; pickup coin and legend equip-replacement.

### Task 9: Construction pipeline activation + structure types + effects
**Files:** Modify `internal/components/basic.go` (consts StructureWorkshop=2, StructureStorehouse=3, StructureShrine=4; add `SiteType uint32` field to ConstructionSiteComponent), `internal/systems/construction.go` (use SiteType when completing; default House), `cmd/game/main.go` (register ConstructionSystem + CraftingSystem in PhaseResolution). Create `internal/systems/structure_effect.go`, `internal/systems/structure_effect_test.go`.
- `StructureEffectSystem` (throttled, every 200 ticks): House → NPCs within 3 tiles get Needs.Rest +5 (cap 100); Shrine → carries `OwnerBeliefID` (extend StructureComponent with `DataA uint32` field, padded) and pushes Belief weight +1 to NPCs within 5; Workshop → on completion ConstructionSystem spawns co-located WorkbenchComponent entity; Storehouse → village Storage cap conceptual: +Food 1/200t generation guard? **No invented mechanics:** Storehouse instead halves SpoilageSystem decay — implement as: villages with a Storehouse within 3 tiles tagged via map the SpoilageSystem already…— too invasive. Simplest real effect: Storehouse adds +500 one-time Wood/Stone/Iron/Food capacity is fictional; instead Storehouse → DemographicsComponent.PeakPopulation +100 (mirrors house logic, documented as logistics capacity). Keep effects: House rest aura, Shrine belief aura, Workshop→Workbench, Storehouse pop-capacity.
- E2E tests: each effect deterministic; construction completes per SiteType.

### Task 10: Build mode (player construction verb)
**Files:** Create `internal/ui/build_mode.go`, `internal/systems/build_actions.go`, `internal/systems/build_actions_test.go`.
- Costs table: House {Wood 50, Stone 50}, Workshop {Wood 80, Stone 40}, Storehouse {Wood 120, Stone 80}, Shrine {Stone 100}.
- `CanPlace(grid, wx, wy, world) error` — in bounds, biome != Ocean, no Structure/site/village within 1.0 tile.
- `PlaceSite(world, grid, playerEnt, buildType uint8, wx, wy float32) error` — validates, deducts from player StorageComponent (must carry materials; buy via trade first), spawns entity Position+Affiliation(player CityID)+ConstructionSiteComponent{SiteType, requirements = cost, MaxProgress 100, WoodGathered=cost (player supplies upfront), StoneGathered=cost}. Player-funded sites start fully stocked; builders only hammer progress. Player can hammer: interact (E near site) → Progress +1 per press w/ stamina −1.
- `build_mode.go`: B toggles; structure picker strip; ghost sprite at cursor (ok/bad tint); click places; Esc exits.
- E2E tests: CanPlace cases (ocean/overlap/bounds), PlaceSite cost deduction + component values, E-hammer progress.

### Task 11: Rank detection & Claim Leadership
**Files:** Create `internal/systems/rank.go`, `internal/systems/rank_test.go`. Modify `internal/ui/hud.go` (rank display), context menu (village → Claim Leadership when eligible).
- `GetRank(world, hooks, ent) (rank uint8, cityID, countryID uint32)` — RankCitizen/RankRuler/RankSovereign consts; Ruler = has AdministrationMarker; Sovereign = ruler AND city has CapitalComponent+CountryComponent (city entity lookup by Affiliation.CityID against village Identity-as-city mapping used elsewhere: villages' Affiliation.CityID).
- `CanClaimLeadership(world, hooks, playerEnt) bool` — player city != 0 && (no current ruler in city || player positive incoming hooks > ruler's +1 bias).
- `ClaimLeadership(world, hooks, playerEnt) error` — remove AdministrationMarker from old ruler, add to player, add LegitimacyComponent{Score 50} + LoyaltyComponent if absent. (LeadershipEmergenceSystem re-runs every 500 ticks — player keeps rank by maintaining hooks; that's the game.)
- E2E tests: claim transfers marker; rank detection for all three tiers; deterministic.

### Task 12: PlayerOrderComponent + PlayerOrderSystem
**Files:** Modify `internal/components/basic.go` (add component + order consts). Create `internal/systems/player_order.go`, `internal/systems/player_order_test.go`. Modify `cmd/game/main.go` (register PhaseAI).
- `const (OrderMove uint8 = 1; OrderAssignJob = 2; OrderAttack = 3; OrderBuildAt = 4; OrderFollow = 5)`
- `type PlayerOrderComponent struct { TargetID uint64; X, Y float32; IssuerID uint64; OrderType uint8; JobID uint8; _ uint16; _ uint32 }` (32B).
- `PlayerOrderSystem` (PhaseAI): per ordered NPC — obedience check once on receipt? Per-spec on execution: roll-free deterministic check: `obeys = legitimacy(issuer city) + hookBalance(npc→issuer) >= loyaltyThreshold(LoyaltyComponent.Value, default 30)`; legitimacy read from issuer's LegitimacyComponent (default 25 if none). Disobey → remove order, MemoryEvent, −5 legitimacy, notification via bridge event.
  - Move/BuildAt: steer velocity toward X,Y (direct seek 1.5 speed, bypass wander) until within 1.0 → order removed (BuildAt also sets JobID=JobBuilder).
  - AssignJob: set JobComponent.JobID = order.JobID, remove order.
  - Attack: add CombatMarker{TargetID}, remove order.
  - Follow: seek issuer position, persists until countermanded.
  - Conflict guard: WanderSystem skips entities with PlayerOrderComponent (modify `wander.go` filter `.Without(orderID)`).
- E2E tests: obedience math table; each order type end-state; disobedience removes order and dents legitimacy; deterministic ×2.

### Task 13: Orders UI + Laws panel
**Files:** Create `internal/ui/panel_orders.go`, `internal/ui/panel_laws.go`. Modify state_playing (ModeOrder flow), context menu.
- Orders flow: ruler selects NPC of own city (Shift+click or via Order ctx item) → order bar appears [Move, Follow me, Attack…, Work as…, Build here…]; target-needing orders arm a click-target mode; issuing attaches PlayerOrderComponent (dedupe: replace existing).
- Laws panel (village selected + ruler): checkboxes toggling IllegalActionIDs bits (Assault/Theft/Murder/Esoteric/Gossip), Contraband bitmask toggles (Wood/Stone/Iron/Food), shows Corruption + Trauma read-only. Sovereign extras: Debasement +/- 0.05 steps (0..0.9) on CountryComponent, Declare War list of other countries → set WarTrackerComponent{TargetCountryID, Active}, Make Peace.
- Pure helpers in `internal/systems/law_actions.go` + tests: `ToggleLaw(jur *JurisdictionComponent, interaction uint8)`, `SetDebasement`, `DeclareWar(world, capitalEnt, targetCountryID) error`, `MakePeace`.

### Task 14: Strategic lenses & labels
**Files:** Create `internal/ui/lenses.go`, `internal/ui/lenses_test.go`. Modify `render_world.go` (zoom<0.5 → lens mode), hud (lens selector when zoomed out).
- `LensNone/LensPolitical/LensWealth/LensCrime/LensCulture` consts; keys F1–F4.
- Build per-frame (throttle: rebuild every 60 ticks) `lensData`: village positions + CityID/CountryID/storage totals/crime counts (CrimeMarker per city)/dominant LanguageID. `LensColor(lens uint8, seedID uint32) color.RGBA` — stable hash → hue table (test: same ID same color, different IDs differ).
- Strategic render: tiles desaturated; villages as 8px blocks colored by lens with influence circles (RadiusSquared); name labels (Identity.Name of village or "City <ID>"); character dots 2px; player gold dot.
- Tests: LensColor determinism; lens data aggregation from small world.

### Task 15: Wake dormant systems + soak test
**Files:** Modify `cmd/game/main.go`. Create `cmd/game/soak_test.go`.
- Register (PhaseResolution unless noted) with correct ctor args: Administration, Agriculture, BlackMarket, Casting, ClassWarfare, Conscription, Construction (done T9), Crafting (done T9), CurrencyExchange, Deforestation, EchoChamber, EconomicBloc, HolyWar, Inflation, JealousyVulnerability, JobMarket, LaborCrisis, Legitimacy, MaritimeLabor, MaritimeMigration, Medical, MentalBreak, Mercenary, Minting, NavalSpawning, Parasite, PenalLabor, PoliticalCoup, PortBinder?, Preacher, PriceNormalization, Propaganda, Refugee, RuinResettlement, Sanitation, Scapegoat, TraumaticTraditions, WarEconomy, Workplace, Xenophobia. Skip ClientPrediction/DeltaExtraction (network-only). Check each constructor signature before registering; respect phase hints in file comments.
- Soak test: build sim 128x128 grid, run 5000 ticks headless, assert no panic and tick time sane; run twice for determinism of TickManager.Ticks-observable aggregates (population count equality).
- Profile: one pprof-less timing print acceptable per CODING_STANDARDS (note in PR).

### Task 16: Ambitions
**Files:** Modify `internal/components/basic.go` (`Ambition` 16B struct + `AmbitionsComponent{Ambitions []Ambition; Offers []Ambition}`). Create `internal/systems/ambition.go`, `internal/systems/ambition_test.go`, `internal/ui/panel_ambitions.go`. Register (PhaseResolution, every 400 ticks).
- Types: AmbitionRuler (TargetID=cityID, done when rank≥Ruler of that city), AmbitionWealth (Goal=N, progress=wealth), AmbitionBuilder (Goal=N structures placed by player — count via system observing player-funded sites: increment counter field in AmbitionsComponent on completion notification from build_actions… simpler: Progress increments when PlaceSite succeeds, wired through pure function `RecordPlayerBuild(amb)`), AmbitionHeir (done when BirthSystem creates child with player FamilyID after accept tick — check family count delta), AmbitionFeud (TargetID = NPC with negative hook to player; done when target dead).
- Generation: from world state (nearest 2 cities, wealth*2 target, etc.), max 3 offers, max 3 active. Completion → Legacy.Prestige += 25, notification.
- Panel: offers (Accept/Dismiss), active w/ progress bars.
- E2E tests: generation determinism, each completion path, prestige award.

### Task 17: Death → heir loop
**Files:** Modify `internal/systems/death.go` (possessed detection → `engine.PlayerEvents` singleton struct `{PlayerDied bool; DeathCause uint8; DeathTick uint64}` in new `internal/engine/player_events.go`). Create `internal/ui/state_heir.go` (overlay within StatePlaying, not a StateManager state — simpler: modal in PlayerContext `HeirModal bool`), heir logic `internal/systems/heir.go` + `heir_test.go`.
- DeathSystem: while collecting toRemove, if entity has Possessed → set PlayerEvents (also strip Possessed so auto-possess failsafe doesn't fire — **remove the auto-possess failsafe** in state_playing; initial possession instead happens once at world start picking a young NPC).
- `FindHeirs(world, familyID uint32, excludeID uint64) []HeirInfo` (Entity, name, age, job, genome summary).
- Modal: "You died (<cause>). Choose your heir." list → click → `world.Add(heirEnt, possessedID)`, camera snap, notification "You live on as <name>". Empty family → Legacy screen modal: dynasty stats (prestige, structures built, ambitions completed) + buttons [Reincarnate (possess random newborn/young NPC), Main Menu].
- Initial possession: in StatePlaying first ready-frame pick NPC age<30 with family, add Possessed + EnsurePlayerStorage + AmbitionsComponent.
- E2E tests: FindHeirs filtering; death event set when possessed dies; possession transfer preserves Legacy inheritance (already in DeathSystem).

### Task 18: Pause menu, save/load UI, extended schema, main menu
**Files:** Create `internal/ui/pause_menu.go`. Modify `internal/ui/state_main_menu.go` (New Game / Continue / Quit + seed and size S(256)/M(512)/L(1024) pickers), `cmd/game/main.go` (plumb options), `internal/engine/save_load.go` (+tables), `internal/engine/save_load_test.go` (extend round-trip).
- Esc → pause modal: Resume / Save Game (→ `saves/world.db`, notification result) / Load Game (rebuild status from LoadWorld — restart StatePlaying with loaded TM; follow existing LoadWorld API) / Controls (static text) / Quit to Menu.
- Save schema additions (same mapped-struct pattern): EquipmentComponent, SanityComponent, TreasuryComponent, MarketComponent, LegitimacyComponent, LoyaltyComponent, JurisdictionComponent, StructureComponent, ConstructionSiteComponent, PlayerOrderComponent, AmbitionsComponent (JSON), CapitalComponent/CountryComponent, AdministrationMarker, CombatMarker, DemographicsComponent, WorkbenchComponent, CultureComponent.
- Round-trip test covers every new table.

### Task 19: Combat & game feel polish
**Files:** Modify `internal/ui/effects.go`, `internal/systems/player_input.go`, `internal/ui/render_world.go`.
- Player damage red vignette flash when own Blood drops (track delta frame-to-frame); damage floaters for combat near camera (CombatSystem emits via bridge: add optional `*InputBridge` events from a tiny `CombatFeedbackSystem` that diffs vitals of entities in viewport — simpler: effects layer samples CombatMarker pairs each frame and shows clash icon).
- Attack swing arc visual on player attack; corpse sprites already (T4); low-stamina (<10) blocks attack with "Exhausted" floater; pain >80 slows player speed ×0.5 (modify player_input WASD speed by vitals).
- Keep deterministic sim — all feel is render-side except pain-slow (sim-side, test it).

### Task 20: Docs, full verification, review
**Files:** Modify `docs/implemented_functionality.md` (new sections), `README.md` (controls table, new screens), `docs/roadmap.md` (mark shell phases). 
- Run: `go vet ./...`, full `go test ./...`, `$env:CGO_ENABLED=0; go build -o game.exe ./cmd/game`, manual launch smoke (menu→play→build→order→save).
- Dispatch review workflow (bugs/perf/standards-compliance reviewers + adversarial verify), fix confirmed findings, re-test, final commit.

## Execution order & checkpoints
M1 = T1–T4 (build+screenshot check), M2 = T5–T8, M3 = T9–T10, M4 = T11–T14, M5 = T15–T18, M6 = T19–T20. Commit per task; review workflow after M2, M4, M6.
