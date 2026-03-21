package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestGenerationalDebtSystem_Integration(t *testing.T) {
	// Initialize Deterministic RNG
	engine.InitializeRNG([32]byte{1, 2, 3})

	tm := engine.NewTickManager(60)

	// Create systems
	hooks := engine.NewSparseHookGraph()
	deathSys := NewDeathSystem(tm.World, hooks)
	debtSys := NewDebtDefaultSystem(hooks)

	tm.AddSystem(deathSys, engine.PhaseResolution)
	tm.AddSystem(debtSys, engine.PhaseResolution)

	world := tm.World

	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	identID := ecs.ComponentID[components.Identity](world)
	loanID := ecs.ComponentID[components.LoanContractComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)

	// Create Father (Debtor)
	legacyID := ecs.ComponentID[components.Legacy](world)
	father := world.NewEntity(posID, needsID, affilID, identID, loanID, npcID, storageID, legacyID)
	fatherNeeds := (*components.Needs)(world.Get(father, needsID))
	fatherNeeds.Food = 0 // Starving
	fatherIdent := (*components.Identity)(world.Get(father, identID))
	fatherIdent.ID = 100
	fatherAffil := (*components.Affiliation)(world.Get(father, affilID))
	fatherAffil.FamilyID = 5
	fatherAffil.GuildID = 1 // Original Guild

	fatherLoan := (*components.LoanContractComponent)(world.Get(father, loanID))
	fatherLoan.CreditorID = 200
	fatherLoan.AssetID = 2 // Creditor's Guild
	fatherLoan.DueTick = 5 // Due soon

	fatherStorage := (*components.StorageComponent)(world.Get(father, storageID))
	fatherStorage.Food = 0 // Cannot pay

	// Create Son (Heir)
	son := world.NewEntity(posID, needsID, affilID, identID, npcID, storageID, legacyID)
	sonNeeds := (*components.Needs)(world.Get(son, needsID))
	sonNeeds.Food = 100 // Alive
	sonIdent := (*components.Identity)(world.Get(son, identID))
	sonIdent.ID = 101
	sonAffil := (*components.Affiliation)(world.Get(son, affilID))
	sonAffil.FamilyID = 5
	sonAffil.GuildID = 1

	sonStorage := (*components.StorageComponent)(world.Get(son, storageID))
	sonStorage.Food = 0 // Cannot pay

	// Create Creditor (so hooks can generate)
	creditor := world.NewEntity(posID, needsID, affilID, identID, npcID)
	creditorIdent := (*components.Identity)(world.Get(creditor, identID))
	creditorIdent.ID = 200
	creditorAffil := (*components.Affiliation)(world.Get(creditor, affilID))
	creditorAffil.GuildID = 2
	creditorNeeds := (*components.Needs)(world.Get(creditor, needsID))
	creditorNeeds.Food = 100 // Alive

	// Tick 1: Father dies. Succession triggers.
	tm.Tick()

	if world.Alive(father) {
		t.Errorf("Father should be dead")
	}

	if !world.Has(son, loanID) {
		t.Fatalf("Son did not inherit the LoanContractComponent")
	}

	sonLoan := (*components.LoanContractComponent)(world.Get(son, loanID))
	if sonLoan.CreditorID != 200 || sonLoan.AssetID != 2 || sonLoan.DueTick != 5 {
		t.Errorf("Son inherited incorrect loan details: %+v", sonLoan)
	}

	// Tick 2-4: Wait for DueTick
	tm.Tick()
	tm.Tick()
	tm.Tick()

	// Tick 5: Debt Default triggers on Son
	tm.Tick()

	if world.Has(son, loanID) {
		t.Errorf("LoanContractComponent should be removed after default")
	}

	sonAffilAfter := (*components.Affiliation)(world.Get(son, affilID))
	if sonAffilAfter.GuildID != 2 {
		t.Errorf("Son's GuildID should be seized by creditor (expected 2, got %d)", sonAffilAfter.GuildID)
	}

	// Check BloodFeud generation
	hook := hooks.GetHook(200, 101) // Creditor -> Son
	if hook != -50 {
		t.Errorf("Creditor should have massive grudge against defaulting Son (expected -50, got %d)", hook)
	}
}
