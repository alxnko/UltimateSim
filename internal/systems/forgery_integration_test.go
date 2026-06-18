package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 70: Deed Forgery Integration Test
// Butterfly Effect: Economy (Desperation) + Genetics (Intellect) -> Justice (Theft/Forgery) -> Blood Feud (Murder)

func TestForgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	forgerySys := NewForgerySystem(&world, hookGraph)
	feudSys := NewBloodFeudSystem(&world, hookGraph)

	// Component IDs
	npcID := ecs.ComponentID[components.NPC](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	posID := ecs.ComponentID[components.Position](&world)

	busTagID := ecs.ComponentID[components.BusinessEntity](&world)
	busCompID := ecs.ComponentID[components.BusinessComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)

	// 1. Create the Victim (Low Intellect Business Owner)
	victimEnt := world.NewEntity(npcID, idID, genID, despID, memID, needsID, affID, posID)

	vPos := (*components.Position)(world.Get(victimEnt, posID))
	vPos.X = 10.0
	vPos.Y = 10.0

	vIdent := (*components.Identity)(world.Get(victimEnt, idID))
	vIdent.ID = 101
	vIdent.Name = "Gullible Bob"

	vGen := (*components.GenomeComponent)(world.Get(victimEnt, genID))
	vGen.Intellect = 20 // Low intellect

	vNeeds := (*components.Needs)(world.Get(victimEnt, needsID))
	vNeeds.Food = 100 // Healthy and fed

	vAff := (*components.Affiliation)(world.Get(victimEnt, affID))
	vAff.FamilyID = 10 // Need FamilyID for BloodFeud

	// 2. Create the Business owned by the Victim
	busEnt := world.NewEntity(busTagID, busCompID, treasuryID)

	bComp := (*components.BusinessComponent)(world.Get(busEnt, busCompID))
	bComp.OwnerID = 101

	// 3. Create the Forger (High Intellect, Desperate)
	forgerEnt := world.NewEntity(npcID, idID, genID, despID, memID, needsID, affID, posID)

	fPos := (*components.Position)(world.Get(forgerEnt, posID))
	fPos.X = 10.0
	fPos.Y = 10.0

	fIdent := (*components.Identity)(world.Get(forgerEnt, idID))
	fIdent.ID = 202
	fIdent.Name = "Slick Rick"

	fGen := (*components.GenomeComponent)(world.Get(forgerEnt, genID))
	fGen.Intellect = 80 // High intellect

	fDesp := (*components.DesperationComponent)(world.Get(forgerEnt, despID))
	fDesp.Level = 90 // Highly desperate

	fNeeds := (*components.Needs)(world.Get(forgerEnt, needsID))
	fNeeds.Food = 100 // Healthy and fed

	fAff := (*components.Affiliation)(world.Get(forgerEnt, affID))
	fAff.FamilyID = 20 // Different family

	// -- PRE-CHECK --
	if bComp.OwnerID != 101 {
		t.Fatalf("Business should start owned by victim (101)")
	}
	if hookGraph.GetHook(101, 202) != 0 {
		t.Fatalf("Hook should start at 0")
	}

	// -- TICK 1: FORGERY EXECUTES --
	forgerySys.Update(&world)

	// Verify Forgery Results
	bCompAfter := (*components.BusinessComponent)(world.Get(busEnt, busCompID))
	if bCompAfter.OwnerID != 202 {
		t.Errorf("Expected business to be stolen by forger (202), got %d", bCompAfter.OwnerID)
	}

	hookStrength := hookGraph.GetHook(101, 202)
	if hookStrength != -100 {
		t.Errorf("Expected victim to have -100 hook against forger, got %d", hookStrength)
	}

	fMem := (*components.Memory)(world.Get(forgerEnt, memID))
	foundTheft := false
	for i := 0; i < len(fMem.Events); i++ {
		if fMem.Events[i].InteractionType == components.InteractionTheft && fMem.Events[i].TargetID == 101 {
			foundTheft = true
			break
		}
	}
	if !foundTheft {
		t.Errorf("Expected InteractionTheft (4) in forger's memory")
	}

	// -- TICK 2: BLOOD FEUD (BUTTERFLY EFFECT) --
	// The BloodFeudSystem should detect the deep negative hook and assign CombatMarker
	feudSys.Update(&world)

	combatID := ecs.ComponentID[components.CombatMarker](&world)
	if !world.Has(victimEnt, combatID) {
		t.Errorf("Expected BloodFeudSystem to assign CombatMarker to victim")
	} else {
		combatMarker := (*components.CombatMarker)(world.Get(victimEnt, combatID))
		if combatMarker.TargetID != 202 {
			t.Errorf("Expected CombatMarker TargetID to be 202, got %d", combatMarker.TargetID)
		}
	}
}
