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
// TestWarEconomySystem_Integration verifies the War Economy Engine's systemic triggers (Phase 50).
func TestWarEconomySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	warID := ecs.ComponentID[components.WarTrackerComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	treasID := ecs.ComponentID[components.TreasuryComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)

	sys := NewWarEconomySystem(&world)

	capital := world.NewEntity()
	world.Add(capital, warID, storageID, treasID, marketID, legitID)

	// Initialize the component states
	war := (*components.WarTrackerComponent)(world.Get(capital, warID))
	war.Active = true

	storage := (*components.StorageComponent)(world.Get(capital, storageID))
	storage.Iron = 10 // Exact amount needed for one war tick

	treasury := (*components.TreasuryComponent)(world.Get(capital, treasID))
	treasury.Wealth = 100 // Exact amount needed for one emergency iron purchase

	market := (*components.MarketComponent)(world.Get(capital, marketID))
	market.IronPrice = 5.0

	legitimacy := (*components.LegitimacyComponent)(world.Get(capital, legitID))
	legitimacy.Score = 100

	// Tick 50: Should drain 10 Iron
	sys.tickCounter = 49
	sys.Update(&world)

	if storage.Iron != 0 {
		t.Errorf("Expected Iron to drain to 0, got %d", storage.Iron)
	}
	if treasury.Wealth != 100 {
		t.Errorf("Expected Wealth to be untouched during normal drain, got %f", treasury.Wealth)
	}
	if market.IronPrice != 5.0 {
		t.Errorf("Expected IronPrice to remain 5.0 during normal drain, got %f", market.IronPrice)
	}
	if legitimacy.Score != 100 {
		t.Errorf("Expected Legitimacy to remain 100 during normal drain, got %d", legitimacy.Score)
	}

	// Tick 100: Iron is 0, should buy iron using 100 Wealth and increase IronPrice by 5
	sys.tickCounter = 99
	sys.Update(&world)

	if treasury.Wealth != 0 {
		t.Errorf("Expected Wealth to drain to 0, got %f", treasury.Wealth)
	}
	if storage.Iron != 10 {
		t.Errorf("Expected Iron to jump to 10 from emergency buy, got %d", storage.Iron)
	}
	if market.IronPrice != 10.0 {
		t.Errorf("Expected IronPrice to spike to 10.0, got %f", market.IronPrice)
	}
	if legitimacy.Score != 100 {
		t.Errorf("Expected Legitimacy to remain 100 during emergency buy, got %d", legitimacy.Score)
	}

	// Drain Iron again manually to simulate war usage after the emergency buy
	storage.Iron = 0

	// Tick 150: Bankrupt (Iron 0, Wealth 0), should drain Legitimacy
	sys.tickCounter = 149
	sys.Update(&world)

	if legitimacy.Score != 90 {
		t.Errorf("Expected Legitimacy to drop to 90 due to bankruptcy, got %d", legitimacy.Score)
	}
	if treasury.Wealth != 0 {
		t.Errorf("Expected Wealth to remain 0, got %f", treasury.Wealth)
	}
	if storage.Iron != 0 {
		t.Errorf("Expected Iron to remain 0, got %d", storage.Iron)
	}

	// Check determinism with a second world instance
	world2 := ecs.NewWorld()
	sys2 := NewWarEconomySystem(&world2)

	capital2 := world2.NewEntity()
	world2.Add(capital2, warID, storageID, treasID, marketID, legitID)

	war2 := (*components.WarTrackerComponent)(world2.Get(capital2, warID))
	war2.Active = true

	storage2 := (*components.StorageComponent)(world2.Get(capital2, storageID))
	storage2.Iron = 10

	treasury2 := (*components.TreasuryComponent)(world2.Get(capital2, treasID))
	treasury2.Wealth = 100

	market2 := (*components.MarketComponent)(world2.Get(capital2, marketID))
	market2.IronPrice = 5.0

	legitimacy2 := (*components.LegitimacyComponent)(world2.Get(capital2, legitID))
	legitimacy2.Score = 100

	sys2.tickCounter = 49
	sys2.Update(&world2)

	sys2.tickCounter = 99
	sys2.Update(&world2)

	storage2.Iron = 0

	sys2.tickCounter = 149
	sys2.Update(&world2)

	if legitimacy2.Score != legitimacy.Score || market2.IronPrice != market.IronPrice {
		t.Errorf("Determinism check failed between worlds: Legitimacy (%d vs %d), IronPrice (%f vs %f)",
			legitimacy.Score, legitimacy2.Score, market.IronPrice, market2.IronPrice)
	}
}
