package systems

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewSiegeSystem(&world)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)
	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// Create Attacker Capital (at war with Defender)
	attackerCap := world.NewEntity(posID, affID, capID, warID)
	world.Get(attackerCap, posID) // initialize
	aAff := (*components.Affiliation)(world.Get(attackerCap, affID))
	aAff.CountryID = 1

	aWar := (*components.WarTrackerComponent)(world.Get(attackerCap, warID))
	aWar.Active = true
	aWar.TargetCountryID = 2 // At war with country 2

	// Create Defender Village
	defVillage := world.NewEntity(posID, affID, marketID, loyaltyID, villageID)
	vPos := (*components.Position)(world.Get(defVillage, posID))
	vPos.X, vPos.Y = 10.0, 10.0

	vAff := (*components.Affiliation)(world.Get(defVillage, affID))
	vAff.CountryID = 2 // The target

	vMarket := (*components.MarketComponent)(world.Get(defVillage, marketID))
	vMarket.FoodPrice = 10.0

	vLoyalty := (*components.LoyaltyComponent)(world.Get(defVillage, loyaltyID))
	vLoyalty.Value = 100

	// Create 1 Defender NPC
	defNPC := world.NewEntity(npcID, posID, affID)
	dnPos := (*components.Position)(world.Get(defNPC, posID))
	dnPos.X, dnPos.Y = 10.0, 10.0 // Exactly at village
	dnAff := (*components.Affiliation)(world.Get(defNPC, affID))
	dnAff.CountryID = 2

	// Create 2 Attacker NPCs near the village (distSq < 25)
	attNPC1 := world.NewEntity(npcID, posID, affID)
	an1Pos := (*components.Position)(world.Get(attNPC1, posID))
	an1Pos.X, an1Pos.Y = 12.0, 10.0 // distSq = 4
	an1Aff := (*components.Affiliation)(world.Get(attNPC1, affID))
	an1Aff.CountryID = 1

	attNPC2 := world.NewEntity(npcID, posID, affID)
	an2Pos := (*components.Position)(world.Get(attNPC2, posID))
	an2Pos.X, an2Pos.Y = 10.0, 12.0 // distSq = 4
	an2Aff := (*components.Affiliation)(world.Get(attNPC2, affID))
	an2Aff.CountryID = 1

	// Run system (100 ticks to trigger)
	for i := 0; i < 100; i++ {
		sys.Update(&world)
	}

	// Verify SiegeMarker was applied
	if !world.Has(defVillage, siegeID) {
		t.Fatalf("Expected village to have SiegeMarker applied")
	}

	marker := (*components.SiegeMarker)(world.Get(defVillage, siegeID))
	if marker.BesiegerCountryID != 1 {
		t.Errorf("Expected BesiegerCountryID to be 1, got %d", marker.BesiegerCountryID)
	}

	vMarket = (*components.MarketComponent)(world.Get(defVillage, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(defVillage, loyaltyID))

	// Verify economic impact (Initial +10 spike)
	if vMarket.FoodPrice != 20.0 {
		t.Errorf("Expected FoodPrice to spike to 20.0, got %f", vMarket.FoodPrice)
	}

	if vLoyalty.Value != 95 {
		t.Errorf("Expected Loyalty to drop to 95, got %d", vLoyalty.Value)
	}

	// Run system for another 100 ticks (Ongoing siege)
	for i := 0; i < 100; i++ {
		sys.Update(&world)
	}

	vMarket = (*components.MarketComponent)(world.Get(defVillage, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(defVillage, loyaltyID))

	// Verify ongoing economic impact (+5 per tick)
	if vMarket.FoodPrice != 25.0 {
		t.Errorf("Expected FoodPrice to increase to 25.0, got %f", vMarket.FoodPrice)
	}

	if vLoyalty.Value != 94 {
		t.Errorf("Expected Loyalty to drop to 94, got %d", vLoyalty.Value)
	}

	// Move attackers away to end siege
	an1Pos.X, an1Pos.Y = 100.0, 100.0 // far away
	an2Pos.X, an2Pos.Y = 100.0, 100.0 // far away

	// Run system for another 100 ticks
	for i := 0; i < 100; i++ {
		sys.Update(&world)
	}

	// Verify SiegeMarker was removed
	if world.Has(defVillage, siegeID) {
		t.Fatalf("Expected village to have SiegeMarker removed after attackers left")
	}
}
