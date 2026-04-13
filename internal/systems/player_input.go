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
