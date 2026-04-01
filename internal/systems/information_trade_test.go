package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"testing"
)

// Phase 34.1: Information Trade Butterfly Effect E2E Test
// Proves that Information is a tangible commodity in the ECS, bridging
// Memetics (Secrets) with Economics (Needs.Wealth) and Justice (Desperation/Crime).

func TestInformationTradeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize SecretRegistry to avoid panics when generating rumors
	engine.GetSecretRegistry()

	// Initialize component mappings
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	ruinID := ecs.ComponentID[components.RuinComponent](&world)

	hookGraph := engine.NewSparseHookGraph()
	tradeSys := NewInformationTradeSystem(&world, hookGraph)

	// 1. Create a Poor NPC (High Desperation Risk) with a High-Value Secret
	poorNPC := world.NewEntity(posID, secretID, needsID, identID, memID)
	poorPos := (*components.Position)(world.Get(poorNPC, posID))
	poorPos.X = 10.0
	poorPos.Y = 10.0

	poorNeeds := (*components.Needs)(world.Get(poorNPC, needsID))
	poorNeeds.Wealth = 5.0 // Below starvation/theft threshold in DesperationSystem
	poorNeeds.Food = 20.0  // Hungry

	poorIdent := (*components.Identity)(world.Get(poorNPC, identID))
	poorIdent.ID = 100
	poorIdent.BaseTraits = components.TraitGossip // Opportunist

	poorSecrets := (*components.SecretComponent)(world.Get(poorNPC, secretID))
	poorSecrets.Secrets = append(poorSecrets.Secrets, components.Secret{
		OriginID: 100,
		SecretID: 42,
		Virality: 250, // High value
		BeliefID: 0,
	})

	poorMem := (*components.Memory)(world.Get(poorNPC, memID))
	poorMem.Head = 0

	// 2. Create a Wealthy NPC (Target Buyer) without the Secret
	wealthyNPC := world.NewEntity(posID, secretID, needsID, identID, memID)
	wealthyPos := (*components.Position)(world.Get(wealthyNPC, posID))
	wealthyPos.X = 10.5
	wealthyPos.Y = 10.5 // Overlapping proximity

	wealthyNeeds := (*components.Needs)(world.Get(wealthyNPC, needsID))
	wealthyNeeds.Wealth = 1000.0 // Very wealthy
	wealthyNeeds.Food = 100.0

	wealthyIdent := (*components.Identity)(world.Get(wealthyNPC, identID))
	wealthyIdent.ID = 200

	wealthySecrets := (*components.SecretComponent)(world.Get(wealthyNPC, secretID))
	wealthySecrets.Secrets = []components.Secret{} // Doesn't know the secret

	wealthyMem := (*components.Memory)(world.Get(wealthyNPC, memID))
	wealthyMem.Head = 0

	// Pre-Trade Assertions
	if poorNeeds.Wealth != 5.0 {
		t.Fatalf("Expected Poor NPC starting wealth 5.0, got %f", poorNeeds.Wealth)
	}
	if len(wealthySecrets.Secrets) != 0 {
		t.Fatalf("Expected Wealthy NPC to have 0 secrets, got %d", len(wealthySecrets.Secrets))
	}

	// 3. Execute Trade System (Needs to run 15 times to hit offset tick)
	for i := 0; i < 15; i++ {
		tradeSys.Update(&world)
	}

	// Post-Trade Assertions

	// Re-fetch pointers
	poorNeeds = (*components.Needs)(world.Get(poorNPC, needsID))
	wealthyNeeds = (*components.Needs)(world.Get(wealthyNPC, needsID))
	wealthySecrets = (*components.SecretComponent)(world.Get(wealthyNPC, secretID))

	// Assertion A: Economic Impact (Wealth Transferred)
	// Price = Virality / 10 = 250 / 10 = 25.0
	expectedPoorWealth := float32(5.0 + 25.0)
	expectedWealthyWealth := float32(1000.0 - 25.0)

	if poorNeeds.Wealth != expectedPoorWealth {
		t.Errorf("Expected Poor NPC wealth to increase to %f, got %f. Trade failed.", expectedPoorWealth, poorNeeds.Wealth)
	}
	if wealthyNeeds.Wealth != expectedWealthyWealth {
		t.Errorf("Expected Wealthy NPC wealth to decrease to %f, got %f. Trade failed.", expectedWealthyWealth, wealthyNeeds.Wealth)
	}

	// Assertion B: Memetic Impact (Secret Transferred)
	if len(wealthySecrets.Secrets) != 1 {
		t.Fatalf("Expected Wealthy NPC to have 1 secret, got %d. Transfer failed.", len(wealthySecrets.Secrets))
	}
	if wealthySecrets.Secrets[0].SecretID != 42 {
		t.Errorf("Expected Wealthy NPC to learn SecretID 42, got %d", wealthySecrets.Secrets[0].SecretID)
	}

	// Assertion C: Social Impact (Hooks Generated)
	// Mutual hooks generated (+1 each way)
	poorToWealthyHooks := hookGraph.GetAllIncomingHooks(200)
	wealthyToPoorHooks := hookGraph.GetAllIncomingHooks(100)

	if poorToWealthyHooks[100] != 1 {
		t.Errorf("Expected +1 hook from Poor (100) to Wealthy (200), got %d", poorToWealthyHooks[100])
	}
	if wealthyToPoorHooks[200] != 1 {
		t.Errorf("Expected +1 hook from Wealthy (200) to Poor (100), got %d", wealthyToPoorHooks[200])
	}

	// Prevent unused warning for ruinID
	_ = ruinID
}

// Phase 34.2: The Lingua Franca Engine
// Proves that massively wealthy factions can impose their LanguageID on poorer factions during trade.

func TestInformationTradeSystem_LinguaFranca(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize SecretRegistry to avoid panics when generating rumors
	engine.GetSecretRegistry()

	// Initialize component mappings
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)

	hookGraph := engine.NewSparseHookGraph()
	tradeSys := NewInformationTradeSystem(&world, hookGraph)

	// Seller's City (Massively Wealthy)
	sellerCity := world.NewEntity(affilID, treasuryID)
	sellerCityAffil := (*components.Affiliation)(world.Get(sellerCity, affilID))
	sellerCityAffil.CityID = 10
	sellerCityTreasury := (*components.TreasuryComponent)(world.Get(sellerCity, treasuryID))
	sellerCityTreasury.Wealth = 10000.0 // > 5000 and > 5x buyer wealth

	// Buyer's City (Poor)
	buyerCity := world.NewEntity(affilID, treasuryID)
	buyerCityAffil := (*components.Affiliation)(world.Get(buyerCity, affilID))
	buyerCityAffil.CityID = 20
	buyerCityTreasury := (*components.TreasuryComponent)(world.Get(buyerCity, treasuryID))
	buyerCityTreasury.Wealth = 1000.0

	// 1. Create a Seller NPC (Wealthy City) with a High-Value Secret
	sellerNPC := world.NewEntity(posID, secretID, needsID, identID, memID, affilID, cultureID)
	sellerPos := (*components.Position)(world.Get(sellerNPC, posID))
	sellerPos.X = 10.0
	sellerPos.Y = 10.0

	sellerNeeds := (*components.Needs)(world.Get(sellerNPC, needsID))
	sellerNeeds.Wealth = 5.0
	sellerNeeds.Food = 20.0

	sellerIdent := (*components.Identity)(world.Get(sellerNPC, identID))
	sellerIdent.ID = 100
	sellerIdent.BaseTraits = components.TraitGossip

	sellerSecrets := (*components.SecretComponent)(world.Get(sellerNPC, secretID))
	sellerSecrets.Secrets = append(sellerSecrets.Secrets, components.Secret{
		OriginID: 100,
		SecretID: 42,
		Virality: 250,
		BeliefID: 0,
	})

	sellerMem := (*components.Memory)(world.Get(sellerNPC, memID))
	sellerMem.Head = 0

	sellerAffil := (*components.Affiliation)(world.Get(sellerNPC, affilID))
	sellerAffil.CityID = 10

	sellerCulture := (*components.CultureComponent)(world.Get(sellerNPC, cultureID))
	sellerCulture.LanguageID = 55 // Seller's language

	// 2. Create a Buyer NPC (Poor City) without the Secret
	buyerNPC := world.NewEntity(posID, secretID, needsID, identID, memID, affilID, cultureID)
	buyerPos := (*components.Position)(world.Get(buyerNPC, posID))
	buyerPos.X = 10.5
	buyerPos.Y = 10.5

	buyerNeeds := (*components.Needs)(world.Get(buyerNPC, needsID))
	buyerNeeds.Wealth = 1000.0
	buyerNeeds.Food = 100.0

	buyerIdent := (*components.Identity)(world.Get(buyerNPC, identID))
	buyerIdent.ID = 200

	buyerSecrets := (*components.SecretComponent)(world.Get(buyerNPC, secretID))
	buyerSecrets.Secrets = []components.Secret{}

	buyerMem := (*components.Memory)(world.Get(buyerNPC, memID))
	buyerMem.Head = 0

	buyerAffil := (*components.Affiliation)(world.Get(buyerNPC, affilID))
	buyerAffil.CityID = 20

	buyerCulture := (*components.CultureComponent)(world.Get(buyerNPC, cultureID))
	buyerCulture.LanguageID = 99 // Buyer's original language
	buyerCulture.ForeignLanguageID = 0
	buyerCulture.ForeignInteractionTicks = 0

	// 3. Execute Trade System (Needs to run 15 times to hit offset tick)
	for i := 0; i < 15; i++ {
		tradeSys.Update(&world)
	}

	// Post-Trade Assertions
	buyerCulture = (*components.CultureComponent)(world.Get(buyerNPC, cultureID))

	if buyerCulture.ForeignLanguageID != 55 {
		t.Errorf("Expected Buyer's ForeignLanguageID to be updated to 55, got %d", buyerCulture.ForeignLanguageID)
	}
	if buyerCulture.ForeignInteractionTicks != 50 {
		t.Errorf("Expected Buyer's ForeignInteractionTicks to be updated to 50, got %d", buyerCulture.ForeignInteractionTicks)
	}
}
