# Boundless Sovereigns: Playable Game Shell — Design Spec

Date: 2026-06-11
Status: Approved (user delegated all recommended options)

## 1. Goal

Convert the existing deep simulation backend (62 registered systems, 199 system files) into a
fully playable game. The simulation is rich; the player layer is nearly empty (WASD movement
only, stub clicks, no UI, no building rendering). This spec defines the complete presentation
and interaction layer.

Locked decisions:
- **UI style**: Streets of Rogue / RimWorld hybrid — bottom HUD bar, right-side inspector,
  context menus, notification feed, strategic lenses at low zoom.
- **One character**: player controls exactly one possessed NPC at a time.
- **Power hierarchy**: orders + laws. As ruler of a city/country, issue orders to subordinates
  (move/work/attack/build) and set laws. Obedience gated by Legitimacy/Loyalty; refusal and
  rebellion remain possible via existing systems.
- **Builds** = building construction (RimWorld-style placement feeding the existing
  ConstructionSite → NPC Builder pipeline).
- **Visuals**: procedural pixel sprites generated in code (16x16 layered characters varying by
  genome/job, distinct building shapes). No asset files.
- **Structure**: sandbox + ambitions (generated optional goals with prestige rewards), plus the
  mandated death→heir legacy loop.
- Renderer stays **Ebiten v2**. Multiplayer and audio are out of scope.

## 2. Architecture

### 2.1 Shared player context (`internal/ui/player_context.go`)
A `PlayerContext` struct owned by `StatePlaying`, passed to UI widgets and player systems:
selected entity, active tool mode (Normal / Build / Order), pending build type, camera, lens,
notification queue, heir-death event flag. UI mutates the world only between ticks
(StatePlaying.Update runs UI input → world mutations → TickManager.Tick sequentially on one
goroutine), so no command queue is needed; direct mutation before Tick is safe and deterministic.

### 2.2 UI toolkit (`internal/ui/widgets.go`)
Immediate-mode helpers drawn with ebiten vector primitives + `text/v2`-free fallback
(`golang.org/x/image/font/basicfont` via `ebiten/v2/text`): Panel, Label, Button, ProgressBar,
Tooltip, ContextMenu, ScrollList, TabBar, IconButton. Theme constants (RimWorld-ish dark panels,
1px light borders). All stateless except focus/hover computed from cursor each frame.

### 2.3 Sprite system (`internal/render/sprites.go`)
`SpriteCache` lazily generates and caches `*ebiten.Image` keyed by sprite descriptor.
- **Characters**: 16x16, layers = body (skin tone from GenomeComponent.Beauty bits), head, hair
  (color from genome Dominant bits), clothing (color per JobID), equipment overlay (weapon if
  EquipmentComponent.Equipped), possession halo for the player. 2-frame walk bob keyed on
  velocity + tick parity. Deterministic per Identity.ID + genome.
- **Buildings**: distinct silhouettes — House (roofed square), Workshop/Workbench (anvil mark),
  Storehouse (wide barn), Shrine (obelisk), Village center (banner hall), Capital (keep + flag),
  Construction site (scaffold + progress), Ruin (broken walls), Ship, Caravan (cart), Item
  (sparkle), Coin pile, Corpse, Ledger (book).
- Macro/micro parity: wealthy villages (high storage) render with stone-texture variant.

### 2.4 World rendering (`internal/ui/state_playing.go`, split into `render_world.go`)
Existing tile loop kept; add render passes: structures/villages/sites/workbenches (from ECS
positions), items/coins/corpses, characters (sprites), selection ring, build ghost, combat
flashes. Mouse-wheel zoom 0.25–4.0. Below zoom 0.5 switch to **strategic mode**: tiles tinted by
active lens (Political = CityID/CountryID color hash; Wealth = village storage heat; Crime =
CrimeMarker density; Culture = LanguageID hash), settlement name labels, character dots.

### 2.5 HUD + panels
- **Bottom bar**: vitals (Blood, Stamina, Pain, Consciousness), needs (Food, Rest, Wealth),
  sanity stress, job + city + rank, date/season, speed controls (pause/play/fast), tool buttons
  (Build, Orders, Ambitions, Character).
- **Right inspector** (opens on click-select): tabs Bio (identity/genome/age/traits), Needs/Vitals,
  Social (memory events, hook summary, secrets count), Politics (affiliation, crime, legitimacy),
  plus building/village tabs (storage, population, market prices, jurisdiction laws) when a
  structure/village is selected. Tile info when empty ground selected.
- **Top-right**: minimap (downsampled biome image, player dot, settlement dots) + notification
  feed (combat, death, ambition complete, order refused, rebellion...).
- **Context menu** (right-click): target-sensitive — NPC: Talk / Trade / Attack / Order (if
  subordinate) / Inspect; Village/structure: Inspect / Laws (if ruler); Item: Pick up; ground:
  Move here / Build here.

### 2.6 Player verbs (wired into existing systems)
- **Attack** (left-click): nearest NPC within 1.5 tiles of cursor world pos → add
  `CombatMarker{TargetID}` to player; existing CombatSystem resolves. Hit feedback flash +
  floating damage text.
- **Talk** (dialog window): Chat (+1 hook both ways, MemoryEvent), Share rumor (transfer random
  secret, language-penalty per existing rules), Gift (wealth transfer → +3 hook), Threaten
  (fear: −2 hook, possible CrimeMarker via jurisdiction).
- **Trade** (window): buy/sell Wood/Stone/Iron/Food vs nearest village storage at local
  MarketComponent prices; player wealth in Needs.Wealth, goods in player StorageComponent
  (added to possessed entity on first trade/pickup).
- **Pick up**: ItemEntity/coin within range → StorageComponent/wealth; legendary items go to
  EquipmentComponent.
- **Build mode** (B): pick structure (House / Workshop / Storehouse / Shrine), ghost preview,
  validate tile (land, no overlap), click → spawn entity with Position +
  `ConstructionSiteComponent` (costs scaled per type, deducted from player storage/wealth);
  existing Builder-job NPCs complete it. Player can work a site via interact (E) to add progress.
  New structure type constants extend `StructureHouse`. Completed structures get
  StructureComponent + type effects: Storehouse adds village storage cap presence, Shrine
  spreads player's top belief (preacher-like radius), Workshop spawns WorkbenchComponent,
  House improves nearby Rest recovery.
- **Possession is fixed** to one character; switching only happens through the heir flow.

### 2.7 Power hierarchy (`internal/systems/player_order.go`, `internal/ui/panel_orders.go`)
- **Rank detection**: player is City Ruler if they hold `AdministrationMarker` (or highest
  positive-hook score for their CityID per LeadershipEmergence); Country Ruler if their city is
  the Capital of a CountryID. Rank shown in HUD.
- **Claim Leadership** action: available when player's incoming positive hooks in their city
  exceed the current ruler's (or ruler dead) — transfers AdministrationMarker, seeds
  LegitimacyComponent.
- **Orders**: new DOD component `PlayerOrderComponent{OrderType uint8, TargetID uint64,
  X, Y float32, IssuerID uint64}` attached to subordinate NPCs (same CityID, or same CountryID
  for country rulers). Types: Move, AssignJob, Attack, BuildAt, FollowMe. New
  `PlayerOrderSystem` (PhaseAI) executes orders: obedience roll = Legitimacy score vs NPC
  Loyalty/hook balance; refusal removes order, posts notification, −legitimacy. Obeyed Move
  routes via PathRequestQueue; AssignJob mutates JobComponent; Attack adds CombatMarker;
  BuildAt sets Builder job + site target.
- **Laws panel** (ruler only): edit village `JurisdictionComponent.IllegalActionIDs` bitmask
  (assault/theft/murder/esoteric/gossip), ContrabandComponent bitmask, view corruption.
  Country ruler additionally: currency Debasement slider (existing inflation pipeline), declare
  war on another country (WarTrackerComponent), make peace.
- All consequences flow through existing systems (Justice, VassalRebellion, MilitaryRevolt,
  Legitimacy, Inflation) — laws/orders are inputs, not new simulations.

### 2.8 Ambitions (`internal/systems/ambition.go`)
DOD component `AmbitionComponent{Type uint8, TargetID uint32, Goal uint32, Progress uint32,
Flags uint8}` (slice held on player entity in an `AmbitionsComponent{Ambitions []Ambition}`).
`AmbitionSystem` (PhaseResolution, throttled): generates up to 3 offered ambitions from world
state (Become ruler of <city>; Amass <N> wealth; Construct <N> buildings; Defeat <feud enemy>;
Raise an heir; Make <city> belief-majority <belief>), tracks progress, completion grants
Legacy.Prestige + notification. Ambitions panel lists offers (accept/dismiss) and active ones
with progress bars.

### 2.9 Death → heir loop
DeathSystem already handles inheritance. Add: when reaping an entity with `Possessed`, write a
`PlayerDeathEvent` (cause, tick) into a small shared struct (engine-level, mutex-free since
single goroutine). StatePlaying sees it and pushes **HeirSelection** overlay: lists family
members (same FamilyID, alive) with stats/age/job; picking one moves `Possessed` to heir
(inheriting existing Legacy debt/prestige flows). No family → **Legacy summary / Game Over**
screen (dynasty stats) → return to main menu or pick any newborn NPC ("reincarnate").

### 2.10 Game states & meta
- **Main menu**: New Game (seed entry, world size S/M/L), Continue (load save), Quit.
- **Pause menu** (Esc): Resume, Save, Load, Controls reference, Quit to menu. Wires existing
  SaveWorld/LoadWorld. Save schema extended with: StorageComponent-on-NPC, EquipmentComponent,
  SanityComponent, TreasuryComponent, MarketComponent, LegitimacyComponent, LoyaltyComponent,
  JurisdictionComponent, StructureComponent, ConstructionSiteComponent, JobComponent fields,
  PlayerOrderComponent, AmbitionsComponent, AdministrationMarker, CapitalComponent,
  CountryComponent, Village/NPC tags already covered.
- **Speed controls**: pause / 1x / fast-forward (existing TickManager flags).

## 3. Components & systems added (all DOD-compliant, E2E tested)
| New | Kind | Notes |
|---|---|---|
| PlayerOrderComponent | component | 24B padded |
| AmbitionsComponent | component | slice header, Ambition entries 16B |
| PlayerOrderSystem | system (PhaseAI) | obedience gating |
| AmbitionSystem | system (PhaseResolution, throttled) | generation + progress |
| PlayerDeathEvent | engine struct | possessed-death surfacing |
| Structure type consts | constants | Workshop=2, Storehouse=3, Shrine=4 |
| StructureEffectSystem | system (throttled) | shrine belief spread, house rest aura |

Modified: PlayerInputSystem (full verb wiring, tool modes), DeathSystem (possessed hook),
state_playing (rendering split + UI), save_load (schema), main.go (registration, window 1280x720).

## 4. Testing
- E2E deterministic tests for PlayerOrderSystem, AmbitionSystem, StructureEffectSystem,
  build-placement validation, heir selection logic, obedience math, screen↔world transforms,
  sprite determinism (same ID → same pixels), save/load round-trip with new schema.
- Existing test suite must stay green. Manual playtest via built binary before completion.

## 5. Error handling
- All ECS mutations from UI happen between ticks; deferred-slice pattern inside systems.
- Build placement validates biome (no ocean), bounds, overlap.
- Orders to dead/missing targets self-remove. Heir flow tolerates empty family (game over path).
- Save failures surface as notifications, never crash.

## 6. Out of scope
Multiplayer wiring, audio, asset pipelines, 3D, Z-levels, modding API.
