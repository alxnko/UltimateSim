package ui

import (
	"image"
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: top-right minimap. The terrain backdrop is rasterized once from
// the biome grid; live markers (player, settlements) refresh every frame.

type Minimap struct {
	base    *ebiten.Image
	scale   float64 // world units per minimap pixel
	lastDot uint64
}

// ensureBase rasterizes the biome backdrop once.
func (m *Minimap) ensureBase(grid *engine.MapGrid) {
	if m.base != nil {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, MinimapSize, MinimapSize))
	m.scale = float64(grid.Width) / float64(MinimapSize)
	for y := 0; y < MinimapSize; y++ {
		for x := 0; x < MinimapSize; x++ {
			tx := int(float64(x) * m.scale)
			ty := int(float64(y) * m.scale)
			if tx >= grid.Width {
				tx = grid.Width - 1
			}
			if ty >= grid.Height {
				ty = grid.Height - 1
			}
			img.Set(x, y, getBiomeColor(grid.GetTile(tx, ty).BiomeID))
		}
	}
	m.base = ebiten.NewImageFromImage(img)
}

// Draw renders the minimap with live markers and the camera viewport.
func (m *Minimap) Draw(screen *ebiten.Image, grid *engine.MapGrid, world *ecs.World, sw int, cam *Camera) {
	m.ensureBase(grid)
	x0 := float64(sw - MinimapSize - 8)
	y0 := 8.0

	ebitenutil.DrawRect(screen, x0-2, y0-2, MinimapSize+4, MinimapSize+4, PanelBorder)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x0, y0)
	screen.DrawImage(m.base, op)

	// Camera viewport rectangle.
	if cam != nil {
		sh := screen.Bounds().Dy()
		ts := cam.TileSize()
		vw := float64(sw) / ts / m.scale
		vh := float64(sh) / ts / m.scale
		vx := x0 + cam.X/m.scale - vw/2
		vy := y0 + cam.Y/m.scale - vh/2
		ebitenutil.DrawRect(screen, vx, vy, vw, 1, AccentCol)
		ebitenutil.DrawRect(screen, vx, vy+vh, vw, 1, AccentCol)
		ebitenutil.DrawRect(screen, vx, vy, 1, vh, AccentCol)
		ebitenutil.DrawRect(screen, vx+vw, vy, 1, vh, AccentCol)
	}

	toMini := func(wx, wy float32) (float64, float64) {
		return x0 + float64(wx)/m.scale, y0 + float64(wy)/m.scale
	}

	posID := ecs.ComponentID[components.Position](world)
	villageID := ecs.ComponentID[components.Village](world)
	capID := ecs.ComponentID[components.CapitalComponent](world)
	q := world.Query(ecs.All(villageID, posID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		mx, my := toMini(pos.X, pos.Y)
		clr := color.RGBA{240, 240, 240, 255}
		if q.Has(capID) {
			clr = color.RGBA{255, 80, 80, 255}
		}
		ebitenutil.DrawRect(screen, mx-1, my-1, 2, 2, clr)
	}

	possessedID := ecs.ComponentID[components.Possessed](world)
	q = world.Query(ecs.All(possessedID, posID))
	if q.Next() {
		pos := (*components.Position)(q.Get(posID))
		mx, my := toMini(pos.X, pos.Y)
		ebitenutil.DrawRect(screen, mx-2, my-2, 4, 4, AccentCol)
		q.Close()
	}
}
