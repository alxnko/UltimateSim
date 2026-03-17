package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 41: The Ostracization Engine (Integration Test)
// Proves the Butterfly Effect: Unpunished theft leads to deep negative hooks,
// which physically isolates the thief from the Information Economy (blocking trades).
func TestOstracizationSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize SecretRegistry to avoid panics
	engine.GetSecretRegistry()

	// Initialize component mappings
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	memID := ecs.ComponentID[components.Memory](&world)

	hookGraph := engine.NewSparseHookGraph()

	ostraSys := NewOstracizationSystem(&world, hookGraph)
	tradeSys := NewInformationTradeSystem(&world, hookGraph)

	// 1. Create NPC 1 (Victim/Seller) with a High-Value Secret and a memory of being robbed
	victimNPC := world.NewEntity(posID, secretID, needsID, identID, memID)
	vPos := (*components.Position)(world.Get(victimNPC, posID))
	vPos.X = 10.0
	vPos.Y = 10.0

	vNeeds := (*components.Needs)(world.Get(victimNPC, needsID))
	vNeeds.Wealth = 5.0 // Poor

	vIdent := (*components.Identity)(world.Get(victimNPC, identID))
	vIdent.ID = 100
	vIdent.BaseTraits = components.TraitGossip // Opportunist to force a trade

	vSecrets := (*components.SecretComponent)(world.Get(victimNPC, secretID))
	vSecrets.Secrets = append(vSecrets.Secrets, components.Secret{
		OriginID: 100,
		SecretID: 42,
		Virality: 250, // High value
		BeliefID: 0,
	})

	vMem := (*components.Memory)(world.Get(victimNPC, memID))
	// Inject two InteractionTheft events against NPC 2 (ID: 200) to push the hook to -40
	vMem.Events[0] = components.MemoryEvent{
		TargetID:        200,
		InteractionType: components.InteractionTheft,
	}
	vMem.Events[1] = components.MemoryEvent{
		TargetID:        200,
		InteractionType: components.InteractionTheft,
	}
	vMem.Head = 2

	// 2. Create NPC 2 (Thief/Buyer) without the Secret
	thiefNPC := world.NewEntity(posID, secretID, needsID, identID, memID)
	tPos := (*components.Position)(world.Get(thiefNPC, posID))
	tPos.X = 10.5
	tPos.Y = 10.5 // Overlapping proximity

	tNeeds := (*components.Needs)(world.Get(thiefNPC, needsID))
	tNeeds.Wealth = 1000.0 // Very wealthy

	tIdent := (*components.Identity)(world.Get(thiefNPC, identID))
	tIdent.ID = 200

	tSecrets := (*components.SecretComponent)(world.Get(thiefNPC, secretID))
	tSecrets.Secrets = []components.Secret{} // Doesn't know the secret

	tMem := (*components.Memory)(world.Get(thiefNPC, memID))
	tMem.Head = 0

	// Pre-Trade Assertions
	if vNeeds.Wealth != 5.0 {
		t.Fatalf("Expected Victim NPC starting wealth 5.0, got %f", vNeeds.Wealth)
	}
	if len(tSecrets.Secrets) != 0 {
		t.Fatalf("Expected Thief NPC to have 0 secrets, got %d", len(tSecrets.Secrets))
	}
	if hookGraph.GetHook(100, 200) != 0 {
		t.Fatalf("Expected starting hook to be 0")
	}

	// 3. Run Ostracization System (Needs to run 20 times to hit offset tick)
	for i := 0; i < 20; i++ {
		ostraSys.Update(&world)
	}

	// Post-Ostracization Assertions
	grudge := hookGraph.GetHook(100, 200)
	if grudge != -40 {
		t.Errorf("Expected Victim to form a -40 grudge against Thief from memory, got %d", grudge)
	}

	// Ensure memory events were cleared
	if vMem.Events[0].InteractionType != 0 || vMem.Events[1].InteractionType != 0 {
		t.Errorf("Expected processed memory events to be cleared to 0")
	}

	// 4. Execute Trade System (Needs to run 15 times to hit offset tick)
	for i := 0; i < 15; i++ {
		tradeSys.Update(&world)
	}

	// Post-Trade Assertions

	// Re-fetch pointers
	vNeeds = (*components.Needs)(world.Get(victimNPC, needsID))
	tNeeds = (*components.Needs)(world.Get(thiefNPC, needsID))
	tSecrets = (*components.SecretComponent)(world.Get(thiefNPC, secretID))

	// Assertion: Trade was blocked
	if vNeeds.Wealth != 5.0 {
		t.Errorf("Trade occurred! Victim NPC wealth changed to %f (expected 5.0).", vNeeds.Wealth)
	}
	if tNeeds.Wealth != 1000.0 {
		t.Errorf("Trade occurred! Thief NPC wealth changed to %f (expected 1000.0).", tNeeds.Wealth)
	}
	if len(tSecrets.Secrets) != 0 {
		t.Fatalf("Trade occurred! Thief NPC learned the secret.")
	}
}
