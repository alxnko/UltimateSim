package ui

import (
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
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
	s.drawBuildGhost(screen)
}

// drawTiles renders visible biome tiles.
func (s *StatePlaying) drawTiles(screen *ebiten.Image) {
	cam := &s.PC.Cam
	tileSize := cam.TileSize()
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	cols := int(float64(sw)/tileSize) + 2
	rows := int(float64(sh)/tileSize) + 2
	startX := int(cam.X) - cols/2
	startY := int(cam.Y) - rows/2

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tx, ty := startX+c, startY+r
			if tx < 0 || tx >= s.Status.Grid.Width || ty < 0 || ty >= s.Status.Grid.Height {
				continue
			}
			tile := s.Status.Grid.GetTile(tx, ty)
			clr := getBiomeColor(tile.BiomeID)
			x, y := cam.WorldToScreen(float32(tx), float32(ty), sw, sh)
			ebitenutil.DrawRect(screen, x, y, tileSize+1, tileSize+1, clr)
		}
	}
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
	type siteBar struct {
		x, y float64
		frac float32
	}
	var siteBars []siteBar
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		site := (*components.ConstructionSiteComponent)(q.Get(siteID))
		s.drawSpriteAt(screen, sp.Static("site"), pos.X, pos.Y, sw, sh)
		sx, sy := cam.WorldToScreen(pos.X, pos.Y, sw, sh)
		if onScreen(sx, sy, sw, sh, 32) && site.MaxProgress > 0 {
			siteBars = append(siteBars, siteBar{sx, sy, float32(site.Progress) / float32(site.MaxProgress)})
		}
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

	// --- Pass 3: overlays ---
	for _, b := range siteBars {
		w := 24.0 * cam.Zoom
		ebitenutil.DrawRect(screen, b.x-w/2, b.y+10*cam.Zoom, w, 3, color.RGBA{40, 40, 40, 255})
		ebitenutil.DrawRect(screen, b.x-w/2, b.y+10*cam.Zoom, w*float64(b.frac), 3, color.RGBA{110, 200, 110, 255})
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

// drawBuildGhost previews placement validity under the cursor in build mode.
func (s *StatePlaying) drawBuildGhost(screen *ebiten.Image) {
	if s.PC.Mode != ModeBuild {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	mx, my := ebiten.CursorPosition()
	wx, wy := s.PC.Cam.ScreenToWorld(mx, my, sw, sh)

	kind := "ghost_ok"
	if err := systems.CanPlace(s.Status.TM.World, s.Status.Grid, wx, wy); err != nil {
		kind = "ghost_bad"
	}
	s.drawSpriteAt(screen, s.PC.Sprites.Static(structureSprite(uint32(s.PC.BuildType))), wx, wy, sw, sh)
	s.drawSpriteAt(screen, s.PC.Sprites.Static(kind), wx, wy, sw, sh)
}

// drawStrategic renders the macro lens view.
func (s *StatePlaying) drawStrategic(screen *ebiten.Image) {
	cam := &s.PC.Cam
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	world := s.Status.TM.World

	// Desaturated terrain backdrop, blocky sampling for speed.
	tileSize := cam.TileSize()
	step := 1
	if tileSize < 4 {
		step = int(4 / tileSize)
		if step < 1 {
			step = 1
		}
	}
	cols := int(float64(sw)/(tileSize*float64(step))) + 2
	rows := int(float64(sh)/(tileSize*float64(step))) + 2
	startX := int(cam.X) - cols*step/2
	startY := int(cam.Y) - rows*step/2

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tx, ty := startX+c*step, startY+r*step
			if tx < 0 || tx >= s.Status.Grid.Width || ty < 0 || ty >= s.Status.Grid.Height {
				continue
			}
			tile := s.Status.Grid.GetTile(tx, ty)
			clr := getBiomeColor(tile.BiomeID)
			gray := (uint16(clr.R) + uint16(clr.G) + uint16(clr.B)) / 3
			mut := color.RGBA{uint8((uint16(clr.R) + gray) / 2), uint8((uint16(clr.G) + gray) / 2), uint8((uint16(clr.B) + gray) / 2), 255}
			x, y := cam.WorldToScreen(float32(tx), float32(ty), sw, sh)
			sz := tileSize*float64(step) + 1
			ebitenutil.DrawRect(screen, x, y, sz, sz, mut)
		}
	}

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
