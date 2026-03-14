package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// TestMintingSystem_Deterministic Minting verifies Phase 15.3 DOD caching logic
func TestMintingSystem_DeterministicMinting(t *testing.T) {
	world := ecs.NewWorld()

	sys := NewMintingSystem(&world)

	// Register necessary components
	villageID := ecs.ComponentID[components.Village](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)
	coinTagID := ecs.ComponentID[components.CoinEntity](&world)
	currencyID := ecs.ComponentID[components.CurrencyComponent](&world)

	// Create test capital entity
	cityEntity := world.NewEntity(villageID, capitalID, countryID, storageID, affilID, posID)
	storage := (*components.StorageComponent)(world.Get(cityEntity, storageID))
	storage.Iron = 150 // Enough to mint

	affil := (*components.Affiliation)(world.Get(cityEntity, affilID))
	affil.CityID = 77

	pos := (*components.Position)(world.Get(cityEntity, posID))
	pos.X = 10.0
	pos.Y = 20.0

	// Run system exactly 99 times. No coin should be minted yet.
	for i := 0; i < 99; i++ {
		sys.Update()
	}

	if storage.Iron != 150 {
		t.Fatalf("Expected Iron to remain 150 before tick 100, got %v", storage.Iron)
	}

	query := world.Query(filter.All(coinTagID, currencyID))
	if query.Count() != 0 {
		t.Fatalf("Expected 0 minted coins before tick 100, got %v", query.Count())
	}
	query.Close()

	// Run tick 100. Minting should occur.
	sys.Update()

	if storage.Iron != 50 {
		t.Fatalf("Expected Iron to drop to 50, got %v", storage.Iron)
	}

	query = world.Query(filter.All(coinTagID, currencyID, posID))
	if query.Count() != 1 {
		t.Fatalf("Expected exactly 1 minted coin, got %v", query.Count())
	}

	// Verify coin parameters
	for query.Next() {
		curr := (*components.CurrencyComponent)(query.Get(currencyID))
		coinPos := (*components.Position)(query.Get(posID))

		if curr.IssuerID != 77 {
			t.Fatalf("Expected coin IssuerID to be 77, got %v", curr.IssuerID)
		}
		if curr.Value != 100.0 {
			t.Fatalf("Expected baseline coin value to be 100.0, got %v", curr.Value)
		}
		if coinPos.X != 10.0 || coinPos.Y != 20.0 {
			t.Fatalf("Expected coin position to match capital (10, 20), got (%v, %v)", coinPos.X, coinPos.Y)
		}
	}
}
