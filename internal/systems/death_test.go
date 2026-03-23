package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: DeathSystem Tests

// TestDeathSystem_E2E validates "Test First" logic:
// If NPC Food reaches 0, the entity must be despawned
func TestDeathSystem_E2E(t *testing.T) {
	world := ecs.NewWorld()
	needsID := ecs.ComponentID[components.Needs](&world)

	hooks := engine.NewSparseHookGraph()
	deathSystem := systems.NewDeathSystem(&world, hooks)

	// Entity 1: Has Food
	e1 := world.NewEntity(needsID)
	n1 := (*components.Needs)(world.Get(e1, needsID))
	n1.Food = 10.0

	// Entity 2: Starving
	e2 := world.NewEntity(needsID)
	n2 := (*components.Needs)(world.Get(e2, needsID))
	n2.Food = 0.0

	// Entity 3: Already dead mathematically (negative)
	e3 := world.NewEntity(needsID)
	n3 := (*components.Needs)(world.Get(e3, needsID))
	n3.Food = -5.0

	deathSystem.Update(&world)

	// Verify
	if !world.Alive(e1) {
		t.Errorf("Expected Entity 1 (Food > 0) to be alive")
	}
	if world.Alive(e2) {
		t.Errorf("Expected Entity 2 (Food = 0) to be dead")
	}
	if world.Alive(e3) {
		t.Errorf("Expected Entity 3 (Food < 0) to be dead")
	}
}

// Phase 25.2: Ideological Succession Engine Integration Test
func TestDeathSystem_BeliefInheritance(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{10, 11, 12})

	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()
	sys := systems.NewDeathSystem(&world, hookGraph)

	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)

	// Create dying parent
	parent := world.NewEntity()
	world.Add(parent, posID, needsID, idID, affilID, legacyID, npcID, beliefID)
	pNeeds := (*components.Needs)(world.Get(parent, needsID))
	pNeeds.Food = 0 // Will die
	pIdent := (*components.Identity)(world.Get(parent, idID))
	pIdent.ID = 100
	pAff := (*components.Affiliation)(world.Get(parent, affilID))
	pAff.FamilyID = 1
	pBelief := (*components.BeliefComponent)(world.Get(parent, beliefID))
	pBelief.Beliefs = append(pBelief.Beliefs, components.Belief{BeliefID: 123, Weight: 50}) // Strong belief
	pBelief.Beliefs = append(pBelief.Beliefs, components.Belief{BeliefID: 456, Weight: 5})  // Weak belief, should not inherit

	// Create heir
	heir := world.NewEntity()
	world.Add(heir, posID, needsID, idID, affilID, legacyID, npcID) // Heir does NOT have belief component initially
	hNeeds := (*components.Needs)(world.Get(heir, needsID))
	hNeeds.Food = 100 // Survives
	hIdent := (*components.Identity)(world.Get(heir, idID))
	hIdent.ID = 200
	hAff := (*components.Affiliation)(world.Get(heir, affilID))
	hAff.FamilyID = 1 // Same family

	// Trigger succession
	sys.Update(&world)

	if world.Alive(parent) {
		t.Errorf("Parent should be dead")
	}

	if !world.Alive(heir) {
		t.Errorf("Heir should be alive")
	}

	// Verify belief inheritance
	if !world.Has(heir, beliefID) {
		t.Fatalf("Heir should have acquired the BeliefComponent")
	}

	hBelief := (*components.BeliefComponent)(world.Get(heir, beliefID))
	if len(hBelief.Beliefs) != 1 {
		t.Fatalf("Heir should have inherited exactly 1 belief, got %d", len(hBelief.Beliefs))
	}

	if hBelief.Beliefs[0].BeliefID != 123 {
		t.Errorf("Expected inherited BeliefID to be 123, got %d", hBelief.Beliefs[0].BeliefID)
	}

	if hBelief.Beliefs[0].Weight != 25 { // 50 / 2
		t.Errorf("Expected inherited Belief Weight to be 25, got %d", hBelief.Beliefs[0].Weight)
	}
}

// Evolution: Phase 25.2 - The Ideological Succession Engine
func TestIdeologicalSuccession_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()
	sys := systems.NewDeathSystem(&world, hookGraph)

	// Register components
	npcID := ecs.ComponentID[components.NPC](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)

	// Father (dying)
	father := world.NewEntity(npcID, needsID, legacyID, identID, affilID, beliefID)
	fatherIdent := (*components.Identity)(world.Get(father, identID))
	fatherIdent.ID = 1

	fatherNeeds := (*components.Needs)(world.Get(father, needsID))
	fatherNeeds.Food = 0 // Starving

	fatherAffil := (*components.Affiliation)(world.Get(father, affilID))
	fatherAffil.FamilyID = 100

	fatherBelief := (*components.BeliefComponent)(world.Get(father, beliefID))
	fatherBelief.Beliefs = append(fatherBelief.Beliefs, components.Belief{
		BeliefID: 42,
		Weight:   100,
	})

	// Son (heir)
	son := world.NewEntity(npcID, needsID, legacyID, identID, affilID, beliefID)
	sonIdent := (*components.Identity)(world.Get(son, identID))
	sonIdent.ID = 2

	sonNeeds := (*components.Needs)(world.Get(son, needsID))
	sonNeeds.Food = 50 // Alive

	sonAffil := (*components.Affiliation)(world.Get(son, affilID))
	sonAffil.FamilyID = 100

	// Trigger DeathSystem
	sys.Update(&world)

	// Verify father despawned
	if world.Alive(father) {
		t.Errorf("Expected father to despawn, but he is still alive")
	}

	// Verify son received ideological succession with weight decay
	if !world.Alive(son) {
		t.Fatalf("Expected son to be alive, but he despawned")
	}

	sonBelief := (*components.BeliefComponent)(world.Get(son, beliefID))
	if len(sonBelief.Beliefs) == 0 {
		t.Fatalf("Expected son to inherit beliefs, but Beliefs array is empty")
	}

	found := false
	for _, b := range sonBelief.Beliefs {
		if b.BeliefID == 42 {
			found = true
			if b.Weight != 50 {
				t.Errorf("Expected inherited belief weight to be 50, got %d", b.Weight)
			}
		}
	}

	if !found {
		t.Errorf("Expected son to inherit BeliefID 42, but it was not found")
	}
}
