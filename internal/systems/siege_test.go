package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	capID := ecs.ComponentID[components.CapitalComponent](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	vilID := ecs.ComponentID[components.Village](&world)
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	loyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	marID := ecs.ComponentID[components.MarketComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)

	// Create a Capital representing the attacking country (Country 1) declaring war on Country 2
	attackerCapital := world.NewEntity(capID, warID, affID)
	war := (*components.WarTrackerComponent)(world.Get(attackerCapital, warID))
	war.Active = true
	war.TargetCountryID = 2

	attackerAff := (*components.Affiliation)(world.Get(attackerCapital, affID))
	attackerAff.CountryID = 1

	// Create a target Village belonging to Country 2
	targetVillage := world.NewEntity(vilID, posID, affID, loyID, marID)
	vPos := (*components.Position)(world.Get(targetVillage, posID))
	vPos.X = 10.0
	vPos.Y = 10.0

	vAff := (*components.Affiliation)(world.Get(targetVillage, affID))
	vAff.CountryID = 2

	vLoy := (*components.LoyaltyComponent)(world.Get(targetVillage, loyID))
	vLoy.Value = 25 // Set low enough to force a surrender quickly

	vMar := (*components.MarketComponent)(world.Get(targetVillage, marID))
	vMar.FoodPrice = 10.0

	// Create two attacking NPCs (Country 1) near the village (distSq <= 25.0)
	for i := 0; i < 2; i++ {
		npc := world.NewEntity(npcID, posID, affID)
		nPos := (*components.Position)(world.Get(npc, posID))
		nPos.X = 12.0 // Distance from 10,10 is 2,2 -> distSq = 8.0
		nPos.Y = 12.0

		nAff := (*components.Affiliation)(world.Get(npc, affID))
		nAff.CountryID = 1
	}

	// Create one defending NPC (Country 2) near the village
	defender := world.NewEntity(npcID, posID, affID)
	dPos := (*components.Position)(world.Get(defender, posID))
	dPos.X = 10.0
	dPos.Y = 10.0
	dAff := (*components.Affiliation)(world.Get(defender, affID))
	dAff.CountryID = 2

	system := NewSiegeSystem(&world)

	// Tick 1-9: Nothing happens due to staggered execution
	for i := 0; i < 9; i++ {
		system.Update(&world)
	}

	// Tick 10: Siege applied (Attackers 2 > Defenders 1)
	system.Update(&world)

	if !world.Has(targetVillage, siegeID) {
		t.Fatalf("Expected village to have SiegeMarker applied")
	}

	siegeMarker := (*components.SiegeMarker)(world.Get(targetVillage, siegeID))
	if siegeMarker.BesiegerCountryID != 1 {
		t.Errorf("Expected BesiegerCountryID to be 1, got %d", siegeMarker.BesiegerCountryID)
	}

	// Tick 11-19: Nothing
	for i := 0; i < 9; i++ {
		system.Update(&world)
	}

	// Tick 20: Siege effects applied (FoodPrice spikes, Loyalty drops)
	system.Update(&world)

	vMar = (*components.MarketComponent)(world.Get(targetVillage, marID))
	if vMar.FoodPrice <= 10.0 {
		t.Errorf("Expected FoodPrice to spike, got %f", vMar.FoodPrice)
	}

	vLoy = (*components.LoyaltyComponent)(world.Get(targetVillage, loyID))
	if vLoy.Value >= 25 {
		t.Errorf("Expected Loyalty to drop, got %d", vLoy.Value)
	}

	// Run enough ticks to force Loyalty to 0 and trigger surrender
	for i := 0; i < 30; i++ {
		system.Update(&world)
	}

	// Verify surrender
	vAff = (*components.Affiliation)(world.Get(targetVillage, affID))
	if vAff.CountryID != 1 {
		t.Errorf("Expected village to surrender to Country 1, got %d", vAff.CountryID)
	}

	// Verify siege marker is removed upon surrender
	if world.Has(targetVillage, siegeID) {
		t.Errorf("Expected SiegeMarker to be removed after surrender")
	}
}
