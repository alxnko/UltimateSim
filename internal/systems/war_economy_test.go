package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestWarEconomySystem_Integration verifies that Phase 50 - Military-Industrial Complex
// correctly drains Iron, spikes IronPrice, drains State Wealth, and ultimately bankrupts
// the country and removes war status while dropping legitimacy.
func TestWarEconomySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewWarEconomySystem(&world)

	capID := ecs.ComponentID[components.CapitalComponent](&world)
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	treasID := ecs.ComponentID[components.TreasuryComponent](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)

	// Create a Capital at war
	e := world.NewEntity(capID, warCompID, storageID, marketID, treasID, legitID)

	war := (*components.WarTrackerComponent)(world.Get(e, warCompID))
	war.Active = true

	storage := (*components.StorageComponent)(world.Get(e, storageID))
	storage.Iron = 6 // Enough for 1 tick of deduction

	market := (*components.MarketComponent)(world.Get(e, marketID))
	market.IronPrice = 1.0

	treasury := (*components.TreasuryComponent)(world.Get(e, treasID))
	treasury.Wealth = 150.0 // Enough for exactly 1 Iron restock

	legit := (*components.LegitimacyComponent)(world.Get(e, legitID))
	legit.Score = 100

	// Tick 1-99: No evaluation
	for i := 1; i < 100; i++ {
		sys.Update(&world)
	}

	if storage.Iron != 6 {
		t.Fatalf("Expected Iron to be 6, got %d", storage.Iron)
	}

	// Tick 100: Iron deducts 5
	sys.Update(&world)
	if storage.Iron != 1 {
		t.Fatalf("Expected Iron to drop to 1, got %d", storage.Iron)
	}

	// Tick 101-199
	for i := 101; i < 200; i++ {
		sys.Update(&world)
	}

	// Tick 200: Iron drops below 5, state purchases
	sys.Update(&world)

	if storage.Iron != 50 { // 0 + 50
		t.Fatalf("Expected Iron to restock to 50, got %d", storage.Iron)
	}
	if treasury.Wealth != 50.0 { // 150 - 100
		t.Fatalf("Expected Wealth to drop to 50, got %f", treasury.Wealth)
	}
	if market.IronPrice != 51.0 { // 1 + 50
		t.Fatalf("Expected IronPrice to spike to 51, got %f", market.IronPrice)
	}
	if !war.Active {
		t.Fatalf("Expected war to still be active")
	}

	// Tick 201-1199: Drain the 50 Iron (10 updates of -5 each)
	for i := 201; i < 1200; i++ {
		sys.Update(&world)
	}

	// At tick 1100, iron is exactly 5. 1100 updates it to 0.
	// Oh wait, 10 updates of -5 is 50.
	// The 10 updates are at ticks 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200.
	// At tick 200, iron went from 1 to 0, then immediately bought 50. So it has 50.
	// Tick 300: 50 -> 45
	// Tick 400: 45 -> 40
	// Tick 500: 40 -> 35
	// Tick 600: 35 -> 30
	// Tick 700: 30 -> 25
	// Tick 800: 25 -> 20
	// Tick 900: 20 -> 15
	// Tick 1000: 15 -> 10
	// Tick 1100: 10 -> 5
	// Tick 1200: 5 -> 0.

	sys.Update(&world) // This is tick 1200. It deducts the last 5 iron.

	if storage.Iron != 0 {
		t.Fatalf("Expected Iron to drain back to 0, got %d", storage.Iron)
	}

	// Tick 1201-1299
	for i := 1201; i < 1300; i++ {
		sys.Update(&world)
	}

	// Tick 1300: State tries to buy but Treasury is 50.0 (Bankrupt)
	sys.Update(&world)

	if war.Active {
		t.Fatalf("Expected state to default and war to become inactive")
	}

	if legit.Score != 50 { // 100 - 50
		t.Fatalf("Expected Legitimacy to drop to 50 due to war loss, got %d", legit.Score)
	}
}
