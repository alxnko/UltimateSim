package ui

import (
	"fmt"
	"image/color"
	"math"
	"sort"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (P5 L2/L3): world-space overlays drawn between the
// entity pass and the HUD chrome — the build-mode ghost footprint with cost
// and failure reason, overhead progress/health bars, the hover name plate,
// the possessed-player marker, and player-construction feedback toasts.

// OverheadZoomThreshold mirrors LensZoomThreshold: overheads only render in
// the action view (at or above the strategic-lens cutoff).
const OverheadZoomThreshold = LensZoomThreshold

// OverheadHealthBloodMax is the Blood value below which an NPC shows an
// overhead health bar (healthy NPCs stay clean).
const OverheadHealthBloodMax = 90

// OverheadsVisible reports whether overhead info draws at the given zoom.
func OverheadsVisible(zoom float64) bool {
	return zoom >= OverheadZoomThreshold
}

// NeedsHealthBar reports whether an NPC's blood level warrants an overhead bar.
func NeedsHealthBar(blood float32) bool {
	return blood < OverheadHealthBloodMax
}

// GhostTile snaps a world position to the tile containing it (floor, so
// negative coordinates snap left/up rather than toward zero).
func GhostTile(wx, wy float32) (int, int) {
	return int(math.Floor(float64(wx))), int(math.Floor(float64(wy)))
}

// SiteProgressFrac converts site progress to a clamped 0..1 bar fraction.
func SiteProgressFrac(progress, maxProgress uint32) float32 {
	if maxProgress == 0 {
		return 0
	}
	f := float32(progress) / float32(maxProgress)
	if f > 1 {
		f = 1
	}
	return f
}

// StructureInitial is the one-letter overhead tag for a structure type,
// derived from its display name ("House" -> "H").
func StructureInitial(t uint32) string {
	return StructureName(t)[:1]
}

// BuildCostLabel formats the ghost cost readout, e.g. "House 50W/50S".
func BuildCostLabel(buildType uint8) string {
	wood, stone := systems.BuildCosts(buildType)
	return fmt.Sprintf("%s %dW/%dS", StructureName(uint32(buildType)), wood, stone)
}

// DrawBuildGhost previews placement while in build mode: a tile-snapped
// footprint rect under the cursor (green = valid, red = invalid), a floating
// cost label, and — when invalid — the reason ("water", "too close to a
// building", "not enough materials", ...). Draw after the world pass and
// before the HUD chrome.
func DrawBuildGhost(s *StatePlaying, screen *ebiten.Image) {
	pc := s.PC
	if pc.Mode != ModeBuild || !OverheadsVisible(pc.Cam.Zoom) {
		return
	}
	world := s.Status.TM.World
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	mx, my := ebiten.CursorPosition()
	wx, wy := pc.Cam.ScreenToWorld(mx, my, sw, sh)

	// Zero entity when nothing is possessed: geometry-only validity.
	player, _ := pc.PossessedEntity()
	reason, ok := systems.CanPlaceReason(world, s.Status.Grid, player, pc.BuildType, wx, wy)

	// Tile-snapped semi-transparent footprint with a solid edge.
	tx, ty := GhostTile(wx, wy)
	gx, gy := pc.Cam.WorldToScreen(float32(tx), float32(ty), sw, sh)
	ts := pc.Cam.TileSize()
	fill := color.RGBA{90, 200, 90, 80}
	edge := color.RGBA{120, 230, 120, 200}
	if !ok {
		fill = color.RGBA{210, 70, 60, 80}
		edge = color.RGBA{240, 100, 90, 200}
	}
	ebitenutil.DrawRect(screen, gx, gy, ts, ts, fill)
	ebitenutil.DrawRect(screen, gx, gy, ts, 1, edge)
	ebitenutil.DrawRect(screen, gx, gy+ts-1, ts, 1, edge)
	ebitenutil.DrawRect(screen, gx, gy, 1, ts, edge)
	ebitenutil.DrawRect(screen, gx+ts-1, gy, 1, ts, edge)

	// Floating cost label above the footprint, reason above that when invalid.
	label := BuildCostLabel(pc.BuildType)
	lw := MeasureText(label)
	lx := int(gx+ts/2) - lw/2
	ly := int(gy) - 20
	ebitenutil.DrawRect(screen, float64(lx-4), float64(ly-2), float64(lw+8), 18, color.RGBA{15, 17, 22, 200})
	DrawText(screen, label, lx, ly, TextCol)
	if !ok {
		rw := MeasureAt(reason, SmallSize)
		rx := int(gx+ts/2) - rw/2
		ry := ly - 17
		ebitenutil.DrawRect(screen, float64(rx-4), float64(ry-1), float64(rw+8), 15, color.RGBA{15, 17, 22, 200})
		DrawSmall(screen, reason, rx, ry, color.RGBA{240, 120, 110, 255})
	}
}

// DrawOverheads renders overhead world-space info at action zoom: construction
// site progress bars with a structure initial, health bars over damaged NPCs,
// the possessed-player marker, and a single hover name plate. It also polls
// player-construction transitions for feedback toasts (at every zoom, so
// builder start/finish news is never missed while zoomed out). Draw after the
// effects layer and before the HUD chrome.
func DrawOverheads(s *StatePlaying, screen *ebiten.Image) {
	pc := s.PC
	world := s.Status.TM.World
	tick := s.Status.TM.Ticks

	playerSiteToasts.Poll(world, tick, pc.PushNote)

	if !OverheadsVisible(pc.Cam.Zoom) {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	cam := &pc.Cam
	zoom := cam.Zoom
	posID := ecs.ComponentID[components.Position](world)

	// --- Construction sites: mini progress bar + structure-type initial ---
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	q := world.Query(ecs.All(siteID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		site := (*components.ConstructionSiteComponent)(q.Get(siteID))
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		if !onScreen(sx, sy, sw, sh, 32) {
			continue
		}
		w := 22 * zoom
		by := sy - 12*zoom
		frac := SiteProgressFrac(site.Progress, site.MaxProgress)
		ebitenutil.DrawRect(screen, sx-w/2, by, w, 3, color.RGBA{30, 32, 38, 220})
		ebitenutil.DrawRect(screen, sx-w/2, by, w*float64(frac), 3, BarGreen)
		ini := StructureInitial(site.SiteType)
		DrawSmall(screen, ini, int(sx)-MeasureAt(ini, SmallSize)/2, int(by)-14, AccentCol)
	}

	// --- Damaged NPCs: small red health bar above the head ---
	npcID := ecs.ComponentID[components.NPC](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	q = world.Query(ecs.All(npcID, posID, vitalsID))
	for q.Next() {
		v := (*components.VitalsComponent)(q.Get(vitalsID))
		if !NeedsHealthBar(v.Blood) {
			continue
		}
		pos := (*components.Position)(q.Get(posID))
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		if !onScreen(sx, sy, sw, sh, 24) {
			continue
		}
		w := 16 * zoom
		by := sy - 11*zoom
		frac := v.Blood / 100
		if frac < 0 {
			frac = 0
		}
		ebitenutil.DrawRect(screen, sx-w/2, by, w, 2, color.RGBA{40, 20, 20, 220})
		ebitenutil.DrawRect(screen, sx-w/2, by, w*float64(frac), 2, BarRed)
	}

	// --- Possessed player: subtle accent underline so you never lose your
	// character in a crowd ---
	if player, ok := pc.PossessedEntity(); ok && world.Has(player, posID) {
		pos := (*components.Position)(world.Get(player, posID))
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		if onScreen(sx, sy, sw, sh, 24) {
			w := 16 * zoom
			ebitenutil.DrawRect(screen, sx-w/2, sy+9*zoom, w, 2, AccentCol)
		}
	}

	// --- Hover name plate: at most ONE per frame ---
	drawHoverPlate(s, screen, sw, sh)
}

// drawHoverPlate shows "Name — Job" above the NPC under the cursor. EntityAt
// resolves a single best entity, so at most one plate renders per frame.
func drawHoverPlate(s *StatePlaying, screen *ebiten.Image, sw, sh int) {
	pc := s.PC
	if pc.Mode != ModeNormal { // build ghost / order targeting own the cursor
		return
	}
	mx, my := ebiten.CursorPosition()
	if pc.UIOccludes(mx, my, sw, sh) {
		return
	}
	world := s.Status.TM.World
	wx, wy := pc.Cam.ScreenToWorld(mx, my, sw, sh)
	ent, kind, found := pc.EntityAt(wx, wy, 1.0)
	if !found || kind != TargetNPC {
		return
	}
	posID := ecs.ComponentID[components.Position](world)
	identID := ecs.ComponentID[components.Identity](world)
	if !world.Has(ent, posID) || !world.Has(ent, identID) {
		return
	}
	pos := (*components.Position)(world.Get(ent, posID))
	ident := (*components.Identity)(world.Get(ent, identID))
	label := ident.Name
	jobID := ecs.ComponentID[components.JobComponent](world)
	if world.Has(ent, jobID) {
		label += " — " + JobName((*components.JobComponent)(world.Get(ent, jobID)).JobID)
	}
	sx, sy := pc.Cam.WorldToScreen(pos.X, pos.Y, sw, sh)
	w := MeasureAt(label, SmallSize)
	px := int(sx) - w/2
	py := int(sy-14*pc.Cam.Zoom) - 16
	ebitenutil.DrawRect(screen, float64(px-4), float64(py-2), float64(w+8), 16, color.RGBA{15, 17, 22, 210})
	DrawSmall(screen, label, px, py, TextCol)
}

// --- Player-construction feedback toasts -----------------------------------

// siteTrack is the remembered state of one construction site entity.
type siteTrack struct {
	siteType    uint32
	playerOwned bool
	started     bool // builder-claim toast already handled
}

// siteToastTracker watches construction sites frame-to-frame and pushes a
// toast when builders start on — and when they finish — a site the player
// funded. Player sites are recognized by their fingerprint at first sighting:
// only PlaceSite spawns fully-funded sites (village sites gather materials
// over many ticks). The first poll after a world change is a silent baseline,
// so loading a save or finishing warmup never replays stale toasts.
type siteToastTracker struct {
	world     *ecs.World
	sites     map[ecs.Entity]siteTrack
	baselined bool
}

// playerSiteToasts lives at package level: state_playing.go is shared, so the
// tracker keeps its own state here and resets when the world pointer changes.
var playerSiteToasts siteToastTracker

// Poll scans construction sites, updates the tracker, and emits toasts via
// push. Safe to call every frame; transitions fire exactly once per site.
func (t *siteToastTracker) Poll(world *ecs.World, tick uint64, push func(text string, tick uint64)) {
	if t.world != world {
		t.world = world
		t.sites = make(map[ecs.Entity]siteTrack)
		t.baselined = false
	}

	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	structID := ecs.ComponentID[components.StructureComponent](world)

	live := make(map[ecs.Entity]bool, len(t.sites))
	q := world.Query(ecs.All(siteID))
	for q.Next() {
		e := q.Entity()
		site := (*components.ConstructionSiteComponent)(q.Get(siteID))
		live[e] = true
		tr, known := t.sites[e]
		if !known {
			tr = siteTrack{siteType: site.SiteType}
			// Fully funded at first sighting == player-placed (PlaceSite
			// supplies all materials upfront; village sites start empty and
			// cannot fill within one frame). Baseline sightings stay silent.
			if t.baselined &&
				site.WoodGathered >= site.WoodRequired &&
				site.StoneGathered >= site.StoneRequired {
				tr.playerOwned = true
			}
		}
		if site.BuilderID != 0 && !tr.started {
			tr.started = true
			if tr.playerOwned {
				push("Builders started on your "+StructureName(tr.siteType)+".", tick)
			}
		}
		t.sites[e] = tr
	}

	// Tracked sites gone from the site set either completed (the entity now
	// carries a StructureComponent) or were destroyed. Sort by entity ID so
	// multiple same-frame completions toast in a deterministic order.
	var gone []ecs.Entity
	for e := range t.sites {
		if !live[e] {
			gone = append(gone, e)
		}
	}
	sort.Slice(gone, func(i, j int) bool { return gone[i].ID() < gone[j].ID() })
	for _, e := range gone {
		tr := t.sites[e]
		if tr.playerOwned && world.Alive(e) && world.Has(e, structID) {
			push("Your "+StructureName(tr.siteType)+" is complete!", tick)
		}
		delete(t.sites, e)
	}
	t.baselined = true
}
