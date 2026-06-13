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

	// Component IDs
	identID := ecs.ComponentID[components.Identity](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	genomeID := ecs.ComponentID[components.GenomeComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	busID := ecs.ComponentID[components.BusinessComponent](&world)

	// Create Owner
	ownerEnt := world.NewEntity(identID, genomeID)

	ownerIdent := (*components.Identity)(world.Get(ownerEnt, identID))
	ownerIdent.ID = 101

	ownerGenome := (*components.GenomeComponent)(world.Get(ownerEnt, genomeID))
	ownerGenome.Intellect = 50

	// Create Business owned by Owner
	busEnt := world.NewEntity(busID)
	bus := (*components.BusinessComponent)(world.Get(busEnt, busID))
	bus.OwnerID = 101

	// Create Forger
	forgerEnt := world.NewEntity(identID, despID, genomeID, memID)

	forgerIdent := (*components.Identity)(world.Get(forgerEnt, identID))
	forgerIdent.ID = 202

	forgerDesp := (*components.DesperationComponent)(world.Get(forgerEnt, despID))
	forgerDesp.Level = 80 // > 50

	forgerGenome := (*components.GenomeComponent)(world.Get(forgerEnt, genomeID))
	forgerGenome.Intellect = 120 // > 100, and > owner's 50

	// Tick until it fires (offset 13)
	for i := 0; i < 13; i++ {
		sys.Update(&world)
	}

	// 1. Assert Ownership Transfer
	if bus.OwnerID != 202 {
		t.Errorf("Expected Business OwnerID to be transferred to 202, got %d", bus.OwnerID)
	}

	// 2. Assert SparseHookGraph negative hook (Owner -> Forger)
	grudge := hooks.GetHook(101, 202)
	if grudge != -100 {
		t.Errorf("Expected -100 grudge from owner to forger, got %d", grudge)
	}

	// 3. Assert InteractionTheft in Forger's Memory
	forgerMem := (*components.Memory)(world.Get(forgerEnt, memID))
	foundTheft := false
	for i := 0; i < len(forgerMem.Events); i++ {
		ev := forgerMem.Events[i]
		if ev.InteractionType == components.InteractionTheft && ev.TargetID == 101 {
			foundTheft = true
			break
		}
	}

	if !foundTheft {
		t.Errorf("Expected InteractionTheft (4) targeting owner 101 in forger's memory")
	}
}
