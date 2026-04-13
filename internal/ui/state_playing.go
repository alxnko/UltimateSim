package ui

import (
	"image/color"
	"math"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mlange-42/arche/ecs"
)

type StatePlaying struct {
	Status *render.LoadingStatus
	Zoom   float64
	CamX   float64
	CamY   float64
}

func NewStatePlaying(status *render.LoadingStatus) *StatePlaying {
	return &StatePlaying{
		Status: status,
		Zoom:   1.0,
		CamX:   512,
		CamY:   512,
	}
}

func (s *StatePlaying) Update(sm *StateManager) error {
	s.Status.Mutex.Lock()
	defer s.Status.Mutex.Unlock()

	if !s.Status.Done {
		return nil
	}

	// Auto-Possess nearest NPC if no one is possessed (temporary failsafe)
	world := s.Status.TM.World
	posID := ecs.ComponentID[components.Position](world)
	possessedID := ecs.ComponentID[components.Possessed](world)
	npcID := ecs.ComponentID[components.NPC](world)

	qCheck := world.Query(ecs.All(possessedID))
	hasPossessed := qCheck.Next()
	qCheck.Close()

	if !hasPossessed {
		qNPC := world.Query(ecs.All(npcID))
		if qNPC.Next() {
			world.Add(qNPC.Entity(), possessedID)
		}
		qNPC.Close()
	}

	s.Status.TM.Tick()

	query := world.Query(ecs.All(posID, possessedID))
	if query.Next() {
		pos := (*components.Position)(query.Get(posID))
		s.CamX = float64(pos.X)
		s.CamY = float64(pos.Y)
	}
	query.Close()

	return nil
}

func (s *StatePlaying) Draw(screen *ebiten.Image) {
	s.Status.Mutex.Lock()
	defer s.Status.Mutex.Unlock()

	if !s.Status.Done {
		screen.Fill(color.RGBA{20, 20, 30, 255})
		ebitenutil.DebugPrintAt(screen, "Loading: "+s.Status.Message, 10, 10)
		return
	}

	screen.Fill(color.RGBA{20, 20, 30, 255})

	tileSize := 16.0 * s.Zoom
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	cols := int(float64(screenWidth)/tileSize) + 2
	rows := int(float64(screenHeight)/tileSize) + 2

	startX := int(s.CamX) - cols/2
	startY := int(s.CamY) - rows/2

	offsetX := -math.Mod(s.CamX, 1.0) * tileSize
	offsetY := -math.Mod(s.CamY, 1.0) * tileSize

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tx, ty := startX+c, startY+r
			if tx < 0 || tx >= s.Status.Grid.Width || ty < 0 || ty >= s.Status.Grid.Height {
				continue
			}
			tile := s.Status.Grid.GetTile(tx, ty)
			clr := getBiomeColor(tile.BiomeID)

			x := float64(c)*tileSize + offsetX
			y := float64(r)*tileSize + offsetY
			ebitenutil.DrawRect(screen, x, y, tileSize, tileSize, clr)
		}
	}

	posID := ecs.ComponentID[components.Position](s.Status.TM.World)
	possessedID := ecs.ComponentID[components.Possessed](s.Status.TM.World)

	query := s.Status.TM.World.Query(ecs.All(posID))
	for query.Next() {
		pos := (*components.Position)(query.Get(posID))
		isPossessed := query.Has(possessedID)

		dx, dy := float64(pos.X)-s.CamX, float64(pos.Y)-s.CamY
		sx := float64(screenWidth)/2 + dx*tileSize
		sy := float64(screenHeight)/2 + dy*tileSize

		if sx < -tileSize || sx > float64(screenWidth)+tileSize || sy < -tileSize || sy > float64(screenHeight)+tileSize {
			continue
		}

		clr := color.RGBA{255, 0, 0, 255}
		if isPossessed {
			clr = color.RGBA{255, 255, 0, 255}
		}

		ebitenutil.DrawRect(screen, sx-4, sy-4, 8, 8, clr)
	}
	query.Close()
	
	// HUD Overlays
	ebitenutil.DebugPrintAt(screen, "HP: 100/100 | Food: 80/100", 10, 10)
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
		return color.RGBA{255, 0, 255, 255}
	}
}
