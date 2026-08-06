package ui

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: PlayerContext is the single mutable hub shared by StatePlaying,
// the HUD, panels and modals. All access happens on the Ebiten goroutine
// between simulation ticks — no locking.

// Tool modes for world clicks.
const (
	ModeNormal uint8 = 0
	ModeBuild  uint8 = 1
	ModeOrder  uint8 = 2 // an order is armed and awaits a target click
)

// Click-target archetypes resolved by EntityAt.
const (
	TargetNone uint8 = iota
	TargetNPC
	TargetVillage
	TargetStructure
	TargetSite
	TargetItem
	TargetCoin
	TargetCorpse
	TargetWorkbench
	TargetCaravan
	TargetShip
)

// PlayerContext carries every piece of player-facing UI state.
type PlayerContext struct {
	Status *render.LoadingStatus
	Cam    Camera

	Bridge  *systems.InputBridge
	Events  *engine.PlayerEvents
	Sprites *render.SpriteCache
	Notes   Notifier
	FX      Effects

	// Selection
	Selected      ecs.Entity
	SelectedKind  uint8
	SelectedValid bool
	InspectorTab  int

	// Tool state
	Mode      uint8
	BuildType uint8 // components.Structure* when Mode == ModeBuild

	// Armed order state (Mode == ModeOrder)
	OrderType   uint8      // components.Order*
	OrderJobID  uint8      // for OrderAssignJob
	OrderTarget ecs.Entity // subordinate receiving the order

	// Open windows / modals
	Menu          ContextMenu
	SubMenu       ContextMenu
	SubMenuKind   uint8 // 1 = order verbs, 2 = job picker
	DialogOpen    bool
	DialogTarget  ecs.Entity
	DialogReply   string
	TradeOpen     bool
	TradeVillage  ecs.Entity
	LawsOpen      bool
	LawsVillage   ecs.Entity
	AmbitionsOpen bool
	CharOpen      bool
	EventLogOpen  bool
	PauseOpen     bool
	HeirOpen      bool
	GameOverOpen  bool
	HeirChoices   []systems.HeirInfo

	// Character select (start-of-game or after full dynasty wipe)
	SelectOpen    bool
	SelectChoices []systems.StartCandidate
	SelIndex      int

	// Warmup ticks remaining before character select opens on a fresh world.
	Warmup int

	// Free camera: true while the player pans away from their character.
	CamFree  bool
	dragging bool
	dragX    int
	dragY    int

	// Strategic lens (active below zoom threshold)
	Lens uint8

	// Event log history (longer than transient notifications)
	Log []Note

	Initialized bool
	lastBlood   float32 // damage vignette detection
	HurtFlash   int     // frames remaining of red vignette
	deathCause  uint8   // surfaced cause for the heir/game-over screens
}

// AnyModal reports whether a window that owns the keyboard is open.
func (pc *PlayerContext) AnyModal() bool {
	return pc.DialogOpen || pc.TradeOpen || pc.LawsOpen || pc.AmbitionsOpen ||
		pc.CharOpen || pc.PauseOpen || pc.HeirOpen || pc.GameOverOpen || pc.EventLogOpen ||
		pc.SelectOpen
}

// CloseAll dismisses every open window and resets tool modes.
func (pc *PlayerContext) CloseAll() {
	pc.DialogOpen = false
	pc.TradeOpen = false
	pc.LawsOpen = false
	pc.AmbitionsOpen = false
	pc.CharOpen = false
	pc.EventLogOpen = false
	pc.PauseOpen = false
	pc.Menu.Visible = false
	pc.SubMenu.Visible = false
	pc.Mode = ModeNormal
}

// PushNote records a notification both transiently and into the event log.
func (pc *PlayerContext) PushNote(text string, tick uint64) {
	pc.Notes.Push(text, tick)
	pc.Log = append(pc.Log, Note{Text: text, Tick: tick})
	if len(pc.Log) > 200 {
		pc.Log = pc.Log[len(pc.Log)-200:]
	}
}

// PossessedEntity finds the player-controlled entity.
func (pc *PlayerContext) PossessedEntity() (ecs.Entity, bool) {
	world := pc.Status.TM.World
	possessedID := ecs.ComponentID[components.Possessed](world)
	q := world.Query(ecs.All(possessedID))
	if q.Next() {
		e := q.Entity()
		q.Close()
		return e, true
	}
	return ecs.Entity{}, false
}

// UIOccludes reports whether a screen point lands on persistent chrome
// (HUD bar, minimap, open inspector) so world clicks are suppressed there.
func (pc *PlayerContext) UIOccludes(x, y, sw, sh int) bool {
	if y >= sh-HUDHeight {
		return true
	}
	if pc.SelectedValid && x >= sw-InspectorWidth {
		return true
	}
	if x >= sw-MinimapSize-16 && y <= MinimapSize+16 {
		return true
	}
	if pc.Menu.Visible || pc.SubMenu.Visible || pc.AnyModal() {
		return true
	}
	return false
}

// classify resolves the archetype of an entity for context menus / inspector.
func classify(world *ecs.World, e ecs.Entity) uint8 {
	has := func(id ecs.ID) bool { return world.Has(e, id) }
	if has(ecs.ComponentID[components.NPC](world)) {
		return TargetNPC
	}
	if has(ecs.ComponentID[components.Village](world)) {
		return TargetVillage
	}
	if has(ecs.ComponentID[components.ConstructionSiteComponent](world)) {
		return TargetSite
	}
	if has(ecs.ComponentID[components.StructureComponent](world)) {
		return TargetStructure
	}
	if has(ecs.ComponentID[components.CoinEntity](world)) {
		return TargetCoin
	}
	if has(ecs.ComponentID[components.ItemEntity](world)) {
		return TargetItem
	}
	if has(ecs.ComponentID[components.CorpseComponent](world)) {
		return TargetCorpse
	}
	if has(ecs.ComponentID[components.WorkbenchComponent](world)) {
		return TargetWorkbench
	}
	if has(ecs.ComponentID[components.Caravan](world)) {
		return TargetCaravan
	}
	if has(ecs.ComponentID[components.ShipComponent](world)) {
		return TargetShip
	}
	return TargetNone
}

// EntityAt picks the best entity near a world point: NPCs first (smaller
// radius), then anything else with a Position within the given radius.
func (pc *PlayerContext) EntityAt(wx, wy float32, radius float32) (ecs.Entity, uint8, bool) {
	world := pc.Status.TM.World
	posID := ecs.ComponentID[components.Position](world)

	var bestEnt ecs.Entity
	var bestKind uint8
	bestDist := radius * radius
	found := false

	q := world.Query(ecs.All(posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		dx := pos.X - wx
		dy := pos.Y - wy
		d := dx*dx + dy*dy
		if d > bestDist {
			continue
		}
		kind := classify(world, q.Entity())
		if kind == TargetNone {
			continue
		}
		// Prefer NPCs strongly: treat their distance as halved.
		if kind == TargetNPC {
			d *= 0.25
		}
		if !found || d < bestDist {
			bestDist = d
			bestEnt = q.Entity()
			bestKind = kind
			found = true
		}
	}
	return bestEnt, bestKind, found
}

// Select sets the inspector target.
func (pc *PlayerContext) Select(e ecs.Entity, kind uint8) {
	pc.Selected = e
	pc.SelectedKind = kind
	pc.SelectedValid = true
	pc.InspectorTab = 0
}

// ClearSelection drops the inspector target.
func (pc *PlayerContext) ClearSelection() {
	pc.SelectedValid = false
}
