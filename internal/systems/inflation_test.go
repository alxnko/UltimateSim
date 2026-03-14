package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// TestInflationSystem_ButterflyEffect verifies that a King's Debasement lowers Iron cost, physically spawns a Debased coin, and triggers organic hyperinflation.
func TestInflationSystem_ButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize Minting and Inflation Systems
	mintSys := NewMintingSystem(&world)
	infSys := NewInflationSystem(&world)

	// Retrieve Component IDs
	villageID := ecs.ComponentID[components.Village](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	coinTagID := ecs.ComponentID[components.CoinEntity](&world)
	currencyID := ecs.ComponentID[components.CurrencyComponent](&world)

	// 1. Create a Capital that is Debasing its coinage heavily (50% Debasement)
	capitalEnt := world.NewEntity(villageID, capitalID, countryID, storageID, affilID, posID, marketID)

	capPos := (*components.Position)(world.Get(capitalEnt, posID))
	capPos.X = 10.0
	capPos.Y = 20.0

	capAffil := (*components.Affiliation)(world.Get(capitalEnt, affilID))
	capAffil.CityID = 1

	capCountry := (*components.CountryComponent)(world.Get(capitalEnt, countryID))
	capCountry.Debasement = 0.5 // 50% debasement

	capStorage := (*components.StorageComponent)(world.Get(capitalEnt, storageID))
	capStorage.Iron = 150 // Should cost 50 Iron (100 * (1.0 - 0.5)) to mint

	capMarket := (*components.MarketComponent)(world.Get(capitalEnt, marketID))
	capMarket.FoodPrice = 10.0
	capMarket.WoodPrice = 10.0
	capMarket.StonePrice = 10.0
	capMarket.IronPrice = 10.0

	// Tick the Minting system up to 100 to trigger coin generation
	for i := 0; i < 99; i++ {
		mintSys.Update()
	}

	// Verify iron wasn't touched yet
	if capStorage.Iron != 150 {
		t.Fatalf("Expected Iron to remain 150 before minting, got %v", capStorage.Iron)
	}

	// Tick 100: Minting occurs
	mintSys.Update()

	// Iron cost should be 50 because of 0.5 Debasement.
	if capStorage.Iron != 100 {
		t.Fatalf("Expected Iron to drop to 100 (costing exactly 50), got %v", capStorage.Iron)
	}

	// 2. Find the physically spawned debased coin
	coinQuery := world.Query(filter.All(coinTagID, currencyID, posID))
	coinCount := 0
	var debasedVal float32 = 0.0

	for coinQuery.Next() {
		coinCount++
		pos := (*components.Position)(coinQuery.Get(posID))
		curr := (*components.CurrencyComponent)(coinQuery.Get(currencyID))

		if pos.X != 10.0 || pos.Y != 20.0 {
			t.Fatalf("Coin generated at wrong position")
		}

		if curr.IssuerID != 1 {
			t.Fatalf("Coin generated with wrong IssuerID")
		}

		debasedVal = curr.Debasement
	}

	if coinCount != 1 {
		t.Fatalf("Expected exactly 1 coin to be physically spawned, found %v", coinCount)
	}

	if debasedVal != 0.5 {
		t.Fatalf("Expected physically spawned coin to inherit 0.5 debasement, got %v", debasedVal)
	}

	// 3. Trigger InflationSystem to simulate the Butterfly Effect of organic physical flow
	infSys.Update()

	// Market multiplier = 1.0 + 0.5 = 1.5. So 10.0 * 1.5 = 15.0
	if capMarket.FoodPrice != 15.0 {
		t.Fatalf("Expected Market Prices to spike organically to 15.0 due to inflation, got %v", capMarket.FoodPrice)
	}
	if capMarket.IronPrice != 15.0 {
		t.Fatalf("Expected Iron Prices to spike organically to 15.0 due to inflation, got %v", capMarket.IronPrice)
	}
}
