package engine

import (
	"testing"
	"time"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Dummy MovementSystem for testing
type MovementSystem struct{}

func (s *MovementSystem) Update(world *ecs.World) {
	// Query entities with Position and Velocity
	filter := ecs.All(
		ecs.ComponentID[components.Position](world),
		ecs.ComponentID[components.Velocity](world),
	)

	query := world.Query(filter)
	for query.Next() {
		pos := (*components.Position)(query.Get(ecs.ComponentID[components.Position](world)))
		vel := (*components.Velocity)(query.Get(ecs.ComponentID[components.Velocity](world)))
		pos.X += vel.X
		pos.Y += vel.Y
	}
}

func TestTickManager_60TPS(t *testing.T) {
	tm := NewTickManager(60)
	tm.AddSystem(&MovementSystem{}, PhaseMovement)

	start := time.Now()
	tm.Run(60) // Run exactly 60 ticks
	duration := time.Since(start)

	expected := time.Second
	// The sleep precision could be slightly off depending on the OS scheduler,
	// so allow some delta. Widen tolerance for CI run environments.
	tolerance := 150 * time.Millisecond
	if duration < expected-tolerance || duration > expected+tolerance {
		t.Errorf("60 ticks took %v, expected roughly %v", duration, expected)
	}
}

func runDeterministicSimulation(seed [32]byte, ticks int) []components.Position {
	InitializeRNG(seed)

	tm := NewTickManager(60)
	tm.AddSystem(&MovementSystem{}, PhaseMovement)

	// Spawn 100 entities with random velocities using our seeded RNG.
	for i := 0; i < 100; i++ {
		entity := tm.World.NewEntity()
		posID := ecs.ComponentID[components.Position](tm.World)
		velID := ecs.ComponentID[components.Velocity](tm.World)

		tm.World.Add(entity, posID, velID)

		// Map GetRandomInt (which is arbitrary sized) to float32 velocity between -5 and 5
		velX := (float32(GetRandomInt()%100) / 10.0) - 5.0
		velY := (float32(GetRandomInt()%100) / 10.0) - 5.0

		vel := (*components.Velocity)(tm.World.Get(entity, velID))
		vel.X = velX
		vel.Y = velY

		pos := (*components.Position)(tm.World.Get(entity, posID))
		pos.X = 0
		pos.Y = 0
	}

	tm.Run(ticks)

	// Collect final positions
	var finalPositions []components.Position
	filter := ecs.All(ecs.ComponentID[components.Position](tm.World))
	query := tm.World.Query(filter)
	for query.Next() {
		pos := (*components.Position)(query.Get(ecs.ComponentID[components.Position](tm.World)))
		finalPositions = append(finalPositions, *pos)
	}

	return finalPositions
}

func TestTickManager_Determinism(t *testing.T) {
	seed := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	ticks := 120

	run1 := runDeterministicSimulation(seed, ticks)
	run2 := runDeterministicSimulation(seed, ticks)

	if len(run1) != len(run2) {
		t.Fatalf("Length mismatch: run1 %d, run2 %d", len(run1), len(run2))
	}

	for i := 0; i < len(run1); i++ {
		if run1[i].X != run2[i].X || run1[i].Y != run2[i].Y {
			t.Errorf("Determinism failure at index %d: run1 %+v, run2 %+v", i, run1[i], run2[i])
		}
	}
}

// Phase 01.3: SystemRunner Sequencing Test
type TrackerSystem struct {
	PhaseName string
	ExecutionLog *[]string
}

func (s *TrackerSystem) Update(world *ecs.World) {
	*s.ExecutionLog = append(*s.ExecutionLog, s.PhaseName)
}

func TestSystemRunner_Sequencing(t *testing.T) {
	tm := NewTickManager(60)
	var log []string

	// Add systems out of phase order
	tm.AddSystem(&TrackerSystem{PhaseName: "Resolution", ExecutionLog: &log}, PhaseResolution)
	tm.AddSystem(&TrackerSystem{PhaseName: "Input", ExecutionLog: &log}, PhaseInput)
	tm.AddSystem(&TrackerSystem{PhaseName: "Cleanup", ExecutionLog: &log}, PhaseCleanup)
	tm.AddSystem(&TrackerSystem{PhaseName: "Movement", ExecutionLog: &log}, PhaseMovement)
	tm.AddSystem(&TrackerSystem{PhaseName: "AI", ExecutionLog: &log}, PhaseAI)
	tm.AddSystem(&TrackerSystem{PhaseName: "AI_2", ExecutionLog: &log}, PhaseAI)

	tm.Tick()

	expectedOrder := []string{"Input", "AI", "AI_2", "Movement", "Resolution", "Cleanup"}

	if len(log) != len(expectedOrder) {
		t.Fatalf("Expected %d executions, got %d", len(expectedOrder), len(log))
	}

	for i, expected := range expectedOrder {
		if log[i] != expected {
			t.Errorf("Order mismatch at %d: expected %s, got %s", i, expected, log[i])
		}
	}
}
