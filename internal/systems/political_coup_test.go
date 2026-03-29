package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 51 - The Debt-Trap Political Coup Engine
// E2E Butterfly Effect Test:
// Lending (Debt Hooks) -> Political Coup -> Sovereignty Swap

func TestPoliticalCoup_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// 1. Setup IDs
	npcID := ecs.ComponentID[components.NPC](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	affilID := ecs.ComponentID[components.Affiliation](&world)
	adminID := ecs.ComponentID[components.AdministrationMarker](&world)

	// 2. Setup Ruler NPC (Entity 1)
	rulerEnt := world.NewEntity()
	world.Add(rulerEnt, npcID, identID, affilID, adminID)

	ident1 := (*components.Identity)(world.Get(rulerEnt, identID))
	ident1.ID = 101

	affil1 := (*components.Affiliation)(world.Get(rulerEnt, affilID))
	affil1.CityID = 5

	// 3. Setup Creditor/Guild Master NPC (Entity 2)
	creditorEnt := world.NewEntity()
	world.Add(creditorEnt, npcID, identID, affilID)

	ident2 := (*components.Identity)(world.Get(creditorEnt, identID))
	ident2.ID = 102

	affil2 := (*components.Affiliation)(world.Get(creditorEnt, affilID))
	affil2.CityID = 5 // Must be in the same city to launch the coup

	// 4. Verify initial state
	if !world.Has(rulerEnt, adminID) {
		t.Fatalf("Ruler should have the AdministrationMarker initially")
	}

	if world.Has(creditorEnt, adminID) {
		t.Fatalf("Creditor should not have the AdministrationMarker initially")
	}

	// 5. Initialize the System
	sys := NewPoliticalCoupSystem(hookGraph)
	// Force tick 100 to execute immediately
	sys.tickCounter = 99

	// 6. Test Base State (No debt hooks)
	sys.Update(&world)
	if !world.Has(rulerEnt, adminID) {
		t.Fatalf("Ruler should still have the marker with no hooks")
	}

	// 7. Inject extreme debt hooks (Phase 10/46 Integration)
	// The creditor gains a massive -250 grudge against the ruler via unpaid predatory lending.
	hookGraph.AddHook(102, 101, -250)

	// 8. Execute Political Coup
	sys.tickCounter = 199
	sys.Update(&world)

	// 9. Verify the structural transition (The Butterfly Effect)
	if world.Has(rulerEnt, adminID) {
		t.Errorf("Ruler was not overthrown! They still have the AdministrationMarker despite extreme debt.")
	}

	if !world.Has(creditorEnt, adminID) {
		t.Errorf("Creditor did not seize power! They should have gained the AdministrationMarker.")
	}

	// 10. Verify that the debt hooks were executed and cleared
	hooks := hookGraph.GetAllIncomingHooks(101)
	if hooks[102] != 0 {
		t.Errorf("Expected the extreme debt hook to be cleared as collateral, but got %d", hooks[102])
	}
}
