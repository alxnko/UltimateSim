package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 05.1 & 14: Settlement Conversion System Test
// End-to-End deterministic test verifying the transformation of NPC into Village logic.

func TestSettlementRuleSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Setup a high-resource tile at (5, 5)
	mapGrid.Resources[5*10+5] = engine.ResourceDepot{WoodValue: 30, FoodValue: 30} // Sum > 50

	system := systems.NewSettlementRuleSystem(mapGrid)

	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	slID := ecs.ComponentID[components.SettlementLogic](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)

	// Spawn migrating NPC at (5, 5) with zero velocity
	entity := world.NewEntity(posID, velID, npcID, slID, idID, affID)

	pos := (*components.Position)(world.Get(entity, posID))
	pos.X = 5
	pos.Y = 5

	vel := (*components.Velocity)(world.Get(entity, velID))
	vel.X = 0
	vel.Y = 0

	sl := (*components.SettlementLogic)(world.Get(entity, slID))
	id := (*components.Identity)(world.Get(entity, idID))
	id.ID = 123

	aff := (*components.Affiliation)(world.Get(entity, affID))
	aff.CityID = 0

	sl.TicksAtZeroVelocity = 0

	// Tick 999 times - conversion should not occur yet
	for i := 0; i < 999; i++ {
		system.Update(&world)
	}

	if !world.Alive(entity) {
		t.Fatalf("Entity despawned too early at tick %d", sl.TicksAtZeroVelocity)
	}

	// Tick 1000th time - conversion should occur
	system.Update(&world)

	if !world.Alive(entity) {
		t.Fatalf("NPC entity should NOT have been despawned")
	}

	// Verify Affiliation is updated
	aff = (*components.Affiliation)(world.Get(entity, affID))
	if aff.CityID != uint32(123) {
		t.Fatalf("NPC CityID was not updated to new Village ID. Expected 123, got %d", aff.CityID)
	}

	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)

	query := world.Query(ecs.All(villageID, posID, storageID, popID, marketID))
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
		if pop.Count != 1 {
			t.Errorf("Village population not initialized correctly, got %d", pop.Count)
		}

		market := (*components.MarketComponent)(query.Get(marketID))
		if market.FoodPrice != 1.0 {
			t.Errorf("Village market not initialized correctly")
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
		npcID := ecs.ComponentID[components.NPC](&world)
		slID := ecs.ComponentID[components.SettlementLogic](&world)
		idID := ecs.ComponentID[components.Identity](&world)

		entity := world.NewEntity(posID, velID, npcID, slID, idID)

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
