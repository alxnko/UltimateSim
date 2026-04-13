# 2D Action-RPG Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the game from a 3D/2D hybrid to a pure 2D Action-RPG where the player directly controls a single possessed NPC via Ebitengine.

**Architecture:** Remove `raylib-go`. Create a `Possessed` ECS component. Modify `WanderSystem` to skip possessed entities. Add `PlayerInputSystem` to handle WASD/Mouse via `Ebitengine` and map to `Velocity`/Interactions. Render the world purely in 2D using `Ebitengine`.

**Tech Stack:** Go, arche-go (ECS), Ebitengine.

---

### Task 1: Add Possessed Component

**Files:**
- Modify: `internal/components/basic.go`
- Modify: `internal/components/basic_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/components/basic_test.go
package components_test

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"ultimatesim/internal/components"
)

func TestPossessedComponent(t *testing.T) {
	world := ecs.NewWorld()
	posComp := ecs.ComponentID[components.Possessed](&world)
	e := world.NewEntity(posComp)
	
	if !world.Has(e, posComp) {
		t.Errorf("expected entity to have Possessed component")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/components -run TestPossessedComponent -v`
Expected: FAIL (undefined: components.Possessed)

- [ ] **Step 3: Write minimal implementation**

```go
// Add to internal/components/basic.go
package components

// Possessed is a marker component indicating that this entity is currently controlled by the player.
type Possessed struct{}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/components -run TestPossessedComponent -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/components/basic.go internal/components/basic_test.go
git commit -m "feat: add Possessed ECS component"
```

### Task 2: Modify WanderSystem to Skip Possessed Entities

**Files:**
- Modify: `internal/systems/wander.go`
- Add: `internal/systems/wander_possessed_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Create internal/systems/wander_possessed_test.go
package systems_test

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"ultimatesim/internal/components"
	"ultimatesim/internal/systems"
)

func TestWanderSystemSkipsPossessed(t *testing.T) {
	world := ecs.NewWorld()
	
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	possessedID := ecs.ComponentID[components.Possessed](&world)
	
	// Create player entity
	player := world.NewEntity(posID, velID, possessedID)
	
	wanderSys := &systems.WanderSystem{}
	wanderSys.Initialize(&world)
	
	// Tick
	wanderSys.Update(&world)
	
	// Player velocity should not be modified by wander
	vel := (*components.Velocity)(world.Get(player, velID))
	if vel.X != 0 || vel.Y != 0 {
		t.Errorf("WanderSystem should not mutate Possessed entity velocity")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/systems -run TestWanderSystemSkipsPossessed -v`
Expected: FAIL (or it might pass if WanderSystem requires more components, but we want to explicitly exclude Possessed via filter)

- [ ] **Step 3: Write minimal implementation**

```go
// Modify internal/systems/wander.go
// Inside Initialize(), update the filter to exclude Possessed:
// filter := ecs.All(posID, velID, needsID).Without(ecs.ComponentID[components.Possessed](world))
// Ensure the system ignores player entities.
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/systems -run TestWanderSystemSkipsPossessed -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/systems/wander.go internal/systems/wander_possessed_test.go
git commit -m "feat: WanderSystem skips possessed entities"
```

### Task 3: Create PlayerInputSystem

**Files:**
- Create: `internal/systems/player_input.go`
- Create: `internal/systems/player_input_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/systems/player_input_test.go
package systems_test

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"ultimatesim/internal/components"
	"ultimatesim/internal/systems"
)

func TestPlayerInputSystem(t *testing.T) {
	world := ecs.NewWorld()
	sys := &systems.PlayerInputSystem{}
	sys.Initialize(&world)
	// Just ensure it initializes without panicking for now
	// Mocking ebiten input requires specialized setup, but system registration must work.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/systems -run TestPlayerInputSystem -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
// internal/systems/player_input.go
package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
	"ultimatesim/internal/components"
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
		
		if ebiten.IsKeyPressed(ebiten.KeyW) { vel.Y -= 1.0 }
		if ebiten.IsKeyPressed(ebiten.KeyS) { vel.Y += 1.0 }
		if ebiten.IsKeyPressed(ebiten.KeyA) { vel.X -= 1.0 }
		if ebiten.IsKeyPressed(ebiten.KeyD) { vel.X += 1.0 }
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/systems -run TestPlayerInputSystem -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/systems/player_input.go internal/systems/player_input_test.go
git commit -m "feat: add PlayerInputSystem for WASD movement"
```

### Task 4: Remove raylib-go and Setup EbitenApp Renderer

**Files:**
- Delete: `internal/render/raylib_app.go`
- Add: `internal/render/ebiten_app.go`
- Modify: `cmd/game/main.go`

- [ ] **Step 1: Delete raylib-go**

Run: `rm internal/render/raylib_app.go`
Run: `go mod tidy`

- [ ] **Step 2: Create EbitenApp Implementation**

```go
// internal/render/ebiten_app.go
package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

type EbitenApp struct {
	World *ecs.World
}

func (app *EbitenApp) Update() error {
	// Call TickManager or directly update ECS world
	return nil
}

func (app *EbitenApp) Draw(screen *ebiten.Image) {
	// Render 2D grid and entities
}

func (app *EbitenApp) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}
```

- [ ] **Step 3: Modify Main**

```go
// cmd/game/main.go
package main

import (
	"log"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
	"ultimatesim/internal/render"
	"ultimatesim/internal/systems"
)

func main() {
	world := ecs.NewWorld()
	
	// Add systems
	// ... (register PlayerInputSystem, etc.)
	
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Boundless Sovereigns")
	
	app := &render.EbitenApp{World: &world}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Run go build to verify compilation**

Run: `go build ./cmd/game`
Expected: Success

- [ ] **Step 5: Commit**

```bash
git add internal/render/ cmd/game/main.go go.mod go.sum
git commit -m "refactor: remove raylib-go and set up unified ebitengine 2D app"
```

### Task 5: 2D Camera Follow

**Files:**
- Modify: `internal/render/ebiten_app.go`

- [ ] **Step 1: Implement Camera Follow logic**

```go
// internal/render/ebiten_app.go
// Inside Update(), find the possessed entity and update app.CamX, app.CamY.
```

- [ ] **Step 2: Run go build to verify compilation**

Run: `go build ./cmd/game`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git add internal/render/ebiten_app.go
git commit -m "feat: implement 2D camera follow for possessed entity"
```

### Task 6: PlayerDirectorSystem (Emergent Quests)

**Files:**
- Create: `internal/systems/player_director.go`
- Modify: `cmd/game/main.go`

- [ ] **Step 1: Create PlayerDirectorSystem**

```go
// internal/systems/player_director.go
package systems

import (
	"fmt"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

type PlayerDirectorSystem struct {
	filter ecs.Filter
}

func (s *PlayerDirectorSystem) Initialize(world *ecs.World) {
	s.filter = ecs.All(ecs.ComponentID[components.Possessed](world))
}

func (s *PlayerDirectorSystem) Update(world *ecs.World) {
	// Logic to evaluate local entropy and suggest actions to the player
}
```

- [ ] **Step 2: Register in main.go**

```go
// cmd/game/main.go
// Add systems.PlayerDirectorSystem to the factory.
```

- [ ] **Step 3: Commit**

```bash
git add internal/systems/player_director.go cmd/game/main.go
git commit -m "feat: add PlayerDirectorSystem for emergent gameplay hooks"
```
