package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

const TileSize = 16.0

// Phase 08.1: Window Management & Camera
// Establish window bounds and input capture mapping to camera vectors (Pan, Zoom).

// App implements ebiten.Game interface
type App struct {
	Camera      *Camera
	TickManager *engine.TickManager
	MapGrid     *engine.MapGrid
}

// NewApp creates a new application instance.
func NewApp(tm *engine.TickManager, mapGrid *engine.MapGrid) *App {
	return &App{
		Camera:      NewCamera(),
		TickManager: tm,
		MapGrid:     mapGrid,
	}
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (a *App) Update() error {
	// Handle Panning via WASD or Arrow Keys
	panSpeed := 10.0 / a.Camera.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		a.Camera.Y -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		a.Camera.Y += panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		a.Camera.X -= panSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		a.Camera.X += panSpeed
	}

	// Handle Zooming via Mouse Wheel
	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		a.Camera.Zoom *= 1.1 // Zoom in
	} else if wheelY < 0 {
		a.Camera.Zoom /= 1.1 // Zoom out
	}

	// Prevent zooming too far out or in
	if a.Camera.Zoom < 0.1 {
		a.Camera.Zoom = 0.1
	} else if a.Camera.Zoom > 10.0 {
		a.Camera.Zoom = 10.0
	}

	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if a.MapGrid == nil {
		return
	}

	// Phase 08.3: Map Rendering & Biomes
	for y := 0; y < a.MapGrid.Height; y++ {
		for x := 0; x < a.MapGrid.Width; x++ {
			tile := a.MapGrid.GetTile(x, y)
			c, ok := BiomeColors[tile.BiomeID]
			if !ok {
				c = color.RGBA{R: 255, G: 0, B: 255, A: 255} // Magenta for unknown
			}

			// Phase 08.5: Visualizing Desire Paths
			// Dynamic Floor Updates: Override base Biome color if FootTraffic is high
			state := a.MapGrid.TileStates[y*a.MapGrid.Width+x]
			if state.FootTraffic > 100 { // Example RenderThreshold
				// Blend towards dirt color #8B4513 (139, 69, 19) based on traffic
				c = color.RGBA{R: 139, G: 69, B: 19, A: 255}
			}

			// Apply Camera transforms
			worldX := float64(x) * TileSize
			worldY := float64(y) * TileSize
			screenX, screenY := a.Camera.WorldToScreen(worldX, worldY)

			// Draw rect scaled by zoom
			drawSize := TileSize * a.Camera.Zoom
			ebitenutil.DrawRect(screen, screenX, screenY, drawSize, drawSize, c)
		}
	}

	// Phase 08.2: Sub-Tick Interpolation
	// Read alpha float variable produced by Phase 1's TickManager.
	alpha := a.TickManager.Alpha
	world := a.TickManager.World

	if world == nil {
		return
	}

	// Fetch component IDs safely
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	familyID := ecs.ComponentID[components.FamilyCluster](world)
	villageID := ecs.ComponentID[components.Village](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	// Phase 08.4: Entity rendering
	// Wandering AI Clusters (FamilyCluster)
	filterFamily := ecs.All(posID, familyID)
	queryFamily := world.Query(&filterFamily)
	for queryFamily.Next() {
		pos := (*components.Position)(queryFamily.Get(posID))
		drawX := float64(pos.X)
		drawY := float64(pos.Y)

		if queryFamily.Has(velID) {
			vel := (*components.Velocity)(queryFamily.Get(velID))
			drawX += float64(vel.X) * alpha
			drawY += float64(vel.Y) * alpha
		}

		screenX, screenY := a.Camera.WorldToScreen(drawX*TileSize, drawY*TileSize)
		drawSize := 4.0 * a.Camera.Zoom
		ebitenutil.DrawRect(screen, screenX-drawSize/2, screenY-drawSize/2, drawSize, drawSize, color.RGBA{R: 255, G: 0, B: 0, A: 255}) // Red dot
	}

	// Villages
	filterVillage := ecs.All(posID, villageID)
	queryVillage := world.Query(&filterVillage)
	for queryVillage.Next() {
		pos := (*components.Position)(queryVillage.Get(posID))
		screenX, screenY := a.Camera.WorldToScreen(float64(pos.X)*TileSize, float64(pos.Y)*TileSize)
		drawSize := 8.0 * a.Camera.Zoom
		ebitenutil.DrawRect(screen, screenX-drawSize/2, screenY-drawSize/2, drawSize, drawSize, color.RGBA{R: 0, G: 0, B: 255, A: 255}) // Blue node
	}

	// Ruins
	filterRuin := ecs.All(posID, ruinID)
	queryRuin := world.Query(&filterRuin)
	for queryRuin.Next() {
		pos := (*components.Position)(queryRuin.Get(posID))
		screenX, screenY := a.Camera.WorldToScreen(float64(pos.X)*TileSize, float64(pos.Y)*TileSize)
		drawSize := 8.0 * a.Camera.Zoom
		ebitenutil.DrawRect(screen, screenX-drawSize/2, screenY-drawSize/2, drawSize, drawSize, color.RGBA{R: 100, G: 100, B: 100, A: 255}) // Dark gray node
	}
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (a *App) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}
