package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// TestTaxationSystem_Deterministic verifies Phase 16.1 DOD taxation loops
func TestTaxationSystem_Deterministic(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// Pre-register components required by the system
	ecs.ComponentID[components.LoyaltyComponent](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.NPC](&world)

	sys := NewTaxationSystem(&world, hookGraph)

	// Component IDs
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)

	// 1. Create a Country Capital entity
	capitalEntity := world.NewEntity(countryID, capitalID, affilID, treasuryID, jurID, identID)

	capAffil := (*components.Affiliation)(world.Get(capitalEntity, affilID))
	capAffil.CountryID = 5 // Represents Country 5

	capTreasury := (*components.TreasuryComponent)(world.Get(capitalEntity, treasuryID))
	capTreasury.Wealth = 0.0

	capJur := (*components.JurisdictionComponent)(world.Get(capitalEntity, jurID))
	capJur.Corruption = 0 // No corruption

	capIdent := (*components.Identity)(world.Get(capitalEntity, identID))
	capIdent.ID = 999

	// 2. Create a Sub-City (Village) entity that belongs to Country 5
	villageEntity := world.NewEntity(villageID, affilID, marketID, treasuryID, loyaltyID)

	vilAffil := (*components.Affiliation)(world.Get(villageEntity, affilID))
	vilAffil.CountryID = 5 // Village is inside Country 5

	vilMarket := (*components.MarketComponent)(world.Get(villageEntity, marketID))
	vilMarket.FoodPrice = 2.0
	vilMarket.WoodPrice = 3.0
	vilMarket.StonePrice = 1.0
	vilMarket.IronPrice = 4.0 // Total prices = 10.0

	vilTreasury := (*components.TreasuryComponent)(world.Get(villageEntity, treasuryID))
	vilTreasury.Wealth = 100.0 // Starting Wealth

	vilLoyalty := (*components.LoyaltyComponent)(world.Get(villageEntity, loyaltyID))
	vilLoyalty.Value = 100 // High loyalty

	// 3. Create another Sub-City (Village) that belongs to a different country (Country 2)
	otherVillageEntity := world.NewEntity(villageID, affilID, marketID, treasuryID, loyaltyID)

	otherAffil := (*components.Affiliation)(world.Get(otherVillageEntity, affilID))
	otherAffil.CountryID = 2 // Village is not inside Country 5

	otherMarket := (*components.MarketComponent)(world.Get(otherVillageEntity, marketID))
	otherMarket.FoodPrice = 5.0
	otherMarket.WoodPrice = 5.0 // Total prices = 10.0

	otherTreasury := (*components.TreasuryComponent)(world.Get(otherVillageEntity, treasuryID))
	otherTreasury.Wealth = 100.0 // Starting Wealth

	otherLoyalty := (*components.LoyaltyComponent)(world.Get(otherVillageEntity, loyaltyID))
	otherLoyalty.Value = 100

	// 4. Run system exactly 99 times. No taxation should occur yet.
	for i := 0; i < 99; i++ {
		sys.Update()
	}

	if capTreasury.Wealth != 0.0 {
		t.Fatalf("Expected Capital Wealth to remain 0.0 before tick 100, got %v", capTreasury.Wealth)
	}

	if vilTreasury.Wealth != 100.0 {
		t.Fatalf("Expected Village Wealth to remain 100.0 before tick 100, got %v", vilTreasury.Wealth)
	}

	// 5. Run tick 100. Taxation should occur.
	sys.Update()

	// Tax amount = (2.0 + 3.0 + 1.0 + 4.0) * 1.0 = 10.0
	// Village Wealth = 100.0 - 10.0 = 90.0
	// Capital Wealth = 0.0 + 10.0 = 10.0

	if vilTreasury.Wealth != 90.0 {
		t.Fatalf("Expected Village Wealth to drop to 90.0, got %v", vilTreasury.Wealth)
	}

	if capTreasury.Wealth != 10.0 {
		t.Fatalf("Expected Capital Wealth to increase to 10.0, got %v", capTreasury.Wealth)
	}

	// 6. Check that other village was unaffected
	if otherTreasury.Wealth != 100.0 {
		t.Fatalf("Expected unaffiliated Village Wealth to remain 100.0, got %v", otherTreasury.Wealth)
	}
}

// TestTaxationSystem_Evasion verifies Phase 42 The Tax Evasion Engine
func TestTaxationSystem_Evasion(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// Pre-register components required by the system
	ecs.ComponentID[components.LoyaltyComponent](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.NPC](&world)

	sys := NewTaxationSystem(&world, hookGraph)

	// Component IDs
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	marketID := ecs.ComponentID[components.MarketComponent](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// 1. Create a Country Capital entity with HIGH CORRUPTION
	capitalEntity := world.NewEntity(countryID, capitalID, affilID, treasuryID, jurID, identID)

	capAffil := (*components.Affiliation)(world.Get(capitalEntity, affilID))
	capAffil.CountryID = 5 // Represents Country 5

	capTreasury := (*components.TreasuryComponent)(world.Get(capitalEntity, treasuryID))
	capTreasury.Wealth = 0.0

	capJur := (*components.JurisdictionComponent)(world.Get(capitalEntity, jurID))
	capJur.Corruption = 50 // HIGH CORRUPTION

	capIdent := (*components.Identity)(world.Get(capitalEntity, identID))
	capIdent.ID = 777 // Ruler ID

	// 2. Create a Sub-City (Village) entity that belongs to Country 5 with LOW LOYALTY
	villageEntity := world.NewEntity(villageID, affilID, marketID, treasuryID, loyaltyID)

	vilAffil := (*components.Affiliation)(world.Get(villageEntity, affilID))
	vilAffil.CountryID = 5 // Village is inside Country 5
	vilAffil.CityID = 100

	vilMarket := (*components.MarketComponent)(world.Get(villageEntity, marketID))
	vilMarket.FoodPrice = 2.0
	vilMarket.WoodPrice = 3.0
	vilMarket.StonePrice = 1.0
	vilMarket.IronPrice = 4.0 // Total prices = 10.0

	vilTreasury := (*components.TreasuryComponent)(world.Get(villageEntity, treasuryID))
	vilTreasury.Wealth = 100.0 // Starting Wealth

	vilLoyalty := (*components.LoyaltyComponent)(world.Get(villageEntity, loyaltyID))
	vilLoyalty.Value = 30 // Low loyalty (30 < 50) -> SHOULD EVADE

	// 3. Create an NPC inside the village
	npcEntity := world.NewEntity(npcID, identID, affilID)
	npcIdent := (*components.Identity)(world.Get(npcEntity, identID))
	npcIdent.ID = 12345

	npcAffil := (*components.Affiliation)(world.Get(npcEntity, affilID))
	npcAffil.CityID = 100

	// 4. Run system exactly 100 times. Taxation should trigger, but evasion occurs.
	for i := 0; i < 100; i++ {
		sys.Update()
	}

	// 5. Verify Evasion Effects

	// Wealth should be completely unchanged
	if vilTreasury.Wealth != 100.0 {
		t.Fatalf("Expected Village Wealth to remain 100.0 due to evasion, got %v", vilTreasury.Wealth)
	}

	if capTreasury.Wealth != 0.0 {
		t.Fatalf("Expected Capital Wealth to remain 0.0 due to evasion, got %v", capTreasury.Wealth)
	}

	// Hook graph should contain a massive negative grudge (-50) against the capital ruler from the NPC
	grudge := hookGraph.GetHook(npcIdent.ID, capIdent.ID)
	if grudge != -50 {
		t.Fatalf("Expected NPC to hold a -50 hook against Capital Ruler due to tax evasion/enforcement, got %d", grudge)
	}
}
