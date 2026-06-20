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

	identID := ecs.ComponentID[components.Identity](&world)
	genomeID := ecs.ComponentID[components.GenomeComponent](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	busID := ecs.ComponentID[components.BusinessComponent](&world)
	busEntID := ecs.ComponentID[components.BusinessEntity](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// Create a business owner with low Intellect
	owner := world.NewEntity(identID, genomeID, npcID)

	ownerIdent := (*components.Identity)(world.Get(owner, identID))
	ownerIdent.ID = 1

	ownerGenome := (*components.GenomeComponent)(world.Get(owner, genomeID))
	ownerGenome.Intellect = 20

	// Create a business owned by the low-intellect owner
	business := world.NewEntity(busEntID, busID)
	busComp := (*components.BusinessComponent)(world.Get(business, busID))
	busComp.OwnerID = ownerIdent.ID

	// Create a desperate NPC with high Intellect
	forger := world.NewEntity(identID, genomeID, despID, memID, npcID)

	forgerIdent := (*components.Identity)(world.Get(forger, identID))
	forgerIdent.ID = 2

	forgerGenome := (*components.GenomeComponent)(world.Get(forger, genomeID))
	forgerGenome.Intellect = 80 // Higher than owner (20)

	forgerDesp := (*components.DesperationComponent)(world.Get(forger, despID))
	forgerDesp.Level = 50 // Desperate

	sys.Update(&world)

	// Verify ownership was transferred
	if busComp.OwnerID != forgerIdent.ID {
		t.Errorf("Expected Business to be owned by %d, but got %d", forgerIdent.ID, busComp.OwnerID)
	}

	// Verify hook was created
	hookStrength := hooks.GetHook(ownerIdent.ID, forgerIdent.ID)
	if hookStrength != -100 {
		t.Errorf("Expected hook strength of -100, but got %d", hookStrength)
	}

	// Verify memory event was created
	forgerMem := (*components.Memory)(world.Get(forger, memID))
	var eventFound bool
	for _, event := range forgerMem.Events {
		if event.InteractionType == components.InteractionTheft && event.TargetID == ownerIdent.ID {
			eventFound = true
			break
		}
	}

	if !eventFound {
		t.Errorf("Expected InteractionTheft memory event to be recorded for the forger, but it was not found")
	}
}
