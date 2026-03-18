package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 31.5: The Winter Heating Engine (Resource Depletion Crisis)
// The "Butterfly Effect" proving Winter Heating ties to Geography (Winter), Economy (Wood Storage), Governance (Loyalty), and Biology (Disease).

func TestWinterHeatingSystem_ButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.PopulationComponent](&world)
	ecs.ComponentID[components.StorageComponent](&world)
	ecs.ComponentID[components.LoyaltyComponent](&world)
	ecs.ComponentID[components.DiseaseEntity](&world)

	// Create a Calendar initialized to Winter
	calendar := engine.NewCalendar()
	calendar.IsWinter = true

	// Create Winter Heating System
	engine.InitializeRNG([32]byte{1, 2, 3})
	heatingSystem := NewWinterHeatingSystem(&world, calendar)

	// 1. Create a heated Village (adequate Wood)
	heatedVillage := world.NewEntity()
	world.Add(heatedVillage,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.PopulationComponent](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.LoyaltyComponent](&world),
	)

	hPos := (*components.Position)(world.Get(heatedVillage, ecs.ComponentID[components.Position](&world)))
	hPos.X, hPos.Y = 10, 10

	hPop := (*components.PopulationComponent)(world.Get(heatedVillage, ecs.ComponentID[components.PopulationComponent](&world)))
	hPop.Count = 100 // 10 wood per tick required

	hStorage := (*components.StorageComponent)(world.Get(heatedVillage, ecs.ComponentID[components.StorageComponent](&world)))
	hStorage.Wood = 100 // Adequate wood for 10 ticks

	hLoyalty := (*components.LoyaltyComponent)(world.Get(heatedVillage, ecs.ComponentID[components.LoyaltyComponent](&world)))
	hLoyalty.Value = 100

	// 2. Create a freezing Village (0 Wood)
	freezingVillage := world.NewEntity()
	world.Add(freezingVillage,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.PopulationComponent](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.LoyaltyComponent](&world),
	)

	fPos := (*components.Position)(world.Get(freezingVillage, ecs.ComponentID[components.Position](&world)))
	fPos.X, fPos.Y = 20, 20

	fPop := (*components.PopulationComponent)(world.Get(freezingVillage, ecs.ComponentID[components.PopulationComponent](&world)))
	fPop.Count = 100 // 10 wood per tick required

	fStorage := (*components.StorageComponent)(world.Get(freezingVillage, ecs.ComponentID[components.StorageComponent](&world)))
	fStorage.Wood = 0 // Immediate Freezing Crisis

	fLoyalty := (*components.LoyaltyComponent)(world.Get(freezingVillage, ecs.ComponentID[components.LoyaltyComponent](&world)))
	fLoyalty.Value = 100

	// Assert base entities setup
	if world.Alive(heatedVillage) == false || world.Alive(freezingVillage) == false {
		t.Fatalf("Failed to initialize villages")
	}

	// Calculate initial entities in the world
	// initialEntityCount := 2 // heatedVillage and freezingVillage

	// 3. Simulate 1 tick
	heatingSystem.Update(&world)

	// Verify the Butterfly Effects

	// Assertion A: Heated Village is warm
	if hStorage.Wood != 90 {
		t.Errorf("Expected Heated Village to have 90 wood, got %d", hStorage.Wood)
	}
	if hLoyalty.Value != 100 {
		t.Errorf("Expected Heated Village to have 100 loyalty, got %d", hLoyalty.Value)
	}

	// Assertion B: Freezing Village collapses
	if fStorage.Wood != 0 {
		t.Errorf("Expected Freezing Village to have 0 wood, got %d", fStorage.Wood)
	}
	if fLoyalty.Value != 95 { // 100 - 5 = 95
		t.Errorf("Expected Freezing Village to drop 5 loyalty, got %d", fLoyalty.Value)
	}

	// 4. Test DiseaseEntity Probability Loop
	// To deterministically guarantee a plague spawns, we artificially loop the Update
	// 500 times. With a 1/100 chance, it will almost certainly spawn at least one.
	for i := 0; i < 500; i++ {
		heatingSystem.Update(&world)
	}

	// Check if any DiseaseEntities were created. The ECS logic should have created at least one.
	diseaseQuery := world.Query(ecs.All(ecs.ComponentID[components.DiseaseEntity](&world)))
	diseaseCount := 0
	for diseaseQuery.Next() {
		diseaseCount++
	}

	if diseaseCount == 0 {
		t.Errorf("Expected at least one DiseaseEntity to spawn over 500 ticks of freezing, got 0.")
	}
}
