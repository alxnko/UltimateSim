package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestForgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// Add component types
	npcID := ecs.ComponentID[components.NPC](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	busTagID := ecs.ComponentID[components.BusinessEntity](&world)
	busCompID := ecs.ComponentID[components.BusinessComponent](&world)
	workID := ecs.ComponentID[components.WorkplaceComponent](&world)

	// Create a wealthy, low-intellect Owner
	ownerEnt := world.NewEntity(npcID, identID, genID)
	ownerIdent := (*components.Identity)(world.Get(ownerEnt, identID))
	ownerIdent.ID = 101
	ownerGen := (*components.GenomeComponent)(world.Get(ownerEnt, genID))
	ownerGen.Intellect = 80

	// Create a Business owned by the low-intellect Owner
	busEnt := world.NewEntity(busTagID, busCompID, workID)
	busComp := (*components.BusinessComponent)(world.Get(busEnt, busCompID))
	busComp.OwnerID = 101
	busWork := (*components.WorkplaceComponent)(world.Get(busEnt, workID))
	busWork.X = 10.0
	busWork.Y = 10.0

	// Create a desperate, high-intellect Forger physically near the Business
	forgerEnt := world.NewEntity(npcID, identID, genID, posID, despID, memID)
	forgerIdent := (*components.Identity)(world.Get(forgerEnt, identID))
	forgerIdent.ID = 999
	forgerGen := (*components.GenomeComponent)(world.Get(forgerEnt, genID))
	forgerGen.Intellect = 120 // 120 > 80 + 20
	forgerPos := (*components.Position)(world.Get(forgerEnt, posID))
	forgerPos.X = 11.0
	forgerPos.Y = 11.0 // distSq = 1^2 + 1^2 = 2.0 <= 4.0
	forgerDesp := (*components.DesperationComponent)(world.Get(forgerEnt, despID))
	forgerDesp.Level = 80 // > 50

	sys := NewForgerySystem(&world, hookGraph)

	// tickCounter % 20 != 0 normally, but let's just loop until it hits
	for i := 0; i < 20; i++ {
		sys.Update(&world)
	}

	// Assertions

	// 1. Ownership transferred
	busCompAfter := (*components.BusinessComponent)(world.Get(busEnt, busCompID))
	if busCompAfter.OwnerID != 999 {
		t.Fatalf("Expected Business to be owned by Forger (999), but was %d", busCompAfter.OwnerID)
	}

	// 2. SparseHookGraph has -100 grudge from owner to forger
	hookVal := hookGraph.GetHook(101, 999)
	if hookVal != -100 {
		t.Fatalf("Expected hook value to be -100, got %d", hookVal)
	}

	// 3. Forger's memory contains InteractionTheft
	forgerMem := (*components.Memory)(world.Get(forgerEnt, memID))
	foundTheft := false
	for _, event := range forgerMem.Events {
		if event.TargetID == 101 && event.InteractionType == components.InteractionTheft {
			foundTheft = true
			break
		}
	}
	if !foundTheft {
		t.Fatalf("Expected Forger to have InteractionTheft memory against 101")
	}
}
