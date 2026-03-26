package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

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
