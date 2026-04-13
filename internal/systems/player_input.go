package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
)

// PlayerInputSystem translates real-time user input (WASD) into physical ECS mutations.
type PlayerInputSystem struct {
	filter ecs.Filter
}

// Initialize sets up the Arche-Go filter for Possessed entities.
func (s *PlayerInputSystem) Initialize(world *ecs.World) {
	s.filter = ecs.All(
		ecs.ComponentID[components.Possessed](world),
		ecs.ComponentID[components.Velocity](world),
	)
}

// Update reads Ebitengine keyboard states and updates the Velocity of possessed entities.
func (s *PlayerInputSystem) Update(world *ecs.World) {
	query := world.Query(s.filter)
	for query.Next() {
		vel := (*components.Velocity)(query.Get(ecs.ComponentID[components.Velocity](world)))
		
		// Reset velocity to avoid sliding unless keys are pressed
		vel.X = 0
		vel.Y = 0
		
		speed := float32(2.0)
		
		if ebiten.IsKeyPressed(ebiten.KeyW) {
			vel.Y = -speed
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) {
			vel.Y = speed
		}
		if ebiten.IsKeyPressed(ebiten.KeyA) {
			vel.X = -speed
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) {
			vel.X = speed
		}
	}
}
