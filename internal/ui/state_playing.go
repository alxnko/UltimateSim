package ui

import (
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: StatePlaying is the gameplay orchestrator. It owns the
// PlayerContext, runs input → world mutation → tick on the single Ebiten
// goroutine, then draws the world, HUD and panels.

type StatePlaying struct {
	Status  *render.LoadingStatus
	PC      *PlayerContext
	lens    lensData
	minimap Minimap
	menuCtx menuContext
}

// NewStatePlaying builds the playing state around a loading status handle.
func NewStatePlaying(status *render.LoadingStatus) *StatePlaying {
	return &StatePlaying{
		Status: status,
		PC: &PlayerContext{
			Status: status,
			Cam:    Camera{X: 512, Y: 512, Zoom: 1.5},
		},
	}
}

func (s *StatePlaying) Update(sm *StateManager) error {
	s.Status.Mutex.Lock()
	defer s.Status.Mutex.Unlock()

	BeginUIFrame()

	if !s.Status.Done {
		return nil
	}
	pc := s.PC

	if !pc.Initialized {
		s.bindHandles()
		pc.Initialized = true
		if _, ok := pc.PossessedEntity(); ok {
			pc.Warmup = 0 // loaded save: body already claimed
		} else {
			pc.Warmup = warmupTicks
		}
	}

	// Fresh world: burst-run the sim so genesis + settlement settle before
	// the player picks a body. ~20 ticks/frame keeps the UI responsive.
	if pc.Warmup > 0 {
		for i := 0; i < 20 && pc.Warmup > 0; i++ {
			s.Status.TM.Tick()
			pc.Warmup--
		}
		if pc.Warmup == 0 {
			s.openCharacterSelect()
		}
		return nil
	}

	world := s.Status.TM.World
	tick := s.Status.TM.Ticks

	// Death handling takes priority over all other input.
	if pc.Events != nil && pc.Events.PlayerDied {
		s.openHeirFlow()
		pc.Events.Clear()
	}

	// Character select owns the keyboard while open.
	if pc.SelectOpen {
		s.handleSelectKeys()
	}

	s.handleHotkeys()
	s.handleMouse()

	// Movement lock while any modal/menu owns input.
	if pc.Bridge != nil {
		pc.Bridge.MovementLocked = pc.AnyModal()
	}

	// Advance the simulation (respects its own pause flag). Speed is the
	// grand-strategy ticks-per-frame multiplier (keys 1-4).
	if !pc.AnyModal() || s.Status.TM.IsPaused {
		speed := s.Status.TM.Speed
		if speed < 1 {
			speed = 1
		}
		for i := 0; i < speed; i++ {
			s.Status.TM.Tick()
		}
	}

	// Drain combat/feedback events into the effects layer.
	if pc.Bridge != nil {
		pc.FX.Consume(pc.Bridge.DrainEvents(), pc, tick)
	}

	// Camera follows the possessed entity unless free-panning.
	if player, ok := pc.PossessedEntity(); ok && !pc.CamFree {
		posID := ecs.ComponentID[components.Position](world)
		if world.Has(player, posID) {
			pos := (*components.Position)(world.Get(player, posID))
			// Hurt vignette: detect blood drop.
			vitalsID := ecs.ComponentID[components.VitalsComponent](world)
			if world.Has(player, vitalsID) {
				v := (*components.VitalsComponent)(world.Get(player, vitalsID))
				if v.Blood < pc.lastBlood {
					pc.HurtFlash = 10
				}
				pc.lastBlood = v.Blood
			}
			// Smooth follow.
			pc.Cam.X += (float64(pos.X) - pc.Cam.X) * 0.15
			pc.Cam.Y += (float64(pos.Y) - pc.Cam.Y) * 0.15
		}
	}
	if pc.HurtFlash > 0 {
		pc.HurtFlash--
	}

	return nil
}

// warmupTicks is how long a fresh world simulates before character select:
// genesis spawns on tick 1; a few seconds of sim binds cities, jobs, leaders.
const warmupTicks = 240

// bindHandles wires the UI to the engine's shared handles. Possession happens
// later, through the character-select screen (or a loaded save).
func (s *StatePlaying) bindHandles() {
	pc := s.PC
	pc.Sprites = render.NewSpriteCache()
	pc.Bridge = s.Status.Bridge
	pc.Events = s.Status.Events
}

// handleHotkeys processes keyboard shortcuts.
func (s *StatePlaying) handleHotkeys() {
	pc := s.PC

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if pc.AnyModal() || pc.Mode != ModeNormal {
			pc.CloseAll()
		} else {
			pc.PauseOpen = true
		}
		return
	}
	if pc.AnyModal() {
		return // modal owns the rest of the keyboard
	}

	// Zoom.
	if _, dy := ebiten.Wheel(); dy != 0 && !pc.overMinimap() {
		if dy > 0 {
			pc.Cam.ZoomBy(1.15)
		} else {
			pc.Cam.ZoomBy(1 / 1.15)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		s.toggleBuildMode()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		pc.AmbitionsOpen = !pc.AmbitionsOpen
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		pc.CharOpen = !pc.CharOpen
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		pc.EventLogOpen = !pc.EventLogOpen
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		pc.Lens = (pc.Lens + 1) % lensCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		pc.Lens = LensPolitical
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		pc.Lens = LensWealth
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		pc.Lens = LensCrime
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		pc.Lens = LensCulture
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.Status.TM.TogglePause()
	}
	// Grand Strategy Phase: sim speed 1/2/4/8 ticks-per-frame on keys 1-4.
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		s.Status.TM.Speed = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		s.Status.TM.Speed = 2
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		s.Status.TM.Speed = 4
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		s.Status.TM.Speed = 8
	}
	// Free camera: arrow keys pan; F (or moving your character) re-follows.
	panStep := 12.0 / pc.Cam.TileSize()
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		pc.Cam.X -= panStep
		pc.CamFree = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		pc.Cam.X += panStep
		pc.CamFree = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		pc.Cam.Y -= panStep
		pc.CamFree = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		pc.Cam.Y += panStep
		pc.CamFree = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) ||
		ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyA) ||
		ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyD) {
		pc.CamFree = false
	}
	// Hammer a nearby construction site.
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		s.tryHammer()
	}
	// Build structure cycle while in build mode.
	if pc.Mode == ModeBuild && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		pc.BuildType++
		if pc.BuildType > components.StructureTavern {
			pc.BuildType = components.StructureHouse
		}
	}
}

func (pc *PlayerContext) overMinimap() bool {
	mx, my := ebiten.CursorPosition()
	sw, _ := ebiten.WindowSize()
	if sw < 960 {
		sw = 960
	}
	return mx > sw-MinimapSize-16 && my < MinimapSize+16
}

// tryHammer advances the nearest construction site near the player.
func (s *StatePlaying) tryHammer() {
	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		return
	}
	posID := ecs.ComponentID[components.Position](world)
	if !world.Has(player, posID) {
		return
	}
	pos := (*components.Position)(world.Get(player, posID))
	if site, found := systems.NearestSite(world, pos.X, pos.Y, 1.5); found {
		if err := systems.HammerSite(world, player, site); err == nil {
			s.PC.PushNote("You labor at the construction site.", s.Status.TM.Ticks)
		}
	}
}

// handleMouse processes world clicks (select / attack / build / context menu).
func (s *StatePlaying) handleMouse() {
	pc := s.PC
	world := s.Status.TM.World
	sw, sh := ebiten.WindowSize()
	if sw < 960 {
		sw = 960
	}
	if sh < 540 {
		sh = 540
	}
	mx, my := ebiten.CursorPosition()

	// Context menu has first claim on clicks.
	if idx, closed := pc.Menu.Update(); closed {
		if idx >= 0 {
			s.runContextAction(idx)
		}
		return
	}
	if idx, closed := pc.SubMenu.Update(); closed {
		if idx >= 0 {
			s.runSubMenuAction(idx)
		}
		return
	}

	if pc.AnyModal() {
		pc.dragging = false
		return
	}

	// Middle-mouse drag pans the camera (grand-strategy map scrubbing).
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		if pc.dragging {
			ts := pc.Cam.TileSize()
			pc.Cam.X -= float64(mx-pc.dragX) / ts
			pc.Cam.Y -= float64(my-pc.dragY) / ts
			pc.CamFree = true
		}
		pc.dragging = true
		pc.dragX, pc.dragY = mx, my
	} else {
		pc.dragging = false
	}

	// Left clicks arrive one tick late via TakeWorldClick so drawn widgets
	// (buttons, modals) always get first claim on them.
	cmx, cmy, clicked := TakeWorldClick()

	// Minimap click: jump the camera to that world spot.
	if clicked && pc.overMinimap() {
		x0 := sw - MinimapSize - 8
		y0 := 8
		if cmx >= x0 && cmx < x0+MinimapSize && cmy >= y0 && cmy < y0+MinimapSize {
			scale := float64(s.Status.Grid.Width) / float64(MinimapSize)
			pc.Cam.X = float64(cmx-x0) * scale
			pc.Cam.Y = float64(cmy-y0) * scale
			pc.CamFree = true
		}
		return
	}

	if pc.UIOccludes(mx, my, sw, sh) {
		return
	}

	wx, wy := pc.Cam.ScreenToWorld(mx, my, sw, sh)
	cwx, cwy := pc.Cam.ScreenToWorld(cmx, cmy, sw, sh)

	// Build placement.
	if pc.Mode == ModeBuild {
		if clicked {
			if player, ok := pc.PossessedEntity(); ok {
				if _, err := systems.PlaceSite(world, s.Status.Grid, player, pc.BuildType, cwx, cwy); err != nil {
					pc.PushNote("Cannot build: "+err.Error(), s.Status.TM.Ticks)
				} else {
					pc.PushNote("Construction site founded.", s.Status.TM.Ticks)
					if ambID := ecs.ComponentID[components.AmbitionsComponent](world); world.Has(player, ambID) {
						systems.RecordPlayerBuild((*components.AmbitionsComponent)(world.Get(player, ambID)))
					}
				}
			}
		}
		return
	}

	// Order targeting (armed move/attack/build order awaiting a click).
	if pc.Mode == ModeOrder {
		if clicked {
			s.placeOrderTarget(cwx, cwy)
		}
		return
	}

	// Left click: attack NPC, or select.
	if clicked {
		ent, kind, found := pc.EntityAt(cwx, cwy, 1.2)
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		if found && kind == TargetNPC && !shift {
			// Attack.
			if pc.Bridge != nil {
				pc.Bridge.AttackX, pc.Bridge.AttackY, pc.Bridge.AttackValid = cwx, cwy, true
			}
		} else if found {
			pc.Select(ent, kind)
		} else {
			pc.ClearSelection()
		}
	}

	// Right click: context menu.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		s.openContextMenu(mx, my, wx, wy)
	}
}

func (s *StatePlaying) Draw(screen *ebiten.Image) {
	s.Status.Mutex.Lock()
	defer s.Status.Mutex.Unlock()

	if !s.Status.Done {
		screen.Fill(color.RGBA{20, 20, 30, 255})
		ebitenutil.DebugPrintAt(screen, "Loading: "+s.Status.Message, 10, 10)
		return
	}
	// Draw can run before the first Update on the state-switch frame; the
	// sprite cache and bridges bind in Update, so wait one frame.
	if !s.PC.Initialized {
		screen.Fill(color.RGBA{20, 20, 30, 255})
		ebitenutil.DebugPrintAt(screen, "Awakening...", 10, 10)
		return
	}
	screen.Fill(color.RGBA{18, 20, 28, 255})

	// Warmup: world visible behind a progress veil, chrome hidden.
	if s.PC.Warmup > 0 {
		s.DrawWorld(screen)
		sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
		DrawPanel(screen, sw/2-160, sh/2-30, 320, 60)
		DrawText(screen, "The world awakens...", sw/2-70, sh/2-20, AccentCol)
		frac := 1 - float32(s.PC.Warmup)/float32(warmupTicks)
		Bar(screen, sw/2-140, sh/2+4, 280, 12, frac, BarBlue, "")
		return
	}

	s.DrawWorld(screen)
	DrawHurtVignette(screen, s.PC.HurtFlash)
	s.PC.FX.Draw(screen, &s.PC.Cam)
	DrawBuildGhost(s, screen) // P5 L2: tile-snapped ghost + cost/reason
	DrawOverheads(s, screen)  // P5 L2/L3: site bars, health bars, name plate, player marker

	// Chrome.
	s.DrawHUD(screen)
	s.minimap.Draw(screen, s.Status.Grid, s.Status.TM.World, screen.Bounds().Dx(), &s.PC.Cam)
	s.PC.Notes.Draw(screen, s.Status.TM.Ticks, screen.Bounds().Dx())
	s.DrawInspector(screen)

	// Panels & modals.
	s.DrawBuildBar(screen)
	s.DrawDialog(screen)
	s.DrawTrade(screen)
	s.DrawLaws(screen)
	s.DrawAmbitions(screen)
	s.DrawCharPanel(screen)
	s.DrawEventLog(screen)
	s.DrawOrderBar(screen)
	s.DrawHeir(screen)
	s.DrawPauseMenu(screen)
	s.DrawCharacterSelect(screen)

	// Popup menus on top.
	s.PC.Menu.Draw(screen)
	s.PC.SubMenu.Draw(screen)

	// Queued tooltips render above everything.
	FlushTooltips(screen)
}

// getBiomeColor maps a biome ID to its terrain color.
func getBiomeColor(biomeID uint8) color.RGBA {
	switch biomeID {
	case engine.BiomeOcean:
		return color.RGBA{38, 70, 130, 255}
	case engine.BiomeBeach:
		return color.RGBA{210, 190, 140, 255}
	case engine.BiomeScorched:
		return color.RGBA{90, 60, 40, 255}
	case engine.BiomeBare:
		return color.RGBA{140, 135, 125, 255}
	case engine.BiomeTundra:
		return color.RGBA{190, 200, 180, 255}
	case engine.BiomeSnow:
		return color.RGBA{235, 240, 245, 255}
	case engine.BiomeTemperateDesert:
		return color.RGBA{198, 180, 120, 255}
	case engine.BiomeShrubland:
		return color.RGBA{130, 150, 90, 255}
	case engine.BiomeGrassland:
		return color.RGBA{96, 158, 72, 255}
	case engine.BiomeTemperateDeciduousForest:
		return color.RGBA{56, 120, 56, 255}
	case engine.BiomeTemperateRainForest:
		return color.RGBA{36, 96, 64, 255}
	case engine.BiomeSubtropicalDesert:
		return color.RGBA{205, 165, 100, 255}
	case engine.BiomeTropicalSeasonalForest:
		return color.RGBA{80, 130, 60, 255}
	case engine.BiomeTropicalRainForest:
		return color.RGBA{30, 90, 50, 255}
	case engine.BiomeMountain:
		return color.RGBA{110, 105, 100, 255}
	default:
		return color.RGBA{60, 60, 70, 255}
	}
}
