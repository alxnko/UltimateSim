package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// TestSiegeSystem_Integration verifies that the Phase 66 Physical Siege Engine
// correctly detects numerical superiority during a war, applies a SiegeMarker,
// and effectively drains the defending village's food and loyalty.
func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	villID := ecs.ComponentID[components.Village](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)

	// Attacker Country: 1, Defender Country: 2
	var attackerCountryID uint32 = 1
	var defenderCountryID uint32 = 2

	// Create attacker capital and initiate war against defender
	attackerCapital := world.NewEntity(capID, affID, warID)
	attAff := (*components.Affiliation)(world.Get(attackerCapital, affID))
	attAff.CountryID = attackerCountryID

	war := (*components.WarTrackerComponent)(world.Get(attackerCapital, warID))
	war.Active = true
	war.TargetCountryID = defenderCountryID

	// Create defender village
	defenderVillage := world.NewEntity(villID, posID, affID, loyaltyID, marketID, storageID)
	defPos := (*components.Position)(world.Get(defenderVillage, posID))
	defPos.X = 10.0
	defPos.Y = 10.0

	defAff := (*components.Affiliation)(world.Get(defenderVillage, affID))
	defAff.CountryID = defenderCountryID

	defStorage := (*components.StorageComponent)(world.Get(defenderVillage, storageID))
	defStorage.Food = 20

	defMarket := (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	defMarket.FoodPrice = 5.0

	defLoyalty := (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))
	defLoyalty.Value = 100

	// Create 3 Attacker NPCs surrounding the village (distance < 5.0)
	for i := 0; i < 3; i++ {
		npc := world.NewEntity(npcID, posID, affID)
		npcPos := (*components.Position)(world.Get(npc, posID))
		npcPos.X = 11.0
		npcPos.Y = 11.0
		npcAff := (*components.Affiliation)(world.Get(npc, affID))
		npcAff.CountryID = attackerCountryID
	}

	// Create 1 Defender NPC near the village
	npcDef := world.NewEntity(npcID, posID, affID)
	npcDefPos := (*components.Position)(world.Get(npcDef, posID))
	npcDefPos.X = 10.5
	npcDefPos.Y = 10.5
	npcDefAff := (*components.Affiliation)(world.Get(npcDef, affID))
	npcDefAff.CountryID = defenderCountryID

	siegeSystem := systems.NewSiegeSystem(&world)

	// Tick 1-9: No update due to throttle
	for i := 0; i < 9; i++ {
		siegeSystem.Update(&world)
	}

	// Tick 10: Siege system evaluates
	siegeSystem.Update(&world)

	// Verify siege was applied
	if !world.Has(defenderVillage, siegeID) {
		t.Fatalf("Expected defender village to have SiegeMarker applied")
	}

	marker := (*components.SiegeMarker)(world.Get(defenderVillage, siegeID))
	if marker.BesiegerCountryID != attackerCountryID {
		t.Errorf("Expected BesiegerCountryID %d, got %d", attackerCountryID, marker.BesiegerCountryID)
	}

	// Fetch components again (structs were updated)
	defMarket = (*components.MarketComponent)(world.Get(defenderVillage, marketID))
	defStorage = (*components.StorageComponent)(world.Get(defenderVillage, storageID))
	defLoyalty = (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))

	if defMarket.FoodPrice <= 5.0 {
		t.Errorf("Expected FoodPrice to spike, got %f", defMarket.FoodPrice)
	}

	if defStorage.Food >= 20 {
		t.Errorf("Expected Food to deplete, got %d", defStorage.Food)
	}

	if defLoyalty.Value < 100 {
		t.Errorf("Loyalty should not deplete until food is 0, got %d", defLoyalty.Value)
	}

	// Tick until food runs out and loyalty breaks
	for i := 0; i < 50; i++ {
		siegeSystem.Update(&world)
	}

	defStorage = (*components.StorageComponent)(world.Get(defenderVillage, storageID))
	defLoyalty = (*components.LoyaltyComponent)(world.Get(defenderVillage, loyaltyID))

	if defStorage.Food != 0 {
		t.Errorf("Expected Food to be completely drained to 0, got %d", defStorage.Food)
	}

	if defLoyalty.Value >= 100 {
		t.Errorf("Expected Loyalty to drain due to starvation, got %d", defLoyalty.Value)
	}
}
