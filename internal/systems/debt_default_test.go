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

	sys := NewDebtDefaultSystem()
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
