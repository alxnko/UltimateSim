package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

func TestRustSystem(t *testing.T) {
	// Initialize Deterministic RNG
	engine.InitializeRNG([32]byte{1, 2, 3})

	tm := engine.NewTickManager(60)
	sys := systems.NewRustSystem()
	tm.AddSystem(sys, engine.PhaseResolution)

	world := tm.World

	storageID := ecs.ComponentID[components.StorageComponent](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	// Create a Village with StorageComponent
	villageEntity := world.NewEntity(storageID)
	storage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	storage.Iron = 1000
	storage.Wood = 1000

	// Create a Caravan with Payload
	caravanEntity := world.NewEntity(payloadID)
	payload := (*components.Payload)(world.Get(caravanEntity, payloadID))
	payload.Iron = 500
	payload.Wood = 500

	// Tick 49 times -> No change
	for i := 0; i < 49; i++ {
		tm.Tick()
	}

	if storage.Iron != 1000 || storage.Wood != 1000 {
		t.Errorf("Expected Village Iron/Wood to be 1000 at tick 49, got %d/%d", storage.Iron, storage.Wood)
	}
	if payload.Iron != 500 || payload.Wood != 500 {
		t.Errorf("Expected Caravan Iron/Wood to be 500 at tick 49, got %d/%d", payload.Iron, payload.Wood)
	}

	// Tick 1 more time -> Rust should occur (50th tick)
	tm.Tick()

	expectedVillageIron := uint32((1000 * 98) / 100) // 980
	if storage.Iron != expectedVillageIron || storage.Wood != expectedVillageIron {
		t.Errorf("Expected Village Iron/Wood to be %d, got %d/%d", expectedVillageIron, storage.Iron, storage.Wood)
	}

	expectedCaravanIron := uint32((500 * 98) / 100) // 490
	if payload.Iron != expectedCaravanIron || payload.Wood != expectedCaravanIron {
		t.Errorf("Expected Caravan Iron/Wood to be %d, got %d/%d", expectedCaravanIron, payload.Iron, payload.Wood)
	}

	// Small value test: At 10, 2% is 0.2 -> 0 in integer math.
	// But system forces at least 1 unit decremented.
	storage.Iron = 10
	for i := 0; i < 50; i++ {
		tm.Tick()
	}

	if storage.Iron != 9 {
		t.Errorf("Expected Village Iron to decrease to 9, got %d", storage.Iron)
	}
}
