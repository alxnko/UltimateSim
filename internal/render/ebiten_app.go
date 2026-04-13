package render

import (
	"image/color"
	"sync"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mlange-42/arche/ecs"
)

// LoadingStatus tracks the asynchronous generation of the simulation.
type LoadingStatus struct {
	Progress  float32
	Message   string
	Done      bool
	TM        *engine.TickManager
	Grid      *engine.MapGrid
	HookGraph *engine.SparseHookGraph
	Mutex     sync.Mutex
}

// EbitenApp is the primary 2D Action-RPG renderer and simulation driver.
type EbitenApp struct {
	Status *LoadingStatus
	Zoom   float64
	CamX   float64
	CamY   float64
}

// NewEbitenApp creates a new EbitenApp.
func NewEbitenApp(factory func(int, int, byte, *LoadingStatus)) *EbitenApp {
	status := &LoadingStatus{}
	// Run simulation build in a goroutine
	go factory(1024, 1024, 42, status)
	
	return &EbitenApp{
		Status: status,
		Zoom:   1.0,
		CamX:   512,
		CamY:   512,
	}
}

// Update handles logic and simulation ticks.
func (app *EbitenApp) Update() error {
	app.Status.Mutex.Lock()
	defer app.Status.Mutex.Unlock()

	if !app.Status.Done {
		return nil
	}

	// Drive the simulation tick
	app.Status.TM.Tick()

	// Update Camera based on player position if possessed
	posID := ecs.ComponentID[components.Position](app.Status.TM.World)
	possessedID := ecs.ComponentID[components.Possessed](app.Status.TM.World)
	
	query := app.Status.TM.World.Query(ecs.All(posID, possessedID))
	if query.Next() {
		pos := (*components.Position)(query.Get(posID))
		app.CamX = float64(pos.X)
		app.CamY = float64(pos.Y)
	}
	query.Close()

	return nil
}

// Draw renders the simulation to the screen.
func (app *EbitenApp) Draw(screen *ebiten.Image) {
	app.Status.Mutex.Lock()
	defer app.Status.Mutex.Unlock()

	if !app.Status.Done {
		// Render loading screen
		ebitenutil.DebugPrint(screen, "Loading: "+app.Status.Message)
		return
	}

	// Basic 2D Rendering of the MapGrid
	// In a real implementation, we would use tiles or sprites.
	// For now, let's just clear to a background color and print debug info.
	screen.Fill(color.RGBA{20, 20, 30, 255})
	
	ebitenutil.DebugPrint(screen, "Boundless Sovereigns - 2D Action RPG Mode\nSimulation Ticking at 60 TPS")
}

// Layout defines the logical screen dimensions.
func (app *EbitenApp) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}
