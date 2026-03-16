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
