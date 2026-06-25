package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: heir & possession transfer tests.
// Deterministic, headless coverage of FindHeirs (family filtering + age-DESC sort),
// PossessEntity (tag transfer + Velocity guarantee), FindStartCandidate (deterministic
// young family-bound pick), and DeathSystem's player-death surfacing on the possessed body.

// spawnHeirNPC creates an NPC+Identity+Affiliation entity with the given identity ID,
// family ID, and age.
func spawnHeirNPC(world *ecs.World, identityID uint64, familyID uint32, age uint16) ecs.Entity {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)

	e := world.NewEntity(npcID, identID, affilID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = identityID
	ident.Age = age
	affil := (*components.Affiliation)(world.Get(e, affilID))
	affil.FamilyID = familyID
	return e
}

// TestFindHeirs verifies family filtering, dead-player exclusion, and eldest-first ordering.
func TestFindHeirs(t *testing.T) {
	world := ecs.NewWorld()

	// Family 5 members: ages 40 / 20 / 30 with identity IDs 1 / 2 / 3.
	// ID 1 (the eldest, age 40) is the dead player and must be excluded.
	deadID := uint64(1)
	spawnHeirNPC(&world, 1, 5, 40) // excluded (dead)
	spawnHeirNPC(&world, 2, 5, 20)
	spawnHeirNPC(&world, 3, 5, 30)
	// An unrelated family-9 member must never appear.
	spawnHeirNPC(&world, 4, 9, 50)

	heirs := FindHeirs(&world, 5, deadID)

	if len(heirs) != 2 {
		t.Fatalf("FindHeirs returned %d heirs, want 2 (family-5 minus the dead player)", len(heirs))
	}

	// Eldest-first: age 30 (ID 3) then age 20 (ID 2).
	if heirs[0].ID != 3 || heirs[0].Age != 30 {
		t.Errorf("heirs[0] = (ID %d, Age %d), want (ID 3, Age 30)", heirs[0].ID, heirs[0].Age)
	}
	if heirs[1].ID != 2 || heirs[1].Age != 20 {
		t.Errorf("heirs[1] = (ID %d, Age %d), want (ID 2, Age 20)", heirs[1].ID, heirs[1].Age)
	}

	// No family-9 or excluded member leaked through.
	for _, h := range heirs {
		if h.ID == deadID {
			t.Errorf("dead player (ID %d) must be excluded from heirs", deadID)
		}
		if h.ID == 4 {
			t.Errorf("family-9 member (ID 4) must not appear among family-5 heirs")
		}
	}

	// Querying a family with no living members returns empty.
	if got := FindHeirs(&world, 99, 0); len(got) != 0 {
		t.Errorf("FindHeirs for empty family returned %d heirs, want 0", len(got))
	}
}

// TestPossessEntity verifies the Possessed tag transfers off the current holder onto the
// target and that the target is guaranteed a Velocity component for movement.
func TestPossessEntity(t *testing.T) {
	world := ecs.NewWorld()

	possessedID := ecs.ComponentID[components.Possessed](&world)
	velID := ecs.ComponentID[components.Velocity](&world)

	// Current holder starts Possessed.
	holder := spawnHeirNPC(&world, 10, 5, 35)
	world.Add(holder, possessedID)

	// Target has neither Possessed nor Velocity yet.
	target := spawnHeirNPC(&world, 11, 5, 28)

	if err := PossessEntity(&world, target); err != nil {
		t.Fatalf("PossessEntity returned error: %v", err)
	}

	if world.Has(holder, possessedID) {
		t.Errorf("previous holder should have lost the Possessed tag")
	}
	if !world.Has(target, possessedID) {
		t.Errorf("target should have gained the Possessed tag")
	}
	if !world.Has(target, velID) {
		t.Errorf("target should have a Velocity component after possession")
	}

	// Idempotent re-possession of the same target keeps the tag and Velocity.
	if err := PossessEntity(&world, target); err != nil {
		t.Fatalf("re-possessing the same target returned error: %v", err)
	}
	if !world.Has(target, possessedID) || !world.Has(target, velID) {
		t.Errorf("re-possessing target should retain Possessed and Velocity")
	}
}

// TestFindStartCandidate verifies a deterministic young, family-bound NPC is chosen:
// FamilyID != 0, Age in [16,40], lowest Identity.ID wins ties.
func TestFindStartCandidate(t *testing.T) {
	world := ecs.NewWorld()

	// Ineligible: no family.
	spawnHeirNPC(&world, 1, 0, 25)
	// Ineligible: too young.
	spawnHeirNPC(&world, 2, 5, 15)
	// Ineligible: too old.
	spawnHeirNPC(&world, 3, 5, 41)
	// Eligible candidates (FamilyID != 0, Age in [16,40]).
	spawnHeirNPC(&world, 7, 5, 16) // lowest eligible ID -> expected winner (boundary age 16)
	spawnHeirNPC(&world, 8, 9, 40) // boundary age 40, higher ID
	spawnHeirNPC(&world, 9, 5, 30)

	ent, found := FindStartCandidate(&world)
	if !found {
		t.Fatalf("FindStartCandidate found no candidate, want one")
	}

	identID := ecs.ComponentID[components.Identity](&world)
	ident := (*components.Identity)(world.Get(ent, identID))
	if ident.ID != 7 {
		t.Errorf("FindStartCandidate picked ID %d, want 7 (lowest eligible ID)", ident.ID)
	}

	// With no eligible NPCs, found is false.
	empty := ecs.NewWorld()
	spawnHeirNPC(&empty, 1, 0, 25) // homeless -> ineligible
	if _, ok := FindStartCandidate(&empty); ok {
		t.Errorf("FindStartCandidate should report not-found when no eligible NPCs exist")
	}
}

// TestDeathSystemPossessedSurfacesPlayerDeath verifies the DeathSystem surfaces the
// death of the player-possessed entity (starving) via PlayerEvents, and that a
// non-possessed starving NPC does NOT raise PlayerDied.
func TestDeathSystemPossessedSurfacesPlayerDeath(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	possessedID := ecs.ComponentID[components.Possessed](&world)

	ds := NewDeathSystem(&world, hooks)
	pe := &engine.PlayerEvents{}
	ds.SetPlayerEvents(pe)

	// Possessed NPC starving (Food == 0) with Identity + Affiliation.
	player := world.NewEntity(needsID, identID, affilID, possessedID)
	pNeeds := (*components.Needs)(world.Get(player, needsID))
	pNeeds.Food = 0
	pIdent := (*components.Identity)(world.Get(player, identID))
	pIdent.ID = 42
	pAffil := (*components.Affiliation)(world.Get(player, affilID))
	pAffil.FamilyID = 7

	ds.Update(&world)

	if !pe.PlayerDied {
		t.Fatalf("PlayerDied should be true after the possessed entity starved")
	}
	if pe.DeathCause != engine.DeathCauseStarvation {
		t.Errorf("DeathCause = %d, want DeathCauseStarvation (%d)", pe.DeathCause, engine.DeathCauseStarvation)
	}
	if pe.DeadID != 42 {
		t.Errorf("DeadID = %d, want 42", pe.DeadID)
	}
	if pe.DeadFamilyID != 7 {
		t.Errorf("DeadFamilyID = %d, want 7", pe.DeadFamilyID)
	}

	// A non-possessed starving NPC, alone, must NOT raise PlayerDied (fresh events sink).
	world2 := ecs.NewWorld()
	hooks2 := engine.NewSparseHookGraph()

	needsID2 := ecs.ComponentID[components.Needs](&world2)
	identID2 := ecs.ComponentID[components.Identity](&world2)
	affilID2 := ecs.ComponentID[components.Affiliation](&world2)
	// Register Possessed so it exists in the world even though no entity holds it.
	ecs.ComponentID[components.Possessed](&world2)

	ds2 := NewDeathSystem(&world2, hooks2)
	pe2 := &engine.PlayerEvents{}
	ds2.SetPlayerEvents(pe2)

	npc := world2.NewEntity(needsID2, identID2, affilID2)
	nNeeds := (*components.Needs)(world2.Get(npc, needsID2))
	nNeeds.Food = 0
	nIdent := (*components.Identity)(world2.Get(npc, identID2))
	nIdent.ID = 99

	ds2.Update(&world2)

	if pe2.PlayerDied {
		t.Errorf("PlayerDied should remain false when a non-possessed NPC starves")
	}
	if pe2.DeathCause != 0 {
		t.Errorf("DeathCause = %d, want 0 (untouched) for non-possessed death", pe2.DeathCause)
	}
}
