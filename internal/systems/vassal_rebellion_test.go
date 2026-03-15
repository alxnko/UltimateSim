package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 28.1: The Vassal Rebellion Engine (E2E Tests)
func TestVassalRebellionSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()
	sys := NewVassalRebellionSystem(&world, hookGraph)

	// Component IDs
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	desperationID := ecs.ComponentID[components.DesperationComponent](&world)

	// 1. Create a Country Capital (The Ruling Entity)
	capital := world.NewEntity(countryID, capitalID, affilID, identID)

	capCountry := (*components.CountryComponent)(world.Get(capital, countryID))
	capCountry.Debasement = 0.5 // High debasement to trigger drain

	capAffil := (*components.Affiliation)(world.Get(capital, affilID))
	capAffil.CountryID = 1

	capIdent := (*components.Identity)(world.Get(capital, identID))
	capIdent.ID = 100 // Ruler ID

	// 2. Create a Sub-City (Village)
	village := world.NewEntity(villageID, affilID, loyaltyID, marketID)

	vilAffil := (*components.Affiliation)(world.Get(village, affilID))
	vilAffil.CountryID = 1 // Bound to Country 1
	vilAffil.CityID = 10   // Its own City ID

	vilLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	vilLoyalty.Value = 3 // Very low loyalty

	vilMarket := (*components.MarketComponent)(world.Get(village, marketID))
	vilMarket.FoodPrice = 2.0 // Normal price

	// 3. Create a desperate citizen inside the Village
	citizen := world.NewEntity(npcID, affilID, desperationID, identID)

	citAffil := (*components.Affiliation)(world.Get(citizen, affilID))
	citAffil.CountryID = 1
	citAffil.CityID = 10

	citDesp := (*components.DesperationComponent)(world.Get(citizen, desperationID))
	citDesp.Level = 55 // Highly desperate

	citIdent := (*components.Identity)(world.Get(citizen, identID))
	citIdent.ID = 200

	// Pre-test asserts
	if vilAffil.CountryID != 1 {
		t.Fatalf("Expected village to start affiliated with Country 1")
	}

	// 4. Run system to exactly 99 ticks (no processing should occur)
	for i := 0; i < 99; i++ {
		sys.Update(&world)
	}

	if vilLoyalty.Value != 3 {
		t.Fatalf("Expected loyalty to be 3 before tick 100, got %v", vilLoyalty.Value)
	}
	if vilAffil.CountryID != 1 {
		t.Fatalf("Expected village to remain affiliated before tick 100")
	}

	// 5. Run tick 100 (Rebellion System fires)
	sys.Update(&world)

	// Debasement is 0.5. Drain = 0.5 * 10 = 5.
	// Loyalty starts at 3. 3 - 5 <= 0. Drops to 0. Village secedes!
	if vilLoyalty.Value != 0 {
		t.Fatalf("Expected loyalty to drop to 0, got %v", vilLoyalty.Value)
	}

	if vilAffil.CountryID != 0 {
		t.Fatalf("Expected village to secede (CountryID = 0), got %v", vilAffil.CountryID)
	}

	// 6. Check if BloodFeud hook was created
	hooks := hookGraph.GetAllIncomingHooks(capIdent.ID)
	foundGrudge := false
	for originID, points := range hooks {
		if originID == citIdent.ID && points == -100 {
			foundGrudge = true
			break
		}
	}

	if !foundGrudge {
		t.Fatalf("Expected massive negative hook (-100) from Citizen 200 to Ruler 100, not found.")
	}
}

// Phase 28.1: Test Extreme Food Prices (Famine) triggering Secession
func TestVassalRebellionSystem_Famine(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()
	sys := NewVassalRebellionSystem(&world, hookGraph)

	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)

	capital := world.NewEntity(countryID, capitalID, affilID, identID)

	capCountry := (*components.CountryComponent)(world.Get(capital, countryID))
	capCountry.Debasement = 0.0 // Zero debasement

	capAffil := (*components.Affiliation)(world.Get(capital, affilID))
	capAffil.CountryID = 2

	capIdent := (*components.Identity)(world.Get(capital, identID))
	capIdent.ID = 101

	village := world.NewEntity(villageID, affilID, loyaltyID, marketID)

	vilAffil := (*components.Affiliation)(world.Get(village, affilID))
	vilAffil.CountryID = 2
	vilAffil.CityID = 20

	vilLoyalty := (*components.LoyaltyComponent)(world.Get(village, loyaltyID))
	vilLoyalty.Value = 4

	vilMarket := (*components.MarketComponent)(world.Get(village, marketID))
	vilMarket.FoodPrice = 10.0 // Extreme Famine Price

	// Drain = 10.0 - 5.0 = 5

	for i := 0; i < 100; i++ {
		sys.Update(&world)
	}

	if vilLoyalty.Value != 0 {
		t.Fatalf("Expected loyalty to drop to 0 due to famine, got %v", vilLoyalty.Value)
	}

	if vilAffil.CountryID != 0 {
		t.Fatalf("Expected village to secede due to famine (CountryID = 0), got %v", vilAffil.CountryID)
	}
}
