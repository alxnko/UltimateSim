package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine Integration Test
// Demonstrates the "Butterfly Effect": Physical proximity of enemy NPCs to a Village
// triggers a SiegeMarker, decimates the local economy (Storage, Market), and
// forces political capitulation (Affiliation, Loyalty).

func TestSiegeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewSiegeSystem(&world)

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	siegeID := ecs.ComponentID[components.SiegeMarker](&world)

	// 1. Create a wealthy, loyal Village for Country 1
	village := world.NewEntity(villageID, posID, affID, storageID, marketID, loyaltyID)

	vPos := (*components.Position)(world.Get(village, posID))
	vPos.X = 100.0
	vPos.Y = 100.0

	vAff := (*components.Affiliation)(world.Get(village, affID))
	vAff.CountryID = 1

	vStorage := (*components.StorageComponent)(world.Get(village, storageID))
	vStorage.Food = 100

	vMarket := (*components.MarketComponent)(world.Get(village, marketID))
	vMarket.FoodPrice = 5.0

	vLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	vLoyalty.Value = 30

	// 2. Spawn one friendly NPC (Country 1) inside the village
	friendlyNPC := world.NewEntity(npcID, posID, affID)
	fPos := (*components.Position)(world.Get(friendlyNPC, posID))
	fPos.X = 101.0
	fPos.Y = 101.0
	fAff := (*components.Affiliation)(world.Get(friendlyNPC, affID))
	fAff.CountryID = 1

	// 3. Spawn three hostile NPCs (Country 2) surrounding the village (distSq < 25)
	for i := 0; i < 3; i++ {
		enemy := world.NewEntity(npcID, posID, affID)
		ePos := (*components.Position)(world.Get(enemy, posID))
		ePos.X = 100.0 + float32(i)
		ePos.Y = 98.0
		eAff := (*components.Affiliation)(world.Get(enemy, affID))
		eAff.CountryID = 2
	}

	// Ensure no siege initially
	if world.Has(village, siegeID) {
		t.Fatalf("Village should not start besieged")
	}

	// 4. Tick to trigger the siege initiation (tick 30)
	for i := 0; i < 30; i++ {
		sys.Update(&world)
	}

	// Assert: SiegeMarker added
	if !world.Has(village, siegeID) {
		t.Fatalf("Village did not receive SiegeMarker despite being outnumbered")
	}

	siege := (*components.SiegeMarker)(world.Get(village, siegeID))
	if siege.BesiegerCountryID != 2 {
		t.Fatalf("Expected BesiegerCountryID to be 2, got %d", siege.BesiegerCountryID)
	}

	// 5. Tick to apply siege effects (tick 60)
	for i := 0; i < 30; i++ {
		sys.Update(&world)
	}

	// Assert: Economic and Political Damage
	vStorage = (*components.StorageComponent)(world.Get(village, storageID))
	vMarket = (*components.MarketComponent)(world.Get(village, marketID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))

	if vStorage.Food != 90 {
		t.Fatalf("Expected Food to deplete to 90, got %d", vStorage.Food)
	}
	if vMarket.FoodPrice != 25.0 {
		t.Fatalf("Expected FoodPrice to spike to 25.0, got %f", vMarket.FoodPrice)
	}
	if vLoyalty.Value != 20 {
		t.Fatalf("Expected Loyalty to drain to 20, got %d", vLoyalty.Value)
	}

	// 6. Tick again to force Capitulation (tick 90 and 120)
	for i := 0; i < 60; i++ {
		sys.Update(&world)
	}

	// Assert: Capitulation and Siege Lifted
	vAff = (*components.Affiliation)(world.Get(village, affID))
	vLoyalty = (*components.LoyaltyComponent)(world.Get(village, loyaltyID))

	if vLoyalty.Value != 0 {
		t.Fatalf("Expected Loyalty to be 0 upon capitulation, got %d", vLoyalty.Value)
	}
	if vAff.CountryID != 2 {
		t.Fatalf("Village did not capitulate. Expected CountryID 2, got %d", vAff.CountryID)
	}
	if world.Has(village, siegeID) {
		t.Fatalf("SiegeMarker was not removed after capitulation")
	}
}
