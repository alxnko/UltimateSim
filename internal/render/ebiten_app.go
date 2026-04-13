package render

import (
	"image/color"
	"math"
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
		query.Close()
	}

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
	tileSize := 16.0 * app.Zoom
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	
	cols := int(float64(screenWidth)/tileSize) + 2
	rows := int(float64(screenHeight)/tileSize) + 2
	
	startX := int(app.CamX) - cols/2
	startY := int(app.CamY) - rows/2
	
	offsetX := -math.Mod(app.CamX, 1.0) * tileSize
	offsetY := -math.Mod(app.CamY, 1.0) * tileSize

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tx, ty := startX+c, startY+r
			if tx < 0 || tx >= app.Status.Grid.Width || ty < 0 || ty >= app.Status.Grid.Height {
				continue
			}
			
			tile := app.Status.Grid.GetTile(tx, ty)
			clr := getBiomeColor(tile.BiomeID)
			
			// Draw simple rectangle for tile
			x := float64(c)*tileSize + offsetX
			y := float64(r)*tileSize + offsetY
			
			ebitenutil.DrawRect(screen, x, y, tileSize, tileSize, clr)
		}
	}
	
	// Render Entities (NPCs)
	posID := ecs.ComponentID[components.Position](app.Status.TM.World)
	possessedID := ecs.ComponentID[components.Possessed](app.Status.TM.World)
	
	query := app.Status.TM.World.Query(ecs.All(posID))
	for query.Next() {
		pos := (*components.Position)(query.Get(posID))
		isPossessed := query.Has(possessedID)
		
		// Map position to screen coordinates
		dx, dy := float64(pos.X)-app.CamX, float64(pos.Y)-app.CamY
		sx := float64(screenWidth)/2 + dx*tileSize
		sy := float64(screenHeight)/2 + dy*tileSize
		
		if sx < -tileSize || sx > float64(screenWidth)+tileSize || sy < -tileSize || sy > float64(screenHeight)+tileSize {
			continue
		}
		
		clr := color.RGBA{255, 0, 0, 255} // Red for NPCs
		if isPossessed {
			clr = color.RGBA{255, 255, 0, 255} // Yellow for Player
		}
		
		ebitenutil.DrawRect(screen, sx-4, sy-4, 8, 8, clr)
	}

	ebitenutil.DebugPrint(screen, "Boundless Sovereigns - 2D Action RPG Mode\nSimulation Ticking at 60 TPS")
}

func getBiomeColor(biomeID uint8) color.RGBA {
	switch biomeID {
	case engine.BiomeOcean:
		return color.RGBA{0, 0, 128, 255}
	case engine.BiomeBeach:
		return color.RGBA{210, 180, 140, 255}
	case engine.BiomeScorched:
		return color.RGBA{139, 69, 19, 255}
	case engine.BiomeBare:
		return color.RGBA{160, 160, 160, 255}
	case engine.BiomeTundra:
		return color.RGBA{220, 220, 220, 255}
	case engine.BiomeSnow:
		return color.RGBA{255, 255, 255, 255}
	case engine.BiomeTemperateDesert:
		return color.RGBA{194, 178, 128, 255}
	case engine.BiomeShrubland:
		return color.RGBA{124, 141, 75, 255}
	case engine.BiomeGrassland:
		return color.RGBA{50, 205, 50, 255}
	case engine.BiomeTemperateDeciduousForest:
		return color.RGBA{34, 139, 34, 255}
	case engine.BiomeTemperateRainForest:
		return color.RGBA{0, 100, 0, 255}
	case engine.BiomeSubtropicalDesert:
		return color.RGBA{210, 105, 30, 255}
	case engine.BiomeTropicalSeasonalForest:
		return color.RGBA{85, 107, 47, 255}
	case engine.BiomeTropicalRainForest:
		return color.RGBA{0, 64, 0, 255}
	case engine.BiomeMountain:
		return color.RGBA{105, 105, 105, 255}
	default:
		return color.RGBA{255, 0, 255, 255} // Error magenta
	}
}

// Layout defines the logical screen dimensions.
func (app *EbitenApp) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}
