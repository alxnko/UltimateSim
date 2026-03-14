package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestHolyWarSystem(t *testing.T) {
	world := ecs.NewWorld()

	holyWarSystem := NewHolyWarSystem(&world)

	// Create City A
	cityA := world.NewEntity()
	world.Add(cityA,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
	)

	identA := (*components.Identity)(world.Get(cityA, ecs.ComponentID[components.Identity](&world)))
	posA := (*components.Position)(world.Get(cityA, ecs.ComponentID[components.Position](&world)))
	storageA := (*components.StorageComponent)(world.Get(cityA, ecs.ComponentID[components.StorageComponent](&world)))
	beliefA := (*components.BeliefComponent)(world.Get(cityA, ecs.ComponentID[components.BeliefComponent](&world)))

	identA.ID = 100
	posA.X = 10.0
	posA.Y = 10.0
	storageA.Food = 500
	storageA.Wood = 500
	beliefA.Beliefs = append(beliefA.Beliefs, components.Belief{BeliefID: 1, Weight: 100})

	// Create City B
	cityB := world.NewEntity()
	world.Add(cityB,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
	)

	identB := (*components.Identity)(world.Get(cityB, ecs.ComponentID[components.Identity](&world)))
	posB := (*components.Position)(world.Get(cityB, ecs.ComponentID[components.Position](&world)))
	storageB := (*components.StorageComponent)(world.Get(cityB, ecs.ComponentID[components.StorageComponent](&world)))
	beliefB := (*components.BeliefComponent)(world.Get(cityB, ecs.ComponentID[components.BeliefComponent](&world)))

	identB.ID = 200
	posB.X = 30.0 // Distance Sq = 400.0 (< 10000.0)
	posB.Y = 10.0
	storageB.Food = 500
	storageB.Wood = 500
	beliefB.Beliefs = append(beliefB.Beliefs, components.Belief{BeliefID: 2, Weight: 100}) // Completely different belief

	// Run system 1000 ticks to trigger spawn
	for i := 0; i < 1000; i++ {
		holyWarSystem.Update(&world)
	}

	// Verify Crusaders spawned
	crusaderFilter := ecs.All(
		ecs.ComponentID[components.CrusaderEntity](&world),
		ecs.ComponentID[components.CrusadeComponent](&world),
	)
	crusaderQuery := world.Query(&crusaderFilter)

	count := 0
	for crusaderQuery.Next() {
		count++
		crusade := (*components.CrusadeComponent)(crusaderQuery.Get(ecs.ComponentID[components.CrusadeComponent](&world)))

		// City A spawns attacking City B and City B spawns attacking City A
		if crusade.TargetCityID != uint32(identA.ID) && crusade.TargetCityID != uint32(identB.ID) {
			t.Errorf("Crusader spawned with unexpected target city ID %d", crusade.TargetCityID)
		}
	}

	if count != 2 { // One from each city
		t.Fatalf("Expected 2 Crusaders to spawn, found %d", count)
	}

	// Move Crusaders to target
	crusaderFilter2 := ecs.All(
		ecs.ComponentID[components.CrusaderEntity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.CrusadeComponent](&world),
	)

	crusaderQuery2 := world.Query(&crusaderFilter2)
	for crusaderQuery2.Next() {
		pos := (*components.Position)(crusaderQuery2.Get(ecs.ComponentID[components.Position](&world)))
		crusade := (*components.CrusadeComponent)(crusaderQuery2.Get(ecs.ComponentID[components.CrusadeComponent](&world)))

		if crusade.TargetCityID == uint32(identB.ID) {
			pos.X = 30.0
			pos.Y = 10.0
		} else {
			pos.X = 10.0
			pos.Y = 10.0
		}
	}

	// Update system to trigger combat
	holyWarSystem.Update(&world)

	// Crusaders should despawn, storages should be damaged
	if storageA.Food != 450 || storageA.Wood != 450 {
		t.Errorf("City A storage should have taken damage. Food: %d, Wood: %d", storageA.Food, storageA.Wood)
	}

	if storageB.Food != 450 || storageB.Wood != 450 {
		t.Errorf("City B storage should have taken damage. Food: %d, Wood: %d", storageB.Food, storageB.Wood)
	}

	// Verify Crusaders despawned
	crusaderQuery3 := world.Query(&crusaderFilter)
	countAfter := 0
	for crusaderQuery3.Next() {
		countAfter++
	}

	if countAfter != 0 {
		t.Errorf("Expected Crusaders to despawn after attacking, but found %d", countAfter)
	}
}
