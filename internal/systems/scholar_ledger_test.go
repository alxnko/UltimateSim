package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 39: The Epistemological Engine Testing
// The "Butterfly Effect" proving ScholarSystem + LedgerDiscoverySystem ties to Military Revolt

func TestScholarLedgerSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.NPC](&world)
	ecs.ComponentID[components.GenomeComponent](&world)
	ecs.ComponentID[components.Needs](&world)
	ecs.ComponentID[components.SecretComponent](&world)
	ecs.ComponentID[components.Ledger](&world)
	ecs.ComponentID[components.LedgerComponent](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.CapitalComponent](&world)

	// Create HookGraph
	hookGraph := engine.NewSparseHookGraph()

	// Create Systems
	scholarSystem := NewScholarSystem(&world)
	discoverySystem := NewLedgerDiscoverySystem(&world)
	revoltSystem := NewMilitaryRevoltSystem(&world, hookGraph)

	// 1. Setup a State with a Banned Secret
	bannedSecretID := uint32(99)

	capEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.CapitalComponent](&world),
	)

	capIdent := (*components.Identity)(world.Get(capEnt, ecs.ComponentID[components.Identity](&world)))
	capIdent.ID = 100 // King/Capital ID

	capPos := (*components.Position)(world.Get(capEnt, ecs.ComponentID[components.Position](&world)))
	capPos.X = 10.0
	capPos.Y = 10.0

	capAff := (*components.Affiliation)(world.Get(capEnt, ecs.ComponentID[components.Affiliation](&world)))
	capAff.CityID = 1

	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	capJur.RadiusSquared = 100.0 // Radius 10
	capJur.BannedSecretID = bannedSecretID

	// 2. Setup a Scholar holding the banned secret
	scholarEnt := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.SecretComponent](&world),
	)

	scholarIdent := (*components.Identity)(world.Get(scholarEnt, ecs.ComponentID[components.Identity](&world)))
	scholarIdent.ID = 200

	scholarPos := (*components.Position)(world.Get(scholarEnt, ecs.ComponentID[components.Position](&world)))
	scholarPos.X = 12.0 // Same location
	scholarPos.Y = 12.0

	scholarGen := (*components.GenomeComponent)(world.Get(scholarEnt, ecs.ComponentID[components.GenomeComponent](&world)))
	scholarGen.Intellect = 200 // High Intellect

	scholarNeeds := (*components.Needs)(world.Get(scholarEnt, ecs.ComponentID[components.Needs](&world)))
	scholarNeeds.Wealth = 100.0 // Has enough wealth

	scholarSecrets := (*components.SecretComponent)(world.Get(scholarEnt, ecs.ComponentID[components.SecretComponent](&world)))
	scholarSecrets.Secrets = []components.Secret{{SecretID: bannedSecretID}}

	// 3. Setup a Loyal Guard WITHOUT the banned secret
	guardEnt := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.SecretComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.NPC](&world),
	)

	guardIdent := (*components.Identity)(world.Get(guardEnt, ecs.ComponentID[components.Identity](&world)))
	guardIdent.ID = 300 // Loyal Guard ID

	guardPos := (*components.Position)(world.Get(guardEnt, ecs.ComponentID[components.Position](&world)))
	guardPos.X = 15.0 // Initially away from Scholar
	guardPos.Y = 15.0

	guardJob := (*components.JobComponent)(world.Get(guardEnt, ecs.ComponentID[components.JobComponent](&world)))
	guardJob.JobID = components.JobGuard
	guardJob.EmployerID = capIdent.ID

	guardAff := (*components.Affiliation)(world.Get(guardEnt, ecs.ComponentID[components.Affiliation](&world)))
	guardAff.CityID = 1

	guardSecrets := (*components.SecretComponent)(world.Get(guardEnt, ecs.ComponentID[components.SecretComponent](&world)))
	// Initially no secrets

	// Step 4. Run the ScholarSystem until the Ledger is generated (100 ticks)
	for i := 0; i < 100; i++ {
		scholarSystem.Update(&world)
	}

	// Verify the Ledger was physically created
	ledgerTagID := ecs.ComponentID[components.Ledger](&world)
	ledgerCompID := ecs.ComponentID[components.LedgerComponent](&world)

	ledgerQuery := world.Query(ecs.All(ledgerTagID, ledgerCompID, ecs.ComponentID[components.Position](&world)))

	ledgerFound := false
	var ledgerX, ledgerY float32
	var ledgerSecrets []uint32

	for ledgerQuery.Next() {
		ledgerFound = true
		pos := (*components.Position)(ledgerQuery.Get(ecs.ComponentID[components.Position](&world)))
		ledgerComp := (*components.LedgerComponent)(ledgerQuery.Get(ledgerCompID))
		ledgerX = pos.X
		ledgerY = pos.Y
		ledgerSecrets = ledgerComp.Secrets
	}

	if !ledgerFound {
		t.Fatalf("Expected Ledger to be generated by the ScholarSystem.")
	}

	if ledgerX != 12.0 || ledgerY != 12.0 {
		t.Errorf("Expected Ledger to spawn at Scholar's position (12, 12), got (%f, %f)", ledgerX, ledgerY)
	}

	if len(ledgerSecrets) != 1 || ledgerSecrets[0] != bannedSecretID {
		t.Errorf("Expected Ledger to contain BannedSecretID (%d), got %v", bannedSecretID, ledgerSecrets)
	}

	// Step 5: The Scholar is killed (abstracted by despawning them)
	world.RemoveEntity(scholarEnt)

	// Step 6: Move the Guard over the Ledger
	guardPos.X = 12.0
	guardPos.Y = 12.0

	// Step 7: Run LedgerDiscoverySystem (ticks every 20)
	for i := 0; i < 20; i++ {
		discoverySystem.Update(&world)
	}

	// Verify Guard learned the secret
	if len(guardSecrets.Secrets) == 0 {
		t.Fatalf("Expected Guard to learn the secret from the Ledger, but they did not.")
	}
	if guardSecrets.Secrets[0].SecretID != bannedSecretID {
		t.Errorf("Expected Guard to learn BannedSecretID (%d), got %d", bannedSecretID, guardSecrets.Secrets[0].SecretID)
	}
	if guardSecrets.Secrets[0].Virality != 255 {
		t.Errorf("Expected highly contagious Virality 255, got %d", guardSecrets.Secrets[0].Virality)
	}

	// Step 8: The Butterfly Effect - Guard Revolts
	// Run MilitaryRevoltSystem (ticks every 10)
	for i := 0; i < 10; i++ {
		revoltSystem.Update(&world)
	}

	if guardJob.JobID != components.JobBandit {
		t.Errorf("Expected Guard to drop JobGuard and become JobBandit upon learning BannedSecretID, got %d", guardJob.JobID)
	}

	hookVal := hookGraph.GetHook(guardIdent.ID, capIdent.ID)
	if hookVal != -100 {
		t.Errorf("Expected Guard to gain -100 hook against Capital, got %d", hookVal)
	}
}
