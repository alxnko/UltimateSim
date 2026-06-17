package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestForgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register components
	ecs.ComponentID[components.DesperationComponent](&world)
	ecs.ComponentID[components.GenomeComponent](&world)
	ecs.ComponentID[components.Memory](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.BusinessComponent](&world)
	ecs.ComponentID[components.BusinessEntity](&world)

	hooks := engine.NewSparseHookGraph()

	// Create a low-intellect owner
	owner := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
	)

	ownerIdent := (*components.Identity)(world.Get(owner, ecs.ComponentID[components.Identity](&world)))
	ownerIdent.ID = 101

	ownerGenome := (*components.GenomeComponent)(world.Get(owner, ecs.ComponentID[components.GenomeComponent](&world)))
	ownerGenome.Intellect = 20

	// Create the target business owned by the low-intellect owner
	biz := world.NewEntity(
		ecs.ComponentID[components.BusinessEntity](&world),
		ecs.ComponentID[components.BusinessComponent](&world),
	)
	bizComp := (*components.BusinessComponent)(world.Get(biz, ecs.ComponentID[components.BusinessComponent](&world)))
	bizComp.OwnerID = ownerIdent.ID

	// Create a high-intellect desperate forger
	forger := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)

	forgerIdent := (*components.Identity)(world.Get(forger, ecs.ComponentID[components.Identity](&world)))
	forgerIdent.ID = 202

	forgerGenome := (*components.GenomeComponent)(world.Get(forger, ecs.ComponentID[components.GenomeComponent](&world)))
	forgerGenome.Intellect = 90

	forgerDesp := (*components.DesperationComponent)(world.Get(forger, ecs.ComponentID[components.DesperationComponent](&world)))
	forgerDesp.Level = 50 // Desperate

	sys := NewForgerySystem(&world, hooks)

	// Run the system
	sys.Update(&world)

	// Validate Ownership change
	updatedBizComp := (*components.BusinessComponent)(world.Get(biz, ecs.ComponentID[components.BusinessComponent](&world)))
	if updatedBizComp.OwnerID != forgerIdent.ID {
		t.Errorf("Expected Business OwnerID to be updated to forger ID %d, got %d", forgerIdent.ID, updatedBizComp.OwnerID)
	}

	// Validate SparseHookGraph penalty
	hookVal := hooks.GetHook(ownerIdent.ID, forgerIdent.ID)
	if hookVal != -100 {
		t.Errorf("Expected -100 hook from owner to forger, got %d", hookVal)
	}

	// Validate Memory event
	mem := (*components.Memory)(world.Get(forger, ecs.ComponentID[components.Memory](&world)))
	foundTheft := false
	for _, event := range mem.Events {
		if event.InteractionType == components.InteractionTheft && event.TargetID == ownerIdent.ID {
			foundTheft = true
			break
		}
	}
	if !foundTheft {
		t.Errorf("Expected InteractionTheft recorded in forger's Memory against target %d", ownerIdent.ID)
	}
}
