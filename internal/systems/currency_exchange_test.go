package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestCurrencyExchangeSystem verifies global rate updates map accurately to CoinEntities
func TestCurrencyExchangeSystem_DeterministicRates(t *testing.T) {
	world := ecs.NewWorld()

	sys := NewCurrencyExchangeSystem(&world)

	// Component IDs
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	coinTagID := ecs.ComponentID[components.CoinEntity](&world)
	currencyID := ecs.ComponentID[components.CurrencyComponent](&world)

	// Create Capital A (High Prestige, High Resources)
	capA := world.NewEntity(capitalID, affilID, storageID, legacyID)
	affilA := (*components.Affiliation)(world.Get(capA, affilID))
	affilA.CityID = 1

	storageA := (*components.StorageComponent)(world.Get(capA, storageID))
	storageA.Iron = 1000
	storageA.Food = 1000

	legacyA := (*components.Legacy)(world.Get(capA, legacyID))
	legacyA.Prestige = 50 // Prestige modifier = 0.5, Resources modifier = 2.0. Expected Rate: 1.0 + 0.5 + 2.0 = 3.5

	// Create Capital B (Low Prestige, Low Resources)
	capB := world.NewEntity(capitalID, affilID, storageID, legacyID)
	affilB := (*components.Affiliation)(world.Get(capB, affilID))
	affilB.CityID = 2

	storageB := (*components.StorageComponent)(world.Get(capB, storageID))
	storageB.Wood = 500

	legacyB := (*components.Legacy)(world.Get(capB, legacyID))
	legacyB.Prestige = 10 // Expected Rate: 1.0 + 0.1 + 0.5 = 1.6

	// Create Physical Coins mapped to Issuer 1 and Issuer 2
	coin1 := world.NewEntity(coinTagID, currencyID)
	curr1 := (*components.CurrencyComponent)(world.Get(coin1, currencyID))
	curr1.IssuerID = 1
	curr1.Value = 1.0

	coin2 := world.NewEntity(coinTagID, currencyID)
	curr2 := (*components.CurrencyComponent)(world.Get(coin2, currencyID))
	curr2.IssuerID = 2
	curr2.Value = 1.0

	// Create Orphan Coin (Issuer 3 doesn't exist)
	coin3 := world.NewEntity(coinTagID, currencyID)
	curr3 := (*components.CurrencyComponent)(world.Get(coin3, currencyID))
	curr3.IssuerID = 3
	curr3.Value = 1.0

	// Tick 99 - no changes
	for i := 0; i < 99; i++ {
		sys.Update()
	}

	if curr1.Value != 1.0 || curr2.Value != 1.0 || curr3.Value != 1.0 {
		t.Fatalf("Expected coin values to remain 1.0 before tick 100")
	}

	// Tick 100 - exchange rates recalculate
	sys.Update()

	if curr1.Value != 3.5 {
		t.Fatalf("Expected coin1 value to be 3.5, got %v", curr1.Value)
	}
	if curr2.Value != 1.6 {
		t.Fatalf("Expected coin2 value to be 1.6, got %v", curr2.Value)
	}
	if curr3.Value != 0.01 {
		t.Fatalf("Expected orphan coin3 value to crater to 0.01, got %v", curr3.Value)
	}

	// Verify marketRates map was updated correctly to maintain O(1) DOD hits
	if rate, exists := sys.marketRates[1]; !exists || rate != 3.5 {
		t.Fatalf("marketRates hashmap failed for Issuer 1")
	}
}
