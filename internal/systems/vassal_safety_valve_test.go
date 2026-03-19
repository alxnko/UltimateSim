package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// TestVassalSafetyValve_Integration verifies that Phase 44 bridges Wealth, Traits, and Hooks deterministically.
func TestVassalSafetyValve_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	engine.InitializeRNG([32]byte{1, 2, 3}) // Determinism

	// We must register Component IDs before adding entities
	npcID := ecs.ComponentID[components.NPC](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	secretID := ecs.ComponentID[components.SecretComponent](&world)

	sys := NewVassalSafetyValveSystem(&world, hooks)

	// Create Monopoly Clan (Clan 1) in City 1
	monopolyNPC := world.NewEntity(npcID, identID, affilID, needsID)

	mIdent := (*components.Identity)(world.Get(monopolyNPC, identID))
	mIdent.ID = 100
	mIdent.BaseTraits = 0 // No jealousy

	mAffil := (*components.Affiliation)(world.Get(monopolyNPC, affilID))
	mAffil.CityID = 1
	mAffil.ClanID = 1

	mNeeds := (*components.Needs)(world.Get(monopolyNPC, needsID))
	mNeeds.Wealth = 5000 // Massive Wealth (> 50% of 5050 and > 1000)

	// Create Jealous NPC (Clan 2) in City 1
	jealousNPC := world.NewEntity(npcID, identID, affilID, needsID, secretID)

	jIdent := (*components.Identity)(world.Get(jealousNPC, identID))
	jIdent.ID = 200
	jIdent.BaseTraits = components.TraitJealous // Has Jealousy Trait

	jAffil := (*components.Affiliation)(world.Get(jealousNPC, affilID))
	jAffil.CityID = 1
	jAffil.ClanID = 2

	jNeeds := (*components.Needs)(world.Get(jealousNPC, needsID))
	jNeeds.Wealth = 50 // Poor

	// Create normal NPC (Clan 3) without jealousy to verify it ignores them
	normalNPC := world.NewEntity(npcID, identID, affilID, needsID, secretID)

	nIdent := (*components.Identity)(world.Get(normalNPC, identID))
	nIdent.ID = 300
	nIdent.BaseTraits = 0 // No jealousy

	nAffil := (*components.Affiliation)(world.Get(normalNPC, affilID))
	nAffil.CityID = 1
	nAffil.ClanID = 3

	nNeeds := (*components.Needs)(world.Get(normalNPC, needsID))
	nNeeds.Wealth = 50 // Poor

	// Run system 499 times (nothing should happen, runs every 500 ticks)
	for i := 0; i < 499; i++ {
		sys.Update(&world)
	}

	// Verify nothing happened yet
	if hooks.GetHook(200, 100) == -50 {
		t.Fatalf("Expected Jealous NPC not to have grudge before 500 ticks")
	}

	// 500th tick triggers the logic
	sys.Update(&world)

	// 1. Verify Grudge (Justice/Blood Feud Bridge)
	jHook := hooks.GetHook(200, 100)
	if jHook != -50 {
		t.Fatalf("Expected Jealous NPC to generate a -50 grudge against Monopoly Ruler, got %d", jHook)
	}

	nHook := hooks.GetHook(300, 100)
	if nHook == -50 {
		t.Fatalf("Expected Normal NPC NOT to generate a grudge, got %d", nHook)
	}

	// 2. Verify Secret Generation (Memetic Bridge)
	jSecret := (*components.SecretComponent)(world.Get(jealousNPC, secretID))
	if len(jSecret.Secrets) == 0 {
		t.Fatalf("Expected Jealous NPC to generate a negative secret against Monopoly Ruler")
	}

	secret := jSecret.Secrets[0]
	if secret.OriginID != 200 {
		t.Errorf("Expected secret origin ID to be Jealous NPC (200), got %d", secret.OriginID)
	}
	if secret.Virality != 255 {
		t.Errorf("Expected secret virality to be 255, got %d", secret.Virality)
	}

	// Check registry for correct string
	registry := engine.GetSecretRegistry()
	text, exists := registry.GetSecret(secret.SecretID)
	if !exists {
		t.Fatalf("Expected secret to exist in registry")
	}
	expectedText := "monopoly_resentment_against_100_tick_500"
	if text != expectedText {
		t.Errorf("Expected secret text '%s', got '%s'", expectedText, text)
	}

	// 3. Verify Determinism
	// Second run after another 500 ticks should not crash and should respect identical states
	// We will run it another 500 ticks. The grudge is already -50, so it shouldn't go below -50 (based on implementation logic)
	for i := 0; i < 500; i++ {
		sys.Update(&world)
	}

	jHook2 := hooks.GetHook(200, 100)
	if jHook2 < -50 {
		t.Fatalf("Expected hook to cap at -50 to prevent infinite stack overflow, got %d", jHook2)
	}
}
