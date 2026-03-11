package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

func TestSpoilageSystem(t *testing.T) {
	// Initialize Deterministic RNG
	engine.InitializeRNG([32]byte{1, 2, 3})

	tm := engine.NewTickManager(60)
	sys := systems.NewSpoilageSystem()
	tm.AddSystem(sys, engine.PhaseResolution)

	world := tm.World

	storageID := ecs.ComponentID[components.StorageComponent](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	// Create a Village with StorageComponent
	villageEntity := world.NewEntity(storageID)
	storage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	storage.Food = 1000

	// Create a Caravan with Payload
	caravanEntity := world.NewEntity(payloadID)
	payload := (*components.Payload)(world.Get(caravanEntity, payloadID))
	payload.Food = 500

	// Tick 9 times -> No change
	for i := 0; i < 9; i++ {
		tm.Tick()
	}

	if storage.Food != 1000 {
		t.Errorf("Expected Village Food to be 1000 at tick 9, got %d", storage.Food)
	}
	if payload.Food != 500 {
		t.Errorf("Expected Caravan Food to be 500 at tick 9, got %d", payload.Food)
	}

	// Tick 1 more time -> Spoilage should occur (10th tick)
	tm.Tick()

	expectedVillageFood := uint32((1000 * 95) / 100) // 950
	if storage.Food != expectedVillageFood {
		t.Errorf("Expected Village Food to be %d, got %d", expectedVillageFood, storage.Food)
	}

	expectedCaravanFood := uint32((500 * 95) / 100) // 475
	if payload.Food != expectedCaravanFood {
		t.Errorf("Expected Caravan Food to be %d, got %d", expectedCaravanFood, payload.Food)
	}

	// Small value test: At 10, 5% is 0.5 -> 0 in integer math.
	// But system forces at least 1 unit decremented.
	storage.Food = 10
	for i := 0; i < 10; i++ {
		tm.Tick()
	}

	if storage.Food != 9 {
		t.Errorf("Expected Village Food to decrease to 9, got %d", storage.Food)
	}
}
