package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 15.3: Predatory Lending Engine (Systemic Emergence) Deterministic Tests
func TestLendingSystem_ButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	loanID := ecs.ComponentID[components.LoanContractComponent](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)

	// Create LendingSystem
	sys := NewLendingSystem(&world)

	// --- 1. Create a Wealthy Creditor ---
	creditor := world.NewEntity(posID, needsID, affID, identID)

	cPos := (*components.Position)(world.Get(creditor, posID))
	cPos.X, cPos.Y = 10.0, 10.0

	cNeeds := (*components.Needs)(world.Get(creditor, needsID))
	cNeeds.Wealth = 1000.0 // Very wealthy

	cAff := (*components.Affiliation)(world.Get(creditor, affID))
	cAff.GuildID = 42 // Target GuildID for defaults

	cIdent := (*components.Identity)(world.Get(creditor, identID))
	cIdent.ID = 999

	// --- 2. Create a Desperate Debtor ---
	debtor := world.NewEntity(posID, needsID, affID, identID, despID)

	dPos := (*components.Position)(world.Get(debtor, posID))
	dPos.X, dPos.Y = 11.0, 11.0 // Within lending distance

	dNeeds := (*components.Needs)(world.Get(debtor, needsID))
	dNeeds.Wealth = 10.0 // Poor

	dAff := (*components.Affiliation)(world.Get(debtor, affID))
	dAff.GuildID = 5 // Original GuildID

	dIdent := (*components.Identity)(world.Get(debtor, identID))
	dIdent.ID = 111

	dDesp := (*components.DesperationComponent)(world.Get(debtor, despID))
	dDesp.Level = 50 // Desperate enough to trigger a loan

	// --- 3. Execute LendingSystem ---
	// Need to fast-forward 10 ticks for the system to process (tickCounter % 10 == 0)
	for i := 0; i < 10; i++ {
		sys.Update(&world)
	}

	// Because adding new components to the Debtor structurally modifies the Archetype,
	// existing pointer references like dNeeds or dDesp MIGHT be invalidated if the
	// internal slices had to reallocate or move. We MUST fetch them again.
	dNeeds = (*components.Needs)(world.Get(debtor, needsID))
	cNeeds = (*components.Needs)(world.Get(creditor, needsID))
	dDesp = (*components.DesperationComponent)(world.Get(debtor, despID))

	// --- 4. Verify Lending Happened ---
	// Debtor should now have a LoanContractComponent
	if !world.Has(debtor, loanID) {
		t.Fatalf("Expected Debtor to receive a LoanContractComponent")
	}

	// Debtor should have received 100 wealth
	if dNeeds.Wealth != 110.0 {
		t.Fatalf("Expected Debtor wealth to be 110.0, got %f", dNeeds.Wealth)
	}

	// Creditor should have lost 100 wealth
	if cNeeds.Wealth != 900.0 {
		t.Fatalf("Expected Creditor wealth to be 900.0, got %f", cNeeds.Wealth)
	}

	// Desperation should be reset
	if dDesp.Level != 0 {
		t.Fatalf("Expected Debtor desperation to reset to 0, got %d", dDesp.Level)
	}

	// Verify Loan details
	loan := (*components.LoanContractComponent)(world.Get(debtor, loanID))
	if loan.CreditorID != 999 {
		t.Fatalf("Expected Loan CreditorID to be 999, got %d", loan.CreditorID)
	}
	if loan.AssetID != 42 {
		t.Fatalf("Expected Loan AssetID to be 42, got %d", loan.AssetID)
	}

	// --- 5. Verify Debt Default Butterfly Effect ---
	// Debtor lacks Storage to repay the loan. Let's run DebtDefaultSystem until DueTick.
	defaultSys := NewDebtDefaultSystem()
	defaultSys.Tick = loan.DueTick - 1 // Fast-forward to right before due

	// Ensure Debtor has a Storage component (LendingSystem should have added it)
	if !world.Has(debtor, storageID) {
		t.Fatalf("Expected LendingSystem to add StorageComponent to Debtor")
	}

	// Update DebtDefaultSystem
	defaultSys.Update(&world) // Now it is at DueTick

	// The Debtor should have defaulted because they have 0 resources in Storage
	// Their GuildID should be forcibly changed to the Creditor's GuildID (AssetID = 42)

	// The component pointer might be invalidated after structural changes (like removing the loan),
	// so we re-fetch it.
	dAffNew := (*components.Affiliation)(world.Get(debtor, affID))

	if dAffNew.GuildID != 42 {
		t.Fatalf("Expected Debtor GuildID to be reassigned to 42 upon default, got %d", dAffNew.GuildID)
	}

	// Verify the LoanContractComponent was removed
	if world.Has(debtor, loanID) {
		t.Fatalf("Expected LoanContractComponent to be removed after default")
	}
}
