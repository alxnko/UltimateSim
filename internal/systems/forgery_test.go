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

	// Create Original Owner
	ownerEnt := world.NewEntity()
	world.Add(ownerEnt,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)

	ownerIdent := (*components.Identity)(world.Get(ownerEnt, sys.idID))
	ownerIdent.ID = 100

	ownerGenome := (*components.GenomeComponent)(world.Get(ownerEnt, sys.genomeID))
	ownerGenome.Intellect = 50 // Average Intellect

	ownerDesp := (*components.DesperationComponent)(world.Get(ownerEnt, sys.despID))
	ownerDesp.Level = 0 // Not desperate

	// Create Business
	busEnt := world.NewEntity()
	world.Add(busEnt,
		ecs.ComponentID[components.BusinessEntity](&world),
		ecs.ComponentID[components.BusinessComponent](&world),
	)

	busComp := (*components.BusinessComponent)(world.Get(busEnt, sys.busCompID))
	busComp.OwnerID = ownerIdent.ID

	// Create Forger
	forgerEnt := world.NewEntity()
	world.Add(forgerEnt,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)

	forgerIdent := (*components.Identity)(world.Get(forgerEnt, sys.idID))
	forgerIdent.ID = 200

	forgerGenome := (*components.GenomeComponent)(world.Get(forgerEnt, sys.genomeID))
	forgerGenome.Intellect = 90 // High Intellect

	forgerDesp := (*components.DesperationComponent)(world.Get(forgerEnt, sys.despID))
	forgerDesp.Level = 80 // Very desperate

	// Step simulation to tick 71
	for i := 0; i < 71; i++ {
		sys.Update(&world)
	}

	// 1. Check Ownership Transfer
	updatedBus := (*components.BusinessComponent)(world.Get(busEnt, sys.busCompID))
	if updatedBus.OwnerID != forgerIdent.ID {
		t.Errorf("Expected business to be transferred to forger (%d), got %d", forgerIdent.ID, updatedBus.OwnerID)
	}

	// 2. Check Negative Hook Generation
	hook := hooks.GetHook(ownerIdent.ID, forgerIdent.ID)
	if hook != -100 {
		t.Errorf("Expected negative hook of -100 against the forger, got %d", hook)
	}

	// 3. Check Memory InteractionTheft
	forgerMem := (*components.Memory)(world.Get(forgerEnt, sys.memID))
	var foundTheft bool
	for _, event := range forgerMem.Events {
		if event.InteractionType == 4 && event.TargetID == ownerIdent.ID && event.Value == -100 {
			foundTheft = true
			break
		}
	}

	if !foundTheft {
		t.Errorf("Expected to find InteractionTheft event in the forger's memory")
	}
}

func TestForgerySystem_Determinism(t *testing.T) {
	hooks1 := engine.NewSparseHookGraph()
	hooks2 := engine.NewSparseHookGraph()

	world1 := setupForgeryWorld(hooks1)
	world2 := setupForgeryWorld(hooks2)

	sys1 := NewForgerySystem(&world1, hooks1)
	sys2 := NewForgerySystem(&world2, hooks2)

	for i := 0; i < 71; i++ {
		sys1.Update(&world1)
		sys2.Update(&world2)
	}

	// Dump components and compare
	busQuery1 := world1.Query(sys1.businessFilter)
	busQuery2 := world2.Query(sys2.businessFilter)

	for busQuery1.Next() && busQuery2.Next() {
		bus1 := (*components.BusinessComponent)(busQuery1.Get(sys1.busCompID))
		bus2 := (*components.BusinessComponent)(busQuery2.Get(sys2.busCompID))

		if bus1.OwnerID != bus2.OwnerID {
			t.Fatalf("Determinism failure: Business ownership differs between worlds (%d vs %d)", bus1.OwnerID, bus2.OwnerID)
		}
	}
}

func setupForgeryWorld(hooks *engine.SparseHookGraph) ecs.World {
	world := ecs.NewWorld()

	// Owner
	ownerEnt := world.NewEntity()
	world.Add(ownerEnt,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)
	ownerIdent := (*components.Identity)(world.Get(ownerEnt, ecs.ComponentID[components.Identity](&world)))
	ownerIdent.ID = 100
	ownerGenome := (*components.GenomeComponent)(world.Get(ownerEnt, ecs.ComponentID[components.GenomeComponent](&world)))
	ownerGenome.Intellect = 50

	// Business
	busEnt := world.NewEntity()
	world.Add(busEnt,
		ecs.ComponentID[components.BusinessEntity](&world),
		ecs.ComponentID[components.BusinessComponent](&world),
	)
	busComp := (*components.BusinessComponent)(world.Get(busEnt, ecs.ComponentID[components.BusinessComponent](&world)))
	busComp.OwnerID = ownerIdent.ID

	// Forger
	forgerEnt := world.NewEntity()
	world.Add(forgerEnt,
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
	)
	forgerIdent := (*components.Identity)(world.Get(forgerEnt, ecs.ComponentID[components.Identity](&world)))
	forgerIdent.ID = 200
	forgerGenome := (*components.GenomeComponent)(world.Get(forgerEnt, ecs.ComponentID[components.GenomeComponent](&world)))
	forgerGenome.Intellect = 90
	forgerDesp := (*components.DesperationComponent)(world.Get(forgerEnt, ecs.ComponentID[components.DesperationComponent](&world)))
	forgerDesp.Level = 80

	return world
}
