package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 08.1: Window Management & Camera
// Establish window bounds and input capture mapping to camera vectors (Pan, Zoom).

// App implements ebiten.Game interface
type App struct {
	Camera      *Camera
	TickManager *engine.TickManager
}

// NewApp creates a new application instance.
func NewApp(tm *engine.TickManager) *App {
	return &App{
		Camera:      NewCamera(),
		TickManager: tm,
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
	// We'll leave Draw empty for now until Map rendering and Entity rendering are implemented.
	// Sub-Tick Interpolation will use a.TickManager.Alpha here.
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (a *App) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}
