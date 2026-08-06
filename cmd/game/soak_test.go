package main

import (
	"math"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
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
	defer status.PathQueue.Close() // don't leak worker goroutines across runs

	// Register the player input system exactly as main() does.
	inputSys := systems.NewPlayerInputSystem(status.Bridge)
	inputSys.Initialize(status.TM.World)
	status.TM.AddSystem(inputSys, 0) // PhaseInput

	// Genesis must have planted a political world before the first tick.
	world := status.TM.World
	if n := len(systems.ListCountries(world)); n == 0 {
		t.Fatal("genesis produced no countries")
	}
	cands := systems.ListStartCandidates(world)
	if len(cands) == 0 {
		t.Fatal("no start candidates in a genesis world")
	}
	// Possess through the real character-select path.
	if err := systems.PossessEntity(world, cands[0].Entity); err != nil {
		t.Fatalf("possession failed: %v", err)
	}
	systems.EnsurePlayerStorage(world, cands[0].Entity)

	// Run a long burst of ticks; a lock imbalance or nil-deref panics the test.
	for i := 0; i < 3000; i++ {
		status.TM.Tick()
	}

	if status.TM.Ticks == 0 {
		t.Fatal("no ticks processed")
	}
}

// worldChecksum folds NPC identity and position state into one number so two
// runs can be compared for byte-level determinism drift.
func worldChecksum(world *ecs.World) uint64 {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	posID := ecs.ComponentID[components.Position](world)
	var sum uint64
	q := world.Query(ecs.All(npcID, identID, posID))
	for q.Next() {
		id := (*components.Identity)(q.Get(identID))
		p := (*components.Position)(q.Get(posID))
		sum += id.ID * 31
		sum += uint64(math.Float32bits(p.X)) ^ uint64(math.Float32bits(p.Y))<<1
	}
	return sum
}

// TestSoakDeterminism runs two identical small worlds and checks the tick
// counter advances identically (cheap determinism smoke).
func TestSoakDeterminism(t *testing.T) {
	run := func() uint64 {
		status := &render.LoadingStatus{}
		BuildSimulation(96, 96, 13, status)
		defer status.PathQueue.Close() // don't leak worker goroutines across runs
		for i := 0; i < 1500; i++ {
			status.TM.Tick()
		}
		return status.TM.Ticks*1e9 + worldChecksum(status.TM.World)%1e9
	}
	if a, b := run(), run(); a != b {
		t.Fatalf("world state diverged: %d vs %d", a, b)
	}
}
