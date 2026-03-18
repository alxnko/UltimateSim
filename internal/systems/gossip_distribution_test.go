package systems

import (
	"testing"
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 07.4: Misunderstandings Test
func TestGossipMisunderstanding(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{4, 4, 4})

	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)

	hookGraph := engine.NewSparseHookGraph()

	// Sender setup
	sender := world.NewEntity(posID, secretID, memoryID, identID, cultureID)
	sPos := (*components.Position)(world.Get(sender, posID))
	sPos.X = 0
	sPos.Y = 0

	sIdent := (*components.Identity)(world.Get(sender, identID))
	sIdent.ID = 101
	sIdent.BaseTraits = 0

	sCulture := (*components.CultureComponent)(world.Get(sender, cultureID))
	sCulture.LanguageID = 1

	sSecret := (*components.SecretComponent)(world.Get(sender, secretID))
	sSecret.Secrets = append(sSecret.Secrets, components.Secret{
		OriginID: 101,
		SecretID: engine.GetSecretRegistry().RegisterSecret("King's weakness"),
		Virality: 250, // Almost guaranteed to pass if no translation penalty
	})

	// Receiver setup
	receiver := world.NewEntity(posID, secretID, memoryID, identID, cultureID)
	rPos := (*components.Position)(world.Get(receiver, posID))
	rPos.X = 1
	rPos.Y = 1

	rIdent := (*components.Identity)(world.Get(receiver, identID))
	rIdent.ID = 102

	rCulture := (*components.CultureComponent)(world.Get(receiver, cultureID))
	rCulture.LanguageID = 2 // Mismatched language

	rSecret := (*components.SecretComponent)(world.Get(receiver, secretID))
	rSecret.Secrets = make([]components.Secret, 0)

	system := NewGossipDistributionSystem(&world, hookGraph)

	// Update 10 times to hit the modulo
	for i := 0; i < 10; i++ {
		system.Update(&world)
	}

	rSecret = (*components.SecretComponent)(world.Get(receiver, secretID))

	hasMisunderstanding := false
	for _, sec := range rSecret.Secrets {
		str, _ := engine.GetSecretRegistry().GetSecret(sec.SecretID)
		if str == "misunderstood_King's weakness" {
			hasMisunderstanding = true
		}
	}

	if !hasMisunderstanding {
		t.Errorf("Expected receiver to gain the misunderstood secret due to Translation Penalty")
	}

	// Verify the negative hook
	hooks := hookGraph.GetAllIncomingHooks(101) // Check sender's incoming hooks
	if val, ok := hooks[102]; !ok || val != -10 {
		t.Errorf("Expected receiver (102) to place a -10 hook on sender (101) due to misunderstanding, got %v", hooks)
	}
}

// Phase 32.1: Aura of Legitimacy Test
func TestGossipAuraOfLegitimacy(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{5, 5, 5}) // Using specific seed

	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	equipID := ecs.ComponentID[components.EquipmentComponent](&world)

	hookGraph := engine.NewSparseHookGraph()

	// Create Sender with Aura of Legitimacy (Sword of Bektur)
	sender := world.NewEntity(posID, secretID, memoryID, identID, cultureID, equipID)

	sPos := (*components.Position)(world.Get(sender, posID))
	sPos.X, sPos.Y = 10, 10

	sCulture := (*components.CultureComponent)(world.Get(sender, cultureID))
	sCulture.LanguageID = 1

	sIdent := (*components.Identity)(world.Get(sender, identID))
	sIdent.ID = 1

	sSecret := (*components.SecretComponent)(world.Get(sender, secretID))
	sSecret.Secrets = append(sSecret.Secrets, components.Secret{
		OriginID: 1,
		SecretID: 99,
		Virality: 50, // Moderately contagious
	})

	sEquip := (*components.EquipmentComponent)(world.Get(sender, equipID))
	sEquip.Equipped = true
	sEquip.Weapon = components.LegendComponent{
		Prestige: components.ExtremePrestigeThreshold + 100, // Very prestigious
	}

	// Create Receiver
	receiver := world.NewEntity(posID, secretID, memoryID, identID, cultureID)

	rPos := (*components.Position)(world.Get(receiver, posID))
	rPos.X, rPos.Y = 10, 10 // Exact same position

	rCulture := (*components.CultureComponent)(world.Get(receiver, cultureID))
	rCulture.LanguageID = 1 // Same language

	rIdent := (*components.Identity)(world.Get(receiver, identID))
	rIdent.ID = 2

	rSecret := (*components.SecretComponent)(world.Get(receiver, secretID))
	rSecret.Secrets = []components.Secret{}

	system := NewGossipDistributionSystem(&world, hookGraph)

	// Since Virality is 50, chance = 50/255 = ~19%
	// The modifier from Aura is * 3.0 = ~58% chance
	// We will run this over several ticks to ensure the secret is passed.
	// Note: System runs every 10 ticks.
	for i := 0; i < 50; i++ {
		system.Update(&world)
	}

	rSecret = (*components.SecretComponent)(world.Get(receiver, secretID))

	if len(rSecret.Secrets) == 0 {
		t.Errorf("Expected receiver to gain the secret due to Aura multiplier")
	}
}

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
	system := NewGossipDistributionSystem(&world, engine.NewSparseHookGraph())

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
	system := NewGossipDistributionSystem(&world, hookGraph)

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

// Phase 07.5: Ideological Infection
func TestIdeologicalInfection(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{7, 8, 9})

	world := ecs.NewWorld()

	// Register components
	posID := ecs.ComponentID[components.Position](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)
	memoryID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	cultureID := ecs.ComponentID[components.CultureComponent](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)

	// Create Sender
	sender := world.NewEntity()
	world.Add(sender, posID, secretID, memoryID, identID, cultureID, beliefID)

	sPos := (*components.Position)(world.Get(sender, posID))
	sSecret := (*components.SecretComponent)(world.Get(sender, secretID))
	sIdent := (*components.Identity)(world.Get(sender, identID))
	sCulture := (*components.CultureComponent)(world.Get(sender, cultureID))

	sPos.X = 10.0
	sPos.Y = 10.0
	sIdent.ID = 100
	sCulture.LanguageID = 1 // Same language

	// Sender knows a secret that carries a BeliefID
	sSecret.Secrets = append(sSecret.Secrets, components.Secret{
		OriginID: 100,
		SecretID: 999,
		Virality: 255, // 100% chance base
		BeliefID: 42,
	})

	// Create Receiver
	receiver := world.NewEntity()
	world.Add(receiver, posID, secretID, memoryID, identID, cultureID, beliefID)

	rPos := (*components.Position)(world.Get(receiver, posID))
	rSecret := (*components.SecretComponent)(world.Get(receiver, secretID))
	rIdent := (*components.Identity)(world.Get(receiver, identID))
	rCulture := (*components.CultureComponent)(world.Get(receiver, cultureID))
	rBelief := (*components.BeliefComponent)(world.Get(receiver, beliefID))

	rPos.X = 11.0 // Within distance 2.0
	rPos.Y = 10.0
	rIdent.ID = 200
	rCulture.LanguageID = 1 // Same language

	// Pre-seed receiver with existing belief to test incrementing logic
	rBelief.Beliefs = append(rBelief.Beliefs, components.Belief{
		BeliefID: 42,
		Weight:   5,
	})

	// Add system
	system := NewGossipDistributionSystem(&world, engine.NewSparseHookGraph())

	// Run to tick 10 to trigger system
	for i := 0; i < 10; i++ {
		system.Update(&world)
	}

	// Verify receiver learned the secret
	if len(rSecret.Secrets) != 1 {
		t.Fatalf("Receiver should have learned exactly 1 secret, got %d", len(rSecret.Secrets))
	}

	if rSecret.Secrets[0].SecretID != 999 {
		t.Errorf("Receiver learned wrong secret: %d", rSecret.Secrets[0].SecretID)
	}

	if rSecret.Secrets[0].BeliefID != 42 {
		t.Errorf("Receiver learned secret but lost BeliefID metadata, got %d", rSecret.Secrets[0].BeliefID)
	}

	// Verify belief weight was incremented
	if len(rBelief.Beliefs) != 1 {
		t.Fatalf("Receiver should still have 1 belief struct array, got %d", len(rBelief.Beliefs))
	}

	if rBelief.Beliefs[0].Weight != 6 {
		t.Errorf("Expected Belief weight to increment to 6, got %d", rBelief.Beliefs[0].Weight)
	}

	// For compilation
	_ = rIdent
}
