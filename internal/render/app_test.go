package render

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// TestAppDraw verifies that the Draw function does not panic when rendering
// an initialized MapGrid and various ECS entities with Position, Velocity, etc.
func TestAppDraw(t *testing.T) {
	// Initialize MapGrid
	grid := engine.NewMapGrid(10, 10)
	seed := [32]byte{1, 2, 3}
	engine.GenerateMap(grid, seed)

	// Initialize TickManager and World
	tm := engine.NewTickManager(60)
	world := tm.World

	// Register Components
	posID := ecs.ComponentID[components.Position](world)
	velID := ecs.ComponentID[components.Velocity](world)
	familyID := ecs.ComponentID[components.FamilyCluster](world)
	villageID := ecs.ComponentID[components.Village](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	// Spawn Entities
	// 1. FamilyCluster with Velocity
	e1 := world.NewEntity(posID, velID, familyID)
	pos1 := (*components.Position)(world.Get(e1, posID))
	vel1 := (*components.Velocity)(world.Get(e1, velID))
	pos1.X = 1.0
	pos1.Y = 2.0
	vel1.X = 0.5
	vel1.Y = -0.5

	// 2. Village
	e2 := world.NewEntity(posID, villageID)
	pos2 := (*components.Position)(world.Get(e2, posID))
	pos2.X = 5.0
	pos2.Y = 5.0

	// 3. Ruin
	e3 := world.NewEntity(posID, ruinID)
	pos3 := (*components.Position)(world.Get(e3, posID))
	pos3.X = 8.0
	pos3.Y = 8.0

	// Create App
	app := NewApp(tm, grid)
	app.TickManager.Alpha = 0.5 // Sub-tick interpolation value

	// Create an empty Ebitengine image to draw onto
	screen := ebiten.NewImage(800, 600)

	// Ensure Draw does not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("App.Draw panicked: %v", r)
			}
		}()
		app.Draw(screen)
	}()
}
