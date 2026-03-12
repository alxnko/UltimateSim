package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.1: The Caravan Entity
// Deterministic End-to-End Test for CaravanSpawnerSystem
func TestCaravanSpawnerSystem(t *testing.T) {
	world := ecs.NewWorld()

	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)

	// Create a Village entity
	villageEntity := world.NewEntity(villageID, storageID, marketID, posID)

	// Initialize components
	storage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	storage.Wood = 100

	market := (*components.MarketComponent)(world.Get(villageEntity, marketID))
	market.FoodPrice = 15.0 // Phase 13.1: > 10.0 triggers Caravan

	pos := (*components.Position)(world.Get(villageEntity, posID))
	pos.X = 1.0
	pos.Y = 1.0

	// Instantiate the system
	sys := NewCaravanSpawnerSystem()

	// Verify pre-conditions
	caravanID := ecs.ComponentID[components.Caravan](&world)
	filterCaravan := ecs.All(caravanID)
	q1 := world.Query(filterCaravan)
	if q1.Count() != 0 {
		t.Errorf("Expected 0 Caravans before update, got %d", q1.Count())
	}
	q1.Close()

	// Run system for 1 tick
	sys.Update(&world)

	// Verify a Caravan was spawned
	q2 := world.Query(filterCaravan)
	if q2.Count() != 1 {
		t.Fatalf("Expected exactly 1 Caravan after update, got %d", q2.Count())
	}

	for q2.Next() {
		entity := q2.Entity()

		// Verify Position
		cPos := (*components.Position)(world.Get(entity, posID))
		if cPos.X != 1.0 || cPos.Y != 1.0 {
			t.Errorf("Expected Caravan Position (1,1), got (%f,%f)", cPos.X, cPos.Y)
		}

		// Verify Payload
		payloadID := ecs.ComponentID[components.Payload](&world)
		payload := (*components.Payload)(world.Get(entity, payloadID))
		if payload.Wood != 50 {
			t.Errorf("Expected Caravan to have 50 Wood in Payload, got %d", payload.Wood)
		}
	}

	// Verify Village storage was decremented
	if storage.Wood != 50 {
		t.Errorf("Expected Village to have 50 Wood remaining in Storage, got %d", storage.Wood)
	}

	// Test Condition not met (Price is stable)
	market.FoodPrice = 5.0 // Phase 13.1: 5.0 <= 10.0 (no caravan)
	sys.Update(&world)

	q3 := world.Query(filterCaravan)
	if q3.Count() != 1 {
		t.Errorf("Expected exactly 1 Caravan after second update (no new spawns), got %d", q3.Count())
	}
}
