package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestForgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	sys := NewForgerySystem(&world, hooks)

	npcID := ecs.ComponentID[components.NPC](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	busID := ecs.ComponentID[components.BusinessComponent](&world)

	// Create Owner Entity (Low Intellect)
	ownerEnt := world.NewEntity(npcID, identID, genID)
	ownerIdent := (*components.Identity)(world.Get(ownerEnt, identID))
	ownerIdent.ID = 101
	ownerIdent.Name = "Rich Owner"

	ownerGen := (*components.GenomeComponent)(world.Get(ownerEnt, genID))
	ownerGen.Intellect = 50

	// Create Business Entity owned by Owner
	busEnt := world.NewEntity(busID)
	busComp := (*components.BusinessComponent)(world.Get(busEnt, busID))
	busComp.OwnerID = 101

	// Create Forger Entity (High Intellect, Desperate)
	forgerEnt := world.NewEntity(npcID, despID, genID, memID, identID)
	forgerIdent := (*components.Identity)(world.Get(forgerEnt, identID))
	forgerIdent.ID = 202
	forgerIdent.Name = "Clever Forger"

	forgerGen := (*components.GenomeComponent)(world.Get(forgerEnt, genID))
	forgerGen.Intellect = 90 // Greater than 50

	// We don't need to set Desperation level because the filter just looks for component

	// Run system
	sys.Update(&world)

	// Assert ownership transferred
	if busComp.OwnerID != 202 {
		t.Errorf("Expected business to be stolen by forger (202), but owner is %d", busComp.OwnerID)
	}

	// Assert negative hook generated (OriginalOwner -> Forger)
	hookStrength := hooks.GetHook(101, 202)
	if hookStrength != -100 {
		t.Errorf("Expected hook strength -100, got %d", hookStrength)
	}

	// Assert Memory logged InteractionTheft
	forgerMem := (*components.Memory)(world.Get(forgerEnt, memID))
	found := false
	for _, event := range forgerMem.Events {
		if event.TargetID == 101 && event.InteractionType == components.InteractionTheft {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected InteractionTheft memory event to be logged for forger")
	}
}
