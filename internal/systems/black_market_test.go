package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 53 - Black Market Smuggling Engine

func TestBlackMarketSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewBlackMarketSystem(&world)
	sys.tickCounter = 49 // Trigger Update

	// Component IDs
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	contraID := ecs.ComponentID[components.ContrabandComponent](&world)
	villID := ecs.ComponentID[components.Village](&world)
	markID := ecs.ComponentID[components.MarketComponent](&world)

	// Create Jurisdiction with Contraband (Iron)
	jurEnt := world.NewEntity(jurID, posID, affID, contraID)
	jurPos := (*components.Position)(world.Get(jurEnt, posID))
	jurPos.X = 10.0
	jurPos.Y = 10.0

	jurAff := (*components.Affiliation)(world.Get(jurEnt, affID))
	jurAff.CityID = 1

	jurComp := (*components.JurisdictionComponent)(world.Get(jurEnt, jurID))
	jurComp.RadiusSquared = 100.0 // Radius 10

	contraComp := (*components.ContrabandComponent)(world.Get(jurEnt, contraID))
	contraComp.Contraband = 1 << components.ItemIron // Iron is illegal

	// Create Village with Market inside Jurisdiction
	villEnt := world.NewEntity(villID, posID, markID)
	villPos := (*components.Position)(world.Get(villEnt, posID))
	villPos.X = 12.0
	villPos.Y = 12.0 // DistSq = 8, well within 100

	market := (*components.MarketComponent)(world.Get(villEnt, markID))
	market.IronPrice = 10.0
	market.FoodPrice = 5.0

	// Run system
	sys.Update(&world)

	// Verify iron price is spiked 5x (50.0)
	if market.IronPrice != 50.0 {
		t.Errorf("Expected IronPrice to be spiked to 50.0 due to contraband risk premium, got %f", market.IronPrice)
	}

	// Verify food price remains unchanged (5.0)
	if market.FoodPrice != 5.0 {
		t.Errorf("Expected FoodPrice to remain 5.0 since it's not contraband, got %f", market.FoodPrice)
	}
}
