package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 47: The Mercenary Engine (Testing & Validation)
// This test validates the core structural shift: converting wealth into physical violence via the SparseHookGraph.

func TestMercenarySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Component Registration for Determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.Needs](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.DesperationComponent](&world)
	ecs.ComponentID[components.MercenaryContractComponent](&world)
	ecs.ComponentID[components.NPC](&world)

	hooks := engine.NewSparseHookGraph()

	sys := NewMercenarySystem(&world, hooks)

	// 2. Spawn entities
	eClient := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.NPC](&world),
	)

	eTarget := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.NPC](&world),
	)

	eMerc := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.NPC](&world),
	)

	// 3. Setup Client (Wealthy, deep negative hook against target)
	cPos := (*components.Position)(world.Get(eClient, ecs.ComponentID[components.Position](&world)))
	cPos.X = 10.0
	cPos.Y = 10.0

	cIdent := (*components.Identity)(world.Get(eClient, ecs.ComponentID[components.Identity](&world)))
	cIdent.ID = 101

	cNeeds := (*components.Needs)(world.Get(eClient, ecs.ComponentID[components.Needs](&world)))
	cNeeds.Wealth = 1000.0 // Plenty of wealth to hire

	// 4. Setup Target
	tIdent := (*components.Identity)(world.Get(eTarget, ecs.ComponentID[components.Identity](&world)))
	tIdent.ID = 202

	// Set deep grudge from Client -> Target
	hooks.AddHook(cIdent.ID, tIdent.ID, -60)

	// 4b. Position of Target
	tPos := (*components.Position)(world.Get(eTarget, ecs.ComponentID[components.Position](&world)))
	tPos.X = 50.0
	tPos.Y = 50.0 // Far away

	// 5. Setup Mercenary (Desperate, Unemployed, Close proximity)
	mPos := (*components.Position)(world.Get(eMerc, ecs.ComponentID[components.Position](&world)))
	mPos.X = 12.0
	mPos.Y = 12.0 // Close to Client

	mIdent := (*components.Identity)(world.Get(eMerc, ecs.ComponentID[components.Identity](&world)))
	mIdent.ID = 303

	mJob := (*components.JobComponent)(world.Get(eMerc, ecs.ComponentID[components.JobComponent](&world)))
	mJob.JobID = components.JobNone

	mDesp := (*components.DesperationComponent)(world.Get(eMerc, ecs.ComponentID[components.DesperationComponent](&world)))
	mDesp.Level = 50 // Desperate enough to kill

	mNeeds := (*components.Needs)(world.Get(eMerc, ecs.ComponentID[components.Needs](&world)))
	mNeeds.Wealth = 10.0 // Poor

	// Verify our setups are correct via Query.
	clientCount := 0
	qC := world.Query(ecs.All(ecs.ComponentID[components.NPC](&world), ecs.ComponentID[components.Needs](&world), ecs.ComponentID[components.Position](&world), ecs.ComponentID[components.Identity](&world)))
	for qC.Next() {
		needs := (*components.Needs)(qC.Get(ecs.ComponentID[components.Needs](&world)))
		if needs.Wealth > 500.0 {
			clientCount++
		}
	}
	if clientCount != 1 {
		t.Errorf("Expected 1 client with wealth > 500, got %d", clientCount)
	}

	mercCount := 0
	qM := world.Query(ecs.All(ecs.ComponentID[components.NPC](&world), ecs.ComponentID[components.JobComponent](&world), ecs.ComponentID[components.Position](&world), ecs.ComponentID[components.DesperationComponent](&world), ecs.ComponentID[components.Identity](&world)))
	for qM.Next() {
		job := (*components.JobComponent)(qM.Get(ecs.ComponentID[components.JobComponent](&world)))
		desp := (*components.DesperationComponent)(qM.Get(ecs.ComponentID[components.DesperationComponent](&world)))
		if job.JobID == components.JobNone && desp.Level >= 30 {
			mercCount++
		}
	}
	if mercCount != 1 {
		t.Errorf("Expected 1 merc entity, got %d", mercCount)
	}

	// 6. Run System until execution tick
	// Must be exactly 100 ticks since the tick counter starts at 0 and increments at the beginning of Update, and fires if tickCounter%100 == 0
	for i := 0; i < 100; i++ {
		sys.Update(&world)
	}

	// Need to re-fetch pointers as arche-go can invalidate them when adding components like MercenaryContractComponent
	mNeeds = (*components.Needs)(world.Get(eMerc, ecs.ComponentID[components.Needs](&world)))
	mJob = (*components.JobComponent)(world.Get(eMerc, ecs.ComponentID[components.JobComponent](&world)))
	mIdent = (*components.Identity)(world.Get(eMerc, ecs.ComponentID[components.Identity](&world)))
	cNeeds = (*components.Needs)(world.Get(eClient, ecs.ComponentID[components.Needs](&world)))

	// 7. Verify the structural transition (The Butterfly Effect)

	// 7a. Client Wealth dropped
	if cNeeds.Wealth != 800.0 {
		t.Errorf("Client wealth did not drop correctly. Expected 800.0, got %f", cNeeds.Wealth)
	}

	// 7b. Mercenary Wealth increased
	if mNeeds.Wealth != 210.0 {
		t.Errorf("Mercenary wealth did not increase correctly. Expected 210.0, got %f", mNeeds.Wealth)
	}

	// 7c. Mercenary Job converted to Hitman
	if mJob.JobID != components.JobMercenary {
		t.Errorf("Mercenary did not acquire JobMercenary. Got JobID %d", mJob.JobID)
	}
	if mJob.EmployerID != cIdent.ID {
		t.Errorf("Mercenary did not acquire EmployerID of the Client. Expected %d, got %d", cIdent.ID, mJob.EmployerID)
	}

	// 7d. Mercenary gained the Contract component structurally
	if !world.Has(eMerc, ecs.ComponentID[components.MercenaryContractComponent](&world)) {
		t.Fatalf("Mercenary did not structurally receive the MercenaryContractComponent.")
	}

	contract := (*components.MercenaryContractComponent)(world.Get(eMerc, ecs.ComponentID[components.MercenaryContractComponent](&world)))
	if contract.TargetID != tIdent.ID {
		t.Errorf("Contract targeted wrong ID. Expected %d, got %d", tIdent.ID, contract.TargetID)
	}

	// 7e. MOST IMPORTANT: Mercenary natively acquired the Blood Feud against the target
	mercHook := hooks.GetHook(mIdent.ID, tIdent.ID)
	if mercHook != -100 { // Changed to check properly. It adds -100, so it should be exactly -100.
		t.Errorf("Mercenary did not inherit the negative Blood Feud hook. Expected -100, got %d", mercHook)
	}
}
