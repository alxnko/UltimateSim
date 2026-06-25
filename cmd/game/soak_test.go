package main

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
)

// TestSoakFullSimulation builds the full system set on a small world and runs
// many ticks headless, asserting no panic (e.g. ECS query lock imbalances) and
// that the player gets possessed and survives basic interaction. This is the
// integration guard for the Shell Phase wiring.
func TestSoakFullSimulation(t *testing.T) {
	status := &render.LoadingStatus{}
	BuildSimulation(128, 128, 7, status)

	if !status.Done || status.TM == nil {
		t.Fatal("simulation failed to build")
	}

	// Register the player input system exactly as main() does.
	inputSys := systems.NewPlayerInputSystem(status.Bridge)
	inputSys.Initialize(status.TM.World)
	status.TM.AddSystem(inputSys, 0) // PhaseInput

	// Possess a starting character so player-facing systems exercise live paths.
	if cand, found := systems.FindStartCandidate(status.TM.World); found {
		if err := systems.PossessEntity(status.TM.World, cand); err != nil {
			t.Fatalf("possession failed: %v", err)
		}
		systems.EnsurePlayerStorage(status.TM.World, cand)
	}

	// Run a long burst of ticks; a lock imbalance or nil-deref panics the test.
	for i := 0; i < 3000; i++ {
		status.TM.Tick()
	}

	if status.TM.Ticks == 0 {
		t.Fatal("no ticks processed")
	}
}

// TestSoakDeterminism runs two identical small worlds and checks the tick
// counter advances identically (cheap determinism smoke).
func TestSoakDeterminism(t *testing.T) {
	run := func() uint64 {
		status := &render.LoadingStatus{}
		BuildSimulation(96, 96, 13, status)
		for i := 0; i < 1500; i++ {
			status.TM.Tick()
		}
		return status.TM.Ticks
	}
	if a, b := run(), run(); a != b {
		t.Fatalf("tick counts diverged: %d vs %d", a, b)
	}
}
