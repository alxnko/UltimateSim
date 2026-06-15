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

	// 1. Create the Owner (low intellect, not desperate)
	owner := world.NewEntity()
	world.Add(owner,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)

	ownerIdent := (*components.Identity)(world.Get(owner, ecs.ComponentID[components.Identity](&world)))
	ownerIdent.ID = 100

	ownerGenome := (*components.GenomeComponent)(world.Get(owner, ecs.ComponentID[components.GenomeComponent](&world)))
	ownerGenome.Intellect = 40 // Low intellect

	// 2. Create the Business owned by the Owner
	biz := world.NewEntity()
	world.Add(biz,
		ecs.ComponentID[components.BusinessEntity](&world),
		ecs.ComponentID[components.BusinessComponent](&world),
	)

	bizComp := (*components.BusinessComponent)(world.Get(biz, ecs.ComponentID[components.BusinessComponent](&world)))
	bizComp.OwnerID = ownerIdent.ID

	// 3. Create the Forger (high intellect, desperate)
	forger := world.NewEntity()
	world.Add(forger,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)

	forgerIdent := (*components.Identity)(world.Get(forger, ecs.ComponentID[components.Identity](&world)))
	forgerIdent.ID = 200

	forgerGenome := (*components.GenomeComponent)(world.Get(forger, ecs.ComponentID[components.GenomeComponent](&world)))
	forgerGenome.Intellect = 80 // High intellect

	forgerDesp := (*components.DesperationComponent)(world.Get(forger, ecs.ComponentID[components.DesperationComponent](&world)))
	forgerDesp.Level = 60 // Desperate (> 50)

	// Pre-test assertions
	if bizComp.OwnerID != 100 {
		t.Fatalf("Expected owner ID to be 100 before update, got %d", bizComp.OwnerID)
	}

	// 4. Run the system
	sys.Update(&world)

	// Post-test assertions

	// A. Check ownership transfer
	if bizComp.OwnerID != 200 {
		t.Errorf("Expected Business OwnerID to be transferred to 200, got %d", bizComp.OwnerID)
	}

	// B. Check Hook generation
	hookValue := hooks.GetHook(100, 200) // Hook from Owner to Forger
	if hookValue != -100 {
		t.Errorf("Expected -100 hook from owner to forger, got %d", hookValue)
	}

	// C. Check Memory Event
	forgerMem := (*components.Memory)(world.Get(forger, ecs.ComponentID[components.Memory](&world)))

	// The event should be at index 0 because it's the first one added and Head was 0
	// Head should now be 1
	if forgerMem.Head != 1 {
		t.Errorf("Expected Memory Head to be 1, got %d", forgerMem.Head)
	}

	lastEvent := forgerMem.Events[0]
	if lastEvent.TargetID != 100 {
		t.Errorf("Expected memory event TargetID to be 100, got %d", lastEvent.TargetID)
	}
	if lastEvent.InteractionType != components.InteractionTheft {
		t.Errorf("Expected memory event InteractionType to be InteractionTheft (%d), got %d", components.InteractionTheft, lastEvent.InteractionType)
	}
}
