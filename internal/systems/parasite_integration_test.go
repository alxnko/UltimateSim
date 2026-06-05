package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 69: The Parasitic Symbiosis Engine Integration Test
func TestParasiteSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components explicitly for Arche-Go determinism
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	parasiteID := ecs.ComponentID[components.ParasiteComponent](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	hooks := engine.NewSparseHookGraph()
	sys := NewParasiteSystem(&world, hooks)

	// Create Parasite
	pEnt := world.NewEntity(posID, needsID, identID, parasiteID)
	pPos := (*components.Position)(world.Get(pEnt, posID))
	pPos.X = 10.0
	pPos.Y = 10.0

	pNeeds := (*components.Needs)(world.Get(pEnt, needsID))
	pNeeds.Food = 10.0 // Hungry vampire

	pIdent := (*components.Identity)(world.Get(pEnt, identID))
	pIdent.ID = 1
	pIdent.BaseTraits = 0

	pComp := (*components.ParasiteComponent)(world.Get(pEnt, parasiteID))
	pComp.BloodSatiety = 0
	pComp.IsHidden = true

	// Create Victim
	vEnt := world.NewEntity(posID, vitalsID, identID, npcID, memID)
	vPos := (*components.Position)(world.Get(vEnt, posID))
	vPos.X = 11.0
	vPos.Y = 11.0 // Within distSq <= 4.0

	vVitals := (*components.VitalsComponent)(world.Get(vEnt, vitalsID))
	vVitals.Blood = 100.0
	vVitals.Pain = 0.0

	vIdent := (*components.Identity)(world.Get(vEnt, identID))
	vIdent.ID = 2

	// Update System
	sys.Update(&world)

	// Verify the Attack
	if pNeeds.Food <= 10.0 {
		t.Fatalf("Expected parasite to feed and restore Needs.Food, got %f", pNeeds.Food)
	}
	if pComp.BloodSatiety != 20.0 {
		t.Errorf("Expected parasite BloodSatiety to increase to 20, got %f", pComp.BloodSatiety)
	}
	if pComp.IsHidden {
		t.Errorf("Expected parasite to be exposed (IsHidden = false)")
	}
	if pIdent.BaseTraits&components.TraitEsoteric == 0 {
		t.Errorf("Expected parasite to gain TraitEsoteric")
	}

	if vVitals.Blood != 80.0 {
		t.Errorf("Expected victim blood to drain to 80, got %f", vVitals.Blood)
	}
	if vVitals.Pain != 20.0 {
		t.Errorf("Expected victim pain to increase to 20, got %f", vVitals.Pain)
	}

	// Verify Hooks
	hookVal := hooks.GetHook(vIdent.ID, pIdent.ID)
	if hookVal != -50 {
		t.Errorf("Expected hook of -50 from victim to parasite, got %d", hookVal)
	}

	// Verify Memory
	vMem := (*components.Memory)(world.Get(vEnt, memID))
	foundEvent := false
	for _, e := range vMem.Events {
		if e.InteractionType == components.InteractionParasite && e.TargetID == pIdent.ID {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Errorf("Expected memory event of InteractionParasite from victim to parasite")
	}
}
