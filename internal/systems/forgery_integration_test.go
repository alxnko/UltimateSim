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

	sys := NewForgerySystem(&world, hookGraph)

	// Set up the components for the query
	busTagID := ecs.ComponentID[components.BusinessEntity](&world)
	busCompID := ecs.ComponentID[components.BusinessComponent](&world)
	wpID := ecs.ComponentID[components.WorkplaceComponent](&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)

	npcID := ecs.ComponentID[components.NPC](&world)
	posID := ecs.ComponentID[components.Position](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	memID := ecs.ComponentID[components.Memory](&world)

	// Create Victim (Dumb rich owner)
	victim := world.NewEntity(identID, genID)
	vIdent := (*components.Identity)(world.Get(victim, identID))
	vIdent.ID = 101
	vGen := (*components.GenomeComponent)(world.Get(victim, genID))
	vGen.Intellect = 80

	// Create Business owned by Victim
	business := world.NewEntity(busTagID, busCompID, wpID, treasuryID)
	busComp := (*components.BusinessComponent)(world.Get(business, busCompID))
	busComp.OwnerID = vIdent.ID
	wpComp := (*components.WorkplaceComponent)(world.Get(business, wpID))
	wpComp.X = 10.0
	wpComp.Y = 10.0
	treasuryComp := (*components.TreasuryComponent)(world.Get(business, treasuryID))
	treasuryComp.Wealth = 5000.0 // Wealthy business

	// Create Forger (Smart poor desperate)
	forger := world.NewEntity(npcID, posID, despID, genID, identID, memID)
	fIdent := (*components.Identity)(world.Get(forger, identID))
	fIdent.ID = 202
	fGen := (*components.GenomeComponent)(world.Get(forger, genID))
	fGen.Intellect = 150
	fDesp := (*components.DesperationComponent)(world.Get(forger, despID))
	fDesp.Level = 80 // Desperate
	fPos := (*components.Position)(world.Get(forger, posID))
	fPos.X = 11.0 // Within range (distSq = 1.0 + 0 = 1.0 < 4.0)
	fPos.Y = 10.0
	fMem := (*components.Memory)(world.Get(forger, memID))
	fMem.Head = 0

	// Run system (needs 50 ticks to trigger offset)
	for i := 0; i < 50; i++ {
		sys.Update(&world)
	}

	// Validate Ownership transferred
	updatedBusComp := (*components.BusinessComponent)(world.Get(business, busCompID))
	if updatedBusComp.OwnerID != fIdent.ID {
		t.Errorf("Expected Business OwnerID to be transferred to forger %d, got %d", fIdent.ID, updatedBusComp.OwnerID)
	}

	// Validate Desperation reset
	updatedDesp := (*components.DesperationComponent)(world.Get(forger, despID))
	if updatedDesp.Level != 0 {
		t.Errorf("Expected Forger Desperation Level to be reset to 0, got %d", updatedDesp.Level)
	}

	// Validate Hook generated
	hookVal := hookGraph.GetHook(vIdent.ID, fIdent.ID)
	if hookVal != -100 {
		t.Errorf("Expected victim to have -100 hook against forger, got %d", hookVal)
	}

	// Validate Memory InteractionTheft
	mem := (*components.Memory)(world.Get(forger, memID))
	if mem.Head != 1 {
		t.Errorf("Expected memory head to increment to 1, got %d", mem.Head)
	}
	if mem.Events[0].InteractionType != components.InteractionTheft {
		t.Errorf("Expected memory event InteractionType to be InteractionTheft (4), got %d", mem.Events[0].InteractionType)
	}
	if mem.Events[0].TargetID != vIdent.ID {
		t.Errorf("Expected memory event TargetID to be victim %d, got %d", vIdent.ID, mem.Events[0].TargetID)
	}
}
