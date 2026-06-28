package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 31: Animal Husbandry Engine (Testing & Validation)
func TestHusbandrySystem(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewHusbandrySystem(&world)

	// Create Employer (Village)
	emp := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.MarketComponent](&world),
		ecs.ComponentID[components.StorageComponent](&world),
	)

	empIdent := (*components.Identity)(world.Get(emp, ecs.ComponentID[components.Identity](&world)))
	empIdent.ID = 100

	empMarket := (*components.MarketComponent)(world.Get(emp, ecs.ComponentID[components.MarketComponent](&world)))
	empMarket.FoodPrice = 5.0 // No famine initially

	empStorage := (*components.StorageComponent)(world.Get(emp, ecs.ComponentID[components.StorageComponent](&world)))
	empStorage.Food = 10

	// Create Herder NPC
	world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.JobComponent](&world),
	)

	// I need to use get and set here
	herderEnt := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.JobComponent](&world),
	)

	herderJob := (*components.JobComponent)(world.Get(herderEnt, ecs.ComponentID[components.JobComponent](&world)))
	herderJob.JobID = components.JobHerder
	herderJob.EmployerID = 100

	// Create Wild Animal
	animalEnt := world.NewEntity(
		ecs.ComponentID[components.AnimalComponent](&world),
	)
	animalComp := (*components.AnimalComponent)(world.Get(animalEnt, ecs.ComponentID[components.AnimalComponent](&world)))
	animalComp.YieldMeat = 50

	// Tick 1: Herder tames the animal
	sys.Update(&world)

	if !world.Has(animalEnt, ecs.ComponentID[components.TamedMarker](&world)) {
		t.Fatalf("Expected animal to be tamed, but it lacks TamedMarker")
	}

	tamedMarker := (*components.TamedMarker)(world.Get(animalEnt, ecs.ComponentID[components.TamedMarker](&world)))
	if tamedMarker.OwnerID != 100 {
		t.Fatalf("Expected tamed animal owner to be 100, got %d", tamedMarker.OwnerID)
	}

	// Tick 2: Normal conditions, animal should not be slaughtered
	sys.Update(&world)
	if !world.Alive(animalEnt) {
		t.Fatalf("Expected animal to remain alive when there is no famine")
	}
	if empStorage.Food != 10 {
		t.Fatalf("Expected food to remain 10, got %d", empStorage.Food)
	}

	// Trigger famine
	empMarket.FoodPrice = 15.0 // > 10.0 threshold

	// Tick 3: Famine conditions, herder slaughters the animal
	sys.Update(&world)

	if world.Alive(animalEnt) {
		t.Fatalf("Expected animal to be slaughtered and entity removed")
	}

	if empStorage.Food != 60 { // 10 + 50 yield
		t.Fatalf("Expected employer food storage to increase by 50 to 60, got %d", empStorage.Food)
	}
}
