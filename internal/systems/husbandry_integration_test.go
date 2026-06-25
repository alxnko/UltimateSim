package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 31 - Flora, Fauna, // Phase 31: Flora, Fauna, & Animal Husbandry Animal Husbandry E2E Test
func TestHusbandrySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewHusbandrySystem(&world, nil)

	posID := ecs.ComponentID[components.Position](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	animalID := ecs.ComponentID[components.AnimalComponent](&world)
	tamedID := ecs.ComponentID[components.TamedMarker](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)

	// 1. Create a Village with a Market and Storage
	cityEnt := world.NewEntity(posID, villageID, affID, marketID, storageID)
	cAff := (*components.Affiliation)(world.Get(cityEnt, affID))
	cAff.CityID = 1

	cMarket := (*components.MarketComponent)(world.Get(cityEnt, marketID))
	cMarket.FoodPrice = 5.0 // Initial low price (No Famine)

	cStore := (*components.StorageComponent)(world.Get(cityEnt, storageID))
	cStore.Food = 100 // Base food

	// 2. Create a Herder (JobHerder) in the city
	herderEnt := world.NewEntity(posID, jobID, needsID, identID, affID)
	hPos := (*components.Position)(world.Get(herderEnt, posID))
	hPos.X, hPos.Y = 10, 10

	hJob := (*components.JobComponent)(world.Get(herderEnt, jobID))
	hJob.JobID = components.JobHerder

	hIdent := (*components.Identity)(world.Get(herderEnt, identID))
	hIdent.ID = 500

	hAff := (*components.Affiliation)(world.Get(herderEnt, affID))
	hAff.CityID = 1

	// 3. Create a wild Animal nearby
	animEnt := world.NewEntity(posID, animalID)
	aPos := (*components.Position)(world.Get(animEnt, posID))
	aPos.X, aPos.Y = 11, 10 // Adjacent (distSq = 1)

	anim := (*components.AnimalComponent)(world.Get(animEnt, animalID))
	anim.YieldMeat = 50

	// ----- Test Taming -----
	sys.Update(&world)

	if !world.Alive(animEnt) {
		t.Fatalf("Animal should not be slaughtered during non-famine")
	}

	if !world.Has(animEnt, tamedID) {
		t.Fatalf("Animal should be tamed by Herder")
	}

	tamed := (*components.TamedMarker)(world.Get(animEnt, tamedID))
	if tamed.OwnerID != 500 {
		t.Fatalf("Animal should be owned by Herder 500, got %d", tamed.OwnerID)
	}

	if cStore.Food != 100 {
		t.Fatalf("Food should not have increased, got %d", cStore.Food)
	}

	// ----- Test Slaughtering (Famine) -----
	cMarket.FoodPrice = 15.0 // Trigger famine

	sys.Update(&world)

	if world.Alive(animEnt) {
		t.Fatalf("Animal should be slaughtered during famine")
	}

	if cStore.Food != 150 { // 100 + 50 yield
		t.Fatalf("Village Storage Food should have increased to 150, got %d", cStore.Food)
	}
}
