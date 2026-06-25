package systems_test

import (
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
	"testing"
)

func TestPlayerInputSystemInitialization(t *testing.T) {
	world := ecs.NewWorld()
	sys := &systems.PlayerInputSystem{}

	// This should fail to compile because PlayerInputSystem is not defined in systems package yet.
	sys.Initialize(&world)
}
