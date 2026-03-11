package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.1: Settlement Conversion System Test
// End-to-End deterministic test verifying the transformation of FamilyCluster into Village.

func TestSettlementRuleSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Setup a high-resource tile at (5, 5)
	mapGrid.Resources[5*10+5] = engine.ResourceDepot{WoodValue: 30, FoodValue: 30} // Sum > 50

	system := systems.NewSettlementRuleSystem(mapGrid)

	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	fcID := ecs.ComponentID[components.FamilyCluster](&world)
	slID := ecs.ComponentID[components.SettlementLogic](&world)

	// Spawn migrating cluster at (5, 5) with zero velocity
	entity := world.NewEntity(posID, velID, fcID, slID)

	pos := (*components.Position)(world.Get(entity, posID))
	pos.X = 5
	pos.Y = 5

	vel := (*components.Velocity)(world.Get(entity, velID))
	vel.X = 0
	vel.Y = 0

	sl := (*components.SettlementLogic)(world.Get(entity, slID))
	sl.TicksAtZeroVelocity = 0

	// Tick 999 times - entity should not despawn yet
	for i := 0; i < 999; i++ {
		system.Update(&world)
	}

	if !world.Alive(entity) {
		t.Fatalf("Entity despawned too early at tick %d", sl.TicksAtZeroVelocity)
	}

	// Tick 1000th time - conversion should occur
	system.Update(&world)

	if world.Alive(entity) {
		t.Fatalf("FamilyCluster entity should have been despawned")
	}

	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)

	query := world.Query(ecs.All(villageID, posID, storageID, popID))
	count := 0

	for query.Next() {
		count++

		newPos := (*components.Position)(query.Get(posID))
		if newPos.X != 5 || newPos.Y != 5 {
			t.Errorf("Village spawned at incorrect location: (%f, %f)", newPos.X, newPos.Y)
		}

		storage := (*components.StorageComponent)(query.Get(storageID))
		if storage.Wood != 100 || storage.Food != 100 {
			t.Errorf("Village storage not initialized correctly")
		}

		pop := (*components.PopulationComponent)(query.Get(popID))
		if pop.Count != 10 {
			t.Errorf("Village population not initialized correctly")
		}
	}

	if count != 1 {
		t.Fatalf("Expected exactly 1 Village entity spawned, got %d", count)
	}
}

// Deterministic Check
func TestSettlementRuleSystem_Deterministic(t *testing.T) {
	runSim := func() float32 {
		engine.InitializeRNG([32]byte{42})
		world := ecs.NewWorld()
		mapGrid := engine.NewMapGrid(10, 10)
		mapGrid.Resources[5*10+5] = engine.ResourceDepot{WoodValue: 30, FoodValue: 30}

		system := systems.NewSettlementRuleSystem(mapGrid)

		posID := ecs.ComponentID[components.Position](&world)
		velID := ecs.ComponentID[components.Velocity](&world)
		fcID := ecs.ComponentID[components.FamilyCluster](&world)
		slID := ecs.ComponentID[components.SettlementLogic](&world)
		idID := ecs.ComponentID[components.Identity](&world)

		entity := world.NewEntity(posID, velID, fcID, slID, idID)

		pos := (*components.Position)(world.Get(entity, posID))
		pos.X = 5
		pos.Y = 5

		id := (*components.Identity)(world.Get(entity, idID))
		id.BaseTraits = uint32(engine.GetRandomInt()) // Random data based on global seed

		for i := 0; i < 1000; i++ {
			system.Update(&world)
		}

		villageID := ecs.ComponentID[components.Village](&world)
		query := world.Query(ecs.All(villageID, idID))
		var sum float32 = 0
		for query.Next() {
			inheritedID := (*components.Identity)(query.Get(idID))
			sum += float32(inheritedID.BaseTraits)
		}

		return sum
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 = %f, Run 2 = %f", result1, result2)
	}
}
