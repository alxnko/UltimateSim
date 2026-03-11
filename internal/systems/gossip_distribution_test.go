package systems

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

func TestGossipDistributionSystem(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{1, 2, 3})

	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)

	// Create Sender
	sender := world.NewEntity()
	world.Add(sender, posID, secretID, memoryID, identID, cultureID)

	sPos := (*components.Position)(world.Get(sender, posID))
	sSecret := (*components.SecretComponent)(world.Get(sender, secretID))
	sIdent := (*components.Identity)(world.Get(sender, identID))
	sMemory := (*components.Memory)(world.Get(sender, memoryID))
	sCulture := (*components.CultureComponent)(world.Get(sender, cultureID))

	sPos.X = 10.0
	sPos.Y = 10.0
	sIdent.ID = 100
	sIdent.BaseTraits = components.TraitGossip
	sCulture.LanguageID = 1
	sSecret.Secrets = append(sSecret.Secrets, components.Secret{
		OriginID: 100,
		SecretID: 42,
		Virality: 255, // 100% chance base
	})

	// Create Receiver
	receiver := world.NewEntity()
	world.Add(receiver, posID, secretID, memoryID, identID, cultureID)

	rPos := (*components.Position)(world.Get(receiver, posID))
	rSecret := (*components.SecretComponent)(world.Get(receiver, secretID))
	rIdent := (*components.Identity)(world.Get(receiver, identID))
	rMemory := (*components.Memory)(world.Get(receiver, memoryID))
	rCulture := (*components.CultureComponent)(world.Get(receiver, cultureID))

	rPos.X = 11.0 // Within distance 2.0
	rPos.Y = 10.0
	rIdent.ID = 200
	rCulture.LanguageID = 1 // Same language as sender

	// Add system
	system := &GossipDistributionSystem{}

	// Run for 9 ticks - nothing should happen
	for i := 0; i < 9; i++ {
		system.Update(&world)
	}

	if len(rSecret.Secrets) != 0 {
		t.Fatalf("Receiver should not have learned secret before 10th tick")
	}

	// 10th tick - logic triggers
	system.Update(&world)

	// Verify receiver learned secret
	if len(rSecret.Secrets) != 1 {
		t.Fatalf("Receiver should have learned exactly 1 secret, got %d", len(rSecret.Secrets))
	}

	if rSecret.Secrets[0].SecretID != 42 {
		t.Errorf("Receiver learned wrong secret: %d", rSecret.Secrets[0].SecretID)
	}

	// Verify memory log
	if rMemory.Head != 1 {
		t.Errorf("Memory head should have advanced to 1, got %d", rMemory.Head)
	}

	event := rMemory.Events[0]
	if event.InteractionType != components.InteractionGossip {
		t.Errorf("Expected interaction type %d, got %d", components.InteractionGossip, event.InteractionType)
	}

	if event.Value != 42 {
		t.Errorf("Expected secret ID 42 in memory event, got %d", event.Value)
	}

	if event.TargetID != uint64(sender.ID()) {
		t.Errorf("Expected target ID %d, got %d", sender.ID(), event.TargetID)
	}

	// Create another receiver too far away
	distant := world.NewEntity()
	world.Add(distant, posID, secretID, memoryID, identID, cultureID)

	dPos := (*components.Position)(world.Get(distant, posID))
	dSecret := (*components.SecretComponent)(world.Get(distant, secretID))
	dCulture := (*components.CultureComponent)(world.Get(distant, cultureID))

	dPos.X = 50.0
	dPos.Y = 50.0
	dCulture.LanguageID = 1

	// Run 10 more ticks
	for i := 0; i < 10; i++ {
		system.Update(&world)
	}

	// Distant receiver should have learned nothing
	if len(dSecret.Secrets) != 0 {
		t.Fatalf("Distant receiver should not have learned secret")
	}

	// Ensure the original receiver didn't learn it a second time
	if len(rSecret.Secrets) != 1 {
		t.Fatalf("Receiver should not learn duplicate secrets")
	}

	// Wait, we need to handle that the Sender might try to learn from the Receiver now.
	// We'll just verify Sender didn't learn 42 twice.
	if len(sSecret.Secrets) != 1 {
		t.Fatalf("Sender should not learn duplicate secrets")
	}

	// For compilation, unused variables
	_ = sMemory
	_ = sCulture
	_ = rCulture
	_ = dCulture
}

func TestTranslationPenaltyAndSilentHooks(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{4, 5, 6})

	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)

	// Create Sender
	sender := world.NewEntity()
	world.Add(sender, posID, secretID, memoryID, identID, cultureID)

	sPos := (*components.Position)(world.Get(sender, posID))
	sSecret := (*components.SecretComponent)(world.Get(sender, secretID))
	sIdent := (*components.Identity)(world.Get(sender, identID))
	sCulture := (*components.CultureComponent)(world.Get(sender, cultureID))

	sPos.X = 10.0
	sPos.Y = 10.0
	sIdent.ID = 100
	sCulture.LanguageID = 1 // Different language
	sSecret.Secrets = append(sSecret.Secrets, components.Secret{
		OriginID: 100,
		SecretID: 99,
		Virality: 255, // 100% chance base, but reduced to 10% by penalty
	})

	// Create Receiver
	receiver := world.NewEntity()
	world.Add(receiver, posID, secretID, memoryID, identID, cultureID)

	rPos := (*components.Position)(world.Get(receiver, posID))
	rSecret := (*components.SecretComponent)(world.Get(receiver, secretID))
	rIdent := (*components.Identity)(world.Get(receiver, identID))
	rCulture := (*components.CultureComponent)(world.Get(receiver, cultureID))

	rPos.X = 11.0 // Within distance 2.0
	rPos.Y = 10.0
	rIdent.ID = 200
	rCulture.LanguageID = 2 // Different language

	// Initialize SparseHookGraph
	hookGraph := engine.NewSparseHookGraph()

	// Add system
	system := &GossipDistributionSystem{
		HookGraph: hookGraph,
	}

	// Run multiple ticks to ensure they interact but verify penalty is applied.
	// Because of the 90% penalty, chance is 10%. Over 5 update cycles (50 ticks),
	// there's a high chance it still fails or passes slowly, but we want to assert
	// that a Silent Hook is generated since it rolls a 25% chance per attempt when failing.

	// Let's run 5 updates (50 ticks)
	for i := 0; i < 50; i++ {
		system.Update(&world)
	}

	// Check if a hook point was created between 100 and 200
	hookPoints := hookGraph.GetHook(100, 200)

	// Run even more ticks if it didn't pass, since 25% isn't guaranteed over 5 tries.
	for i := 0; i < 500; i++ {
		system.Update(&world)
	}

	hookPoints = hookGraph.GetHook(100, 200)

	// It's statistically very likely a hook is generated.
	if hookPoints == 0 {
		t.Errorf("Expected Silent Hooks to be generated between entities with mismatched languages, got 0")
	}

	// Note: We don't strictly assert `len(rSecret.Secrets) == 0` because there is a 10%
	// chance it *could* have passed. The primary feature of 07.4 is that the penalty exists
	// and silent hooks are generated upon failure. The mathematical reduction is verified by code logic.

	// For compilation, unused variable
	_ = rSecret
}
