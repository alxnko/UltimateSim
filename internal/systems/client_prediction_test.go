package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/network"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 12.3: Robust Client Prediction & Smoothing E2E Tests

func TestClientPredictionSystem_Smoothing(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	idID := ecs.ComponentID[components.Identity](&world)

	predictSys := systems.NewClientPredictionSystem(&world)

	// Spawn Entity
	e1 := world.NewEntity()
	world.Add(e1, posID, idID)

	// Set initial local state
	pos := (*components.Position)(world.Get(e1, posID))
	pos.X = 10.0
	pos.Y = 10.0

	id := (*components.Identity)(world.Get(e1, idID))
	id.ID = 555

	// Server says the entity is actually at 20, 20
	deltas := []network.PositionDelta{
		{EntityID: 555, X: 20.0, Y: 20.0},
	}
	predictSys.QueueDeltas(deltas)

	// Tick 1: Expect 10.0 -> interpolate -> 11.0
	predictSys.Update(&world)

	if pos.X != 11.0 || pos.Y != 11.0 {
		t.Errorf("Expected Pos.X/Y to interpolate to 11.0, got X:%f Y:%f", pos.X, pos.Y)
	}

	// Re-queue same delta for tick 2
	predictSys.QueueDeltas(deltas)

	// Tick 2: Expect 11.0 -> interpolate -> 11.9
	predictSys.Update(&world)

	// Using tiny epsilon for float comparison
	epsilon := float32(0.0001)
	if (pos.X < 11.9-epsilon || pos.X > 11.9+epsilon) || (pos.Y < 11.9-epsilon || pos.Y > 11.9+epsilon) {
		t.Errorf("Expected Pos.X/Y to interpolate to 11.9, got X:%f Y:%f", pos.X, pos.Y)
	}
}

func TestClientPredictionSystem_Deterministic(t *testing.T) {
	// Function to simulate identical networking overrides over multiple ticks
	runSim := func() float32 {
		world := ecs.NewWorld()

		posID := ecs.ComponentID[components.Position](&world)
		idID := ecs.ComponentID[components.Identity](&world)

		predictSys := systems.NewClientPredictionSystem(&world)

		// Spawn 10 identical entities
		for i := uint64(1); i <= 10; i++ {
			e := world.NewEntity(posID, idID)
			pos := (*components.Position)(world.Get(e, posID))
			pos.X = float32(i) * 5.0
			pos.Y = float32(i) * 5.0

			id := (*components.Identity)(world.Get(e, idID))
			id.ID = i
		}

		// Apply server deltas moving them all to 0,0
		var deltas []network.PositionDelta
		for i := uint64(1); i <= 10; i++ {
			deltas = append(deltas, network.PositionDelta{
				EntityID: i,
				X:        0,
				Y:        0,
			})
		}

		// Run 5 ticks of smoothing
		for tick := 0; tick < 5; tick++ {
			predictSys.QueueDeltas(deltas)
			predictSys.Update(&world)
		}

		// Return sum of all positions for verification
		var sum float32
		query := world.Query(ecs.All(posID))
		for query.Next() {
			pos := (*components.Position)(query.Get(posID))
			sum += pos.X + pos.Y
		}
		return sum
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed for ClientPredictionSystem: Run 1 gave %f, Run 2 gave %f", result1, result2)
	}
}
