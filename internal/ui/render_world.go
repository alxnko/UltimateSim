package ui

import (
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: full world rendering — biome tiles, buildings, items, sprite
// characters, selection ring, build ghost; strategic lens mode at low zoom.

// Layout constants shared across HUD/panels.
const (
	HUDHeight      = 64
	InspectorWidth = 300
	MinimapSize    = 128
)

// structureSprite maps a StructureComponent type to a sprite kind.
func structureSprite(t uint32) string {
	switch uint8(t) {
	case components.StructureHouse:
		return "house"
	case components.StructureWorkshop:
		return "workshop"
	case components.StructureStorehouse:
		return "storehouse"
	case components.StructureShrine:
		return "shrine"
	case components.StructureFarm:
		return "farm"
	case components.StructureTavern:
		return "tavern"
	}
	return "house"
}

// DrawWorld renders the complete world view for the current camera.
func (s *StatePlaying) DrawWorld(screen *ebiten.Image) {
	if s.PC.Cam.Zoom < LensZoomThreshold {
		s.drawStrategic(screen)
		return
	}
	s.drawTiles(screen)
	s.drawEntities(screen)
	// Build ghost + overhead bars moved to render_overlays.go (P5 L2/L3).
}

// terrainCache is the whole biome map rasterized once at 1px/tile. Per-frame
// terrain becomes ONE scaled DrawImage instead of tens of thousands of
// DrawRect calls (the per-tile loop hit 4-6 FPS at large window sizes).
var (
	terrainCache     *ebiten.Image
	terrainCacheGrid interface{} // grid identity the cache was built from
)

func (s *StatePlaying) ensureTerrainCache() *ebiten.Image {
	grid := s.Status.Grid
	if terrainCache != nil && terrainCacheGrid == interface{}(grid) {
		return terrainCache
	}
	img := ebiten.NewImage(grid.Width, grid.Height)
	pix := make([]byte, grid.Width*grid.Height*4)
	for i := 0; i < grid.Width*grid.Height; i++ {
		clr := getBiomeColor(grid.Tiles[i].BiomeID)
		pix[i*4], pix[i*4+1], pix[i*4+2], pix[i*4+3] = clr.R, clr.G, clr.B, 255
	}
	img.WritePixels(pix)
	terrainCache = img
	terrainCacheGrid = grid
	return terrainCache
}

// drawTiles renders the visible terrain as one scaled blit from the cache.
func (s *StatePlaying) drawTiles(screen *ebiten.Image) {
	cam := &s.PC.Cam
	ts := cam.TileSize()
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(ts, ts)
	op.GeoM.Translate(float64(sw)/2-cam.X*ts, float64(sh)/2-cam.Y*ts)
	screen.DrawImage(s.ensureTerrainCache(), op)
}

// onScreen culls world positions outside the viewport.
func onScreen(sx, sy float64, sw, sh int, margin float64) bool {
	return sx > -margin && sx < float64(sw)+margin && sy > -margin && sy < float64(sh)+margin
}

// drawSpriteAt draws a sprite centered on a world position, scaled by zoom.
func (s *StatePlaying) drawSpriteAt(screen *ebiten.Image, img *ebiten.Image, wx, wy float32, sw, sh int) {
	if img == nil {
		return
	}
	cam := &s.PC.Cam
	sx, sy := cam.WorldToScreen(wx, wy, sw, sh)
	if !onScreen(sx, sy, sw, sh, 48) {
		return
	}
	scale := cam.Zoom
	w := float64(img.Bounds().Dx()) * scale
	h := float64(img.Bounds().Dy()) * scale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(sx-w/2, sy-h/2)
	screen.DrawImage(img, op)
}

// drawEntities renders every physical entity class with its sprite.
func (s *StatePlaying) drawEntities(screen *ebiten.Image) {
	world := s.Status.TM.World
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	sp := s.PC.Sprites
	cam := &s.PC.Cam
	tick := s.Status.TM.Ticks

	posID := ecs.ComponentID[components.Position](world)

	// --- Pass 1: ground furniture (structures, sites, villages, items...) ---
	structID := ecs.ComponentID[components.StructureComponent](world)
	q := world.Query(ecs.All(structID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		st := (*components.StructureComponent)(q.Get(structID))
		s.drawSpriteAt(screen, sp.Static(structureSprite(st.StructureType)), pos.X, pos.Y, sw, sh)
	}

	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	q = world.Query(ecs.All(siteID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		s.drawSpriteAt(screen, sp.Static("site"), pos.X, pos.Y, sw, sh)
	}

	wbID := ecs.ComponentID[components.WorkbenchComponent](world)
	q = world.Query(ecs.All(wbID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		s.drawSpriteAt(screen, sp.Static("workbench"), pos.X, pos.Y, sw, sh)
	}

	villageID := ecs.ComponentID[components.Village](world)
	storID := ecs.ComponentID[components.StorageComponent](world)
	capID := ecs.ComponentID[components.CapitalComponent](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)
	q = world.Query(ecs.All(villageID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		kind := "village"
		if q.Has(ruinID) {
			kind = "ruin"
		} else if q.Has(capID) {
			kind = "capital"
		} else if q.Has(storID) {
			st := (*components.StorageComponent)(q.Get(storID))
			if st.Wood+st.Stone+st.Iron+st.Food > 2000 {
				kind = "village_rich"
			}
		}
		s.drawSpriteAt(screen, sp.Static(kind), pos.X, pos.Y, sw, sh)
	}

	// Standalone ruins (former settlements stripped of Village tag).
	ruinFilter := ecs.All(ruinID, posID).Without(villageID)
	q = world.Query(&ruinFilter)
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		s.drawSpriteAt(screen, sp.Static("ruin"), pos.X, pos.Y, sw, sh)
	}

	simple := []struct {
		id   ecs.ID
		kind string
	}{
		{ecs.ComponentID[components.Caravan](world), "caravan"},
		{ecs.ComponentID[components.ShipComponent](world), "ship"},
		{ecs.ComponentID[components.CoinEntity](world), "coin"},
		{ecs.ComponentID[components.CorpseComponent](world), "corpse"},
		{ecs.ComponentID[components.Ledger](world), "ledger"},
	}
	for _, pass := range simple {
		q = world.Query(ecs.All(pass.id, posID))
		for q.Next() {
			pos := (*components.Position)(q.Get(posID))
			s.drawSpriteAt(screen, sp.Static(pass.kind), pos.X, pos.Y, sw, sh)
		}
	}

	itemID := ecs.ComponentID[components.ItemEntity](world)
	coinID := ecs.ComponentID[components.CoinEntity](world)
	itemFilter := ecs.All(itemID, posID).Without(coinID)
	q = world.Query(&itemFilter)
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		s.drawSpriteAt(screen, sp.Static("item"), pos.X, pos.Y, sw, sh)
	}

	// --- Pass 2: characters ---
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	velID := ecs.ComponentID[components.Velocity](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	possessedID := ecs.ComponentID[components.Possessed](world)

	q = world.Query(ecs.All(npcID, posID, identID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		ident := (*components.Identity)(q.Get(identID))
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		if !onScreen(sx, sy, sw, sh, 24) {
			continue
		}

		var genome *components.GenomeComponent
		if q.Has(genomeID) {
			genome = (*components.GenomeComponent)(q.Get(genomeID))
		}
		var job uint8
		if q.Has(jobID) {
			job = (*components.JobComponent)(q.Get(jobID)).JobID
		}
		frame := 0
		if q.Has(velID) {
			vel := (*components.Velocity)(q.Get(velID))
			if vel.X != 0 || vel.Y != 0 {
				frame = int((tick/12 + ident.ID) % 2)
			}
		}
		isPlayer := q.Has(possessedID)
		if isPlayer {
			// Possession ring under the player sprite.
			ringSize := 20.0 * cam.Zoom
			ebitenutil.DrawRect(screen, sx-ringSize/2, sy-ringSize/2, ringSize, ringSize, color.RGBA{230, 180, 60, 70})
		}
		s.drawSpriteAt(screen, sp.Char(ident.ID, genome, job, frame), pos.X, pos.Y, sw, sh)
	}

	if s.PC.SelectedValid && world.Alive(s.PC.Selected) && world.Has(s.PC.Selected, posID) {
		pos := (*components.Position)(world.Get(s.PC.Selected, posID))
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		size := 22.0 * cam.Zoom
		clr := color.RGBA{120, 220, 120, 200}
		ebitenutil.DrawRect(screen, sx-size/2, sy-size/2, size, 1, clr)
		ebitenutil.DrawRect(screen, sx-size/2, sy+size/2, size, 1, clr)
		ebitenutil.DrawRect(screen, sx-size/2, sy-size/2, 1, size, clr)
		ebitenutil.DrawRect(screen, sx+size/2, sy-size/2, 1, size, clr)
	}
}

// drawStrategic renders the macro lens view.
func (s *StatePlaying) drawStrategic(screen *ebiten.Image) {
	cam := &s.PC.Cam
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	world := s.Status.TM.World

	// Terrain backdrop from the cache (one blit), desaturated by a
	// translucent gray wash so lens overlays pop.
	ts := cam.TileSize()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(ts, ts)
	op.GeoM.Translate(float64(sw)/2-cam.X*ts, float64(sh)/2-cam.Y*ts)
	screen.DrawImage(s.ensureTerrainCache(), op)
	ebitenutil.DrawRect(screen, 0, 0, float64(sw), float64(sh), color.RGBA{128, 128, 128, 96})

	// City overlays.
	s.lens.rebuild(world, s.Status.TM.Ticks)
	maxWealth, maxCrime := s.lens.maxima()

	for i := range s.lens.cities {
		c := &s.lens.cities[i]
		sx, sy := cam.WorldToScreen(c.X, c.Y, sw, sh)
		if !onScreen(sx, sy, sw, sh, 64) {
			continue
		}
		clr := c.Color(s.PC.Lens, maxWealth, maxCrime)

		// Influence halo sized by jurisdiction radius.
		radius := float64(0)
		if c.RadiusSq > 0 {
			radius = float64(sqrt32(c.RadiusSq)) * cam.TileSize()
		}
		if radius > 2 {
			halo := color.RGBA{clr.R, clr.G, clr.B, 50}
			ebitenutil.DrawRect(screen, sx-radius, sy-radius, radius*2, radius*2, halo)
		}

		size := 8.0
		if c.IsCapital {
			size = 12.0
		}
		ebitenutil.DrawRect(screen, sx-size/2, sy-size/2, size, size, clr)
		label := c.Name
		if c.IsCapital {
			label = "★ " + label
		}
		DrawText(screen, label, int(sx)-MeasureText(label)/2, int(sy)-22, TextCol)
	}

	// Characters as dots; player highlighted.
	posID := ecs.ComponentID[components.Position](world)
	npcID := ecs.ComponentID[components.NPC](world)
	possessedID := ecs.ComponentID[components.Possessed](world)
	q := world.Query(ecs.All(npcID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		if !onScreen(sx, sy, sw, sh, 4) {
			continue
		}
		if q.Has(possessedID) {
			ebitenutil.DrawRect(screen, sx-3, sy-3, 6, 6, AccentCol)
		} else {
			ebitenutil.DrawRect(screen, sx-1, sy-1, 2, 2, color.RGBA{235, 235, 235, 180})
		}
	}

	// Lens selector hint.
	hint := "Lens [F1-F4]: " + LensName(s.PC.Lens)
	DrawText(screen, hint, 10, 10, AccentCol)
}

func sqrt32(v float32) float32 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 16; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}
