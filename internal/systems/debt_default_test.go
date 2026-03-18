package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 10.1: Debt Default Execution (The Hook Trap) Tests

func TestDebtDefaultSystem(t *testing.T) {
	world := ecs.NewWorld()
	tm := engine.NewTickManager(60)

	sys := NewDebtDefaultSystem(nil)
	tm.AddSystem(sys, engine.PhaseResolution)

	loanID := ecs.ComponentID[components.LoanContractComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	affiliationID := ecs.ComponentID[components.Affiliation](&world)

	// Entity 1: Insufficient Storage (Default Expected)
	e1 := world.NewEntity()
	world.Add(e1, loanID, storageID, affiliationID)

	loan1 := (*components.LoanContractComponent)(world.Get(e1, loanID))
	loan1.DueTick = 10
	loan1.AssetID = 99 // Guild ID to transfer on default

	storage1 := (*components.StorageComponent)(world.Get(e1, storageID))
	storage1.Food = 10
	storage1.Wood = 10

	aff1 := (*components.Affiliation)(world.Get(e1, affiliationID))
	aff1.GuildID = 1

	// Entity 2: Sufficient Storage (Repayment Expected)
	e2 := world.NewEntity()
	world.Add(e2, loanID, storageID, affiliationID)

	loan2 := (*components.LoanContractComponent)(world.Get(e2, loanID))
	loan2.DueTick = 10
	loan2.AssetID = 99

	storage2 := (*components.StorageComponent)(world.Get(e2, storageID))
	storage2.Food = 50
	storage2.Wood = 60 // Total = 110 >= 100

	aff2 := (*components.Affiliation)(world.Get(e2, affiliationID))
	aff2.GuildID = 1

	// Run ticks 1 to 9 (not due yet)
	for i := 0; i < 9; i++ {
		sys.Update(&world)
	}

	// Verify nothing changed yet
	if aff1.GuildID != 1 || !world.Has(e1, loanID) {
		t.Errorf("Entity 1 defaulted prematurely")
	}

	if aff2.GuildID != 1 || !world.Has(e2, loanID) {
		t.Errorf("Entity 2 repaid prematurely")
	}

	// Run tick 10 (contracts are due)
	sys.Update(&world)

	// Verify Entity 1 Defaulted
	if world.Has(e1, loanID) {
		t.Errorf("Entity 1 still has LoanContractComponent after default")
	}
	aff1 = (*components.Affiliation)(world.Get(e1, affiliationID))
	if aff1.GuildID != 99 {
		t.Errorf("Entity 1 did not transfer GuildID on default, got %d", aff1.GuildID)
	}

	// Verify Entity 2 Repaid
	if world.Has(e2, loanID) {
		t.Errorf("Entity 2 still has LoanContractComponent after repayment")
	}
	aff2 = (*components.Affiliation)(world.Get(e2, affiliationID))
	if aff2.GuildID != 1 {
		t.Errorf("Entity 2 transferred GuildID on repayment but shouldn't have, got %d", aff2.GuildID)
	}
	storage2 = (*components.StorageComponent)(world.Get(e2, storageID))
	if storage2.Food != 0 || storage2.Wood != 10 { // 50 food deducted, 50 wood deducted
		t.Errorf("Entity 2 storage not correctly deducted, got Food=%d, Wood=%d", storage2.Food, storage2.Wood)
	}
}

// Phase 10.1: Contractual Law & Blacklisting Butterfly Effect
func TestDebtDefaultSystem_Blacklisting(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	sys := NewDebtDefaultSystem(hookGraph)

	loanID := ecs.ComponentID[components.LoanContractComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	affiliationID := ecs.ComponentID[components.Affiliation](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// Debtor Entity: Defaults on loan to Guild 99
	debtor := world.NewEntity(loanID, storageID, affiliationID, identID, npcID)
	loanD := (*components.LoanContractComponent)(world.Get(debtor, loanID))
	loanD.DueTick = 10
	loanD.AssetID = 99 // Target Guild ID for default

	storageD := (*components.StorageComponent)(world.Get(debtor, storageID))
	storageD.Food = 0 // Will cause default

	affD := (*components.Affiliation)(world.Get(debtor, affiliationID))
	affD.GuildID = 1

	identD := (*components.Identity)(world.Get(debtor, identID))
	identD.ID = 100

	// Guild Member 1: Belongs to Guild 99
	member1 := world.NewEntity(affiliationID, identID, npcID)
	affM1 := (*components.Affiliation)(world.Get(member1, affiliationID))
	affM1.GuildID = 99
	identM1 := (*components.Identity)(world.Get(member1, identID))
	identM1.ID = 200

	// Guild Member 2: Belongs to Guild 99
	member2 := world.NewEntity(affiliationID, identID, npcID)
	affM2 := (*components.Affiliation)(world.Get(member2, affiliationID))
	affM2.GuildID = 99
	identM2 := (*components.Identity)(world.Get(member2, identID))
	identM2.ID = 201

	// Unrelated NPC: Belongs to Guild 5
	unrelated := world.NewEntity(affiliationID, identID, npcID)
	affU := (*components.Affiliation)(world.Get(unrelated, affiliationID))
	affU.GuildID = 5
	identU := (*components.Identity)(world.Get(unrelated, identID))
	identU.ID = 300

	// Run to DueTick
	for i := 0; i < 10; i++ {
		sys.Update(&world)
	}

	// Verify the debtor's guild changed to the creditor's
	affD = (*components.Affiliation)(world.Get(debtor, affiliationID))
	if affD.GuildID != 99 {
		t.Fatalf("Debtor did not transfer to Guild 99 upon default")
	}

	// Verify Blacklisting Hooks
	// Member 1 should have a -50 hook against Debtor (100)
	hook1 := hookGraph.GetHook(200, 100)
	if hook1 != -50 {
		t.Errorf("Expected Member 1 to have a -50 hook against debtor, got %d", hook1)
	}

	// Member 2 should have a -50 hook against Debtor (100)
	hook2 := hookGraph.GetHook(201, 100)
	if hook2 != -50 {
		t.Errorf("Expected Member 2 to have a -50 hook against debtor, got %d", hook2)
	}

	// Unrelated should NOT have a hook against Debtor (100)
	hookU := hookGraph.GetHook(300, 100)
	if hookU != 0 {
		t.Errorf("Expected Unrelated NPC to have 0 hook against debtor, got %d", hookU)
	}
}
