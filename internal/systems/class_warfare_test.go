package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestClassWarfare_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	sys := NewClassWarfareSystem(&world, hooks)

	// 1. Setup Capital/Village with High Food Hoard and Exorbitant Prices
	cityE := world.NewEntity()
	world.Add(cityE,
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.MarketComponent](&world),
	)

	cityAff := (*components.Affiliation)(world.Get(cityE, ecs.ComponentID[components.Affiliation](&world)))
	cityAff.CityID = 100

	cityStor := (*components.StorageComponent)(world.Get(cityE, ecs.ComponentID[components.StorageComponent](&world)))
	cityStor.Food = 1000 // Ruler is hoarding 1000 food

	cityMarket := (*components.MarketComponent)(world.Get(cityE, ecs.ComponentID[components.MarketComponent](&world)))
	cityMarket.FoodPrice = 50.0 // Exorbitant price

	// 2. Setup the Sovereign Ruler
	rulerE := world.NewEntity()
	world.Add(rulerE,
		ecs.ComponentID[components.AdministrationMarker](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	rulerAff := (*components.Affiliation)(world.Get(rulerE, ecs.ComponentID[components.Affiliation](&world)))
	rulerAff.CityID = 100

	rulerIdent := (*components.Identity)(world.Get(rulerE, ecs.ComponentID[components.Identity](&world)))
	rulerIdent.ID = 1001

	// 3. Setup the Starving Peasant
	peasantE := world.NewEntity()
	world.Add(peasantE,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	peasantAff := (*components.Affiliation)(world.Get(peasantE, ecs.ComponentID[components.Affiliation](&world)))
	peasantAff.CityID = 100

	peasantIdent := (*components.Identity)(world.Get(peasantE, ecs.ComponentID[components.Identity](&world)))
	peasantIdent.ID = 2002

	peasantNeeds := (*components.Needs)(world.Get(peasantE, ecs.ComponentID[components.Needs](&world)))
	peasantNeeds.Food = 10.0  // Starving (< 20)
	peasantNeeds.Wealth = 5.0 // Too poor to buy food (5 < 50)

	// Step 4. Run the update loop to trigger class warfare
	// The system processes every 50 ticks
	for i := 0; i < 50; i++ {
		sys.Update(&world)
	}

	// Verify the structural hook generation
	val := hooks.GetHook(peasantIdent.ID, rulerIdent.ID)

	if val >= 0 {
		t.Fatalf("Expected peasant to develop a negative hook against the ruler, got %d", val)
	}

	if val != -5 {
		t.Fatalf("Expected peasant hook to be -5, got %d", val)
	}
}
