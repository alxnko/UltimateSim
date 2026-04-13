# Action-RPG Complete Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a robust State Machine architecture (`MainMenu`, `Playing`, `GameOver`) with a clean HUD and twin-stick mouse-aim controls.

**Architecture:** Create a `GameState` interface and `StateManager` in `internal/ui`. Refactor `EbitenApp` to delegate all rendering and updating to the active `GameState`. Overhaul `PlayerInputSystem` to handle mouse position aiming and click-based interactions. Wrap the ECS simulation entirely within the `Playing` state.

**Tech Stack:** Go, arche-go (ECS), Ebitengine.

---

### Task 1: Core UI State Machine

**Files:**
- Create: `internal/ui/state.go`
- Create: `internal/ui/state_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/ui/state_test.go
package ui_test

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

type mockState struct {
	updateCalled bool
	drawCalled   bool
}

func (m *mockState) Update(sm *ui.StateManager) error {
	m.updateCalled = true
	return nil
}

func (m *mockState) Draw(screen *ebiten.Image) {
	m.drawCalled = true
}

func TestStateManager(t *testing.T) {
	sm := ui.NewStateManager()
	ms := &mockState{}
	sm.Push(ms)

	sm.Update()
	if !ms.updateCalled {
		t.Error("expected state update to be called")
	}

	sm.Draw(ebiten.NewImage(10, 10))
	if !ms.drawCalled {
		t.Error("expected state draw to be called")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -v`
Expected: FAIL (package ui not found or undefined functions)

- [ ] **Step 3: Write minimal implementation**

```go
// internal/ui/state.go
package ui

import "github.com/hajimehoshi/ebiten/v2"

type GameState interface {
	Update(sm *StateManager) error
	Draw(screen *ebiten.Image)
}

type StateManager struct {
	states []GameState
}

func NewStateManager() *StateManager {
	return &StateManager{
		states: make([]GameState, 0),
	}
}

func (sm *StateManager) Push(state GameState) {
	sm.states = append(sm.states, state)
}

func (sm *StateManager) Pop() {
	if len(sm.states) > 0 {
		sm.states = sm.states[:len(sm.states)-1]
	}
}

func (sm *StateManager) Switch(state GameState) {
	if len(sm.states) > 0 {
		sm.states[len(sm.states)-1] = state
	} else {
		sm.Push(state)
	}
}

func (sm *StateManager) Update() error {
	if len(sm.states) > 0 {
		return sm.states[len(sm.states)-1].Update(sm)
	}
	return nil
}

func (sm *StateManager) Draw(screen *ebiten.Image) {
	if len(sm.states) > 0 {
		sm.states[len(sm.states)-1].Draw(screen)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ui/
git commit -m "feat: add core UI state machine"
```

### Task 2: Implement Main Menu State

**Files:**
- Create: `internal/ui/state_main_menu.go`

- [ ] **Step 1: Write minimal implementation**

```go
// internal/ui/state_main_menu.go
package ui

import (
	"image/color"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type StateMainMenu struct {
	OnStart func()
}

func NewStateMainMenu(onStart func()) *StateMainMenu {
	return &StateMainMenu{OnStart: onStart}
}

func (s *StateMainMenu) Update(sm *StateManager) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if s.OnStart != nil {
			s.OnStart()
		}
	}
	return nil
}

func (s *StateMainMenu) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{10, 10, 20, 255})
	ebitenutil.DebugPrintAt(screen, "BOUNDLESS SOVEREIGNS\n\nPress ENTER to Start", 300, 250)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/ui/state_main_menu.go
git commit -m "feat: implement main menu state"
```

### Task 3: StatePlaying and Rendering Extraction

**Files:**
- Create: `internal/ui/state_playing.go`
- Modify: `internal/render/ebiten_app.go`

- [ ] **Step 1: Create StatePlaying logic**

```go
// internal/ui/state_playing.go
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
```

- [ ] **Step 2: Clean up EbitenApp**

```go
// internal/render/ebiten_app.go
package render

import (
	"sync"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

type LoadingStatus struct {
	Progress  float32
	Message   string
	Done      bool
	TM        *engine.TickManager
	Grid      *engine.MapGrid
	HookGraph *engine.SparseHookGraph
	Mutex     sync.Mutex
}
```

- [ ] **Step 3: Run go build to verify compilation**

Run: `go build ./internal/ui`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add internal/ui/state_playing.go internal/render/ebiten_app.go
git commit -m "refactor: extract rendering into StatePlaying"
```

### Task 4: PlayerInputSystem Mouse Aim Integration

**Files:**
- Modify: `internal/systems/player_input.go`

- [ ] **Step 1: Write minimal implementation**

```go
// internal/systems/player_input.go
package systems

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

type PlayerInputSystem struct {
	filter ecs.Filter
}

func (s *PlayerInputSystem) Initialize(world *ecs.World) {
	s.filter = ecs.All(
		ecs.ComponentID[components.Possessed](world),
		ecs.ComponentID[components.Velocity](world),
	)
}

func (s *PlayerInputSystem) Update(world *ecs.World) {
	query := world.Query(s.filter)
	for query.Next() {
		vel := (*components.Velocity)(query.Get(ecs.ComponentID[components.Velocity](world)))
		
		vel.X = 0
		vel.Y = 0
		speed := float32(2.0)
		
		if ebiten.IsKeyPressed(ebiten.KeyW) { vel.Y = -speed }
		if ebiten.IsKeyPressed(ebiten.KeyS) { vel.Y = speed }
		if ebiten.IsKeyPressed(ebiten.KeyA) { vel.X = -speed }
		if ebiten.IsKeyPressed(ebiten.KeyD) { vel.X = speed }

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cx, cy := ebiten.CursorPosition()
			fmt.Printf("Attack Triggered towards screen pos: %d, %d\n", cx, cy)
		}
		
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			cx, cy := ebiten.CursorPosition()
			fmt.Printf("Interact Triggered towards screen pos: %d, %d\n", cx, cy)
		}
	}
}
```

- [ ] **Step 2: Run go build**

Run: `go build ./internal/systems`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add internal/systems/player_input.go
git commit -m "feat: add mouse click hooks to PlayerInputSystem"
```

### Task 5: Main App Wiring

**Files:**
- Modify: `cmd/game/main.go`

- [ ] **Step 1: Rewrite main.go**

```go
// cmd/game/main.go
// Keep BuildSimulation exactly as it is now.
// Rewrite the main() function:

type Game struct {
	sm *ui.StateManager
}

func (g *Game) Update() error {
	return g.sm.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.sm.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	// Setup PPROF... (keep existing flag parsing logic)
	// ...

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Boundless Sovereigns - Action RPG")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	sm := ui.NewStateManager()
	
	status := &render.LoadingStatus{}
	
	// Create Playing State first (but don't push it yet)
	playingState := ui.NewStatePlaying(status)

	// Create Main Menu
	menuState := ui.NewStateMainMenu(func() {
		// When "Start" is clicked, push the playing state
		sm.Switch(playingState)
	})

	// Start the async engine build
	go func() {
		BuildSimulation(1024, 1024, 42, status)
		// Register Input System manually here after it's built
		status.Mutex.Lock()
		inputSys := &systems.PlayerInputSystem{}
		inputSys.Initialize(status.TM.World)
		status.TM.AddSystem(inputSys, engine.PhaseInput)
		
		directorSys := systems.NewPlayerDirectorSystem(status.HookGraph)
		directorSys.Initialize(status.TM.World)
		status.TM.AddSystem(directorSys, engine.PhaseResolution)
		status.Mutex.Unlock()
	}()

	sm.Push(menuState)

	game := &Game{sm: sm}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 2: Add UI import to main**

Run: `go build ./cmd/game`
Expected: FAIL (missing import for `github.com/ALXNKO/UltimateSim/internal/ui`)
Fix: Add `"github.com/ALXNKO/UltimateSim/internal/ui"` to the `import` block in `cmd/game/main.go`.

- [ ] **Step 3: Run final compilation check**

Run: `go build ./cmd/game`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add cmd/game/main.go
git commit -m "refactor: wire up complete UI state machine in main loop"
```
