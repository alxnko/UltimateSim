package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 04.5: The Epistemological Layer Testing
// The "Butterfly Effect" proving Propaganda System ties to Aging, Justice, and Secrets.

func TestPropagandaSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.NPC](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.SecretComponent](&world)
	ecs.ComponentID[components.Needs](&world)
	ecs.ComponentID[components.Ledger](&world)
	ecs.ComponentID[components.LedgerComponent](&world)

	// Create Propaganda System
	propagandaSystem := NewPropagandaSystem(&world)

	// 1. Create a Jurisdiction with a Banned Secret ID
	bannedSecretID := uint32(42)

	jurEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
	)

	jurPos := (*components.Position)(world.Get(jurEnt, ecs.ComponentID[components.Position](&world)))
	jurPos.X = 10.0
	jurPos.Y = 10.0

	jur := (*components.JurisdictionComponent)(world.Get(jurEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	jur.RadiusSquared = 100.0 // Radius 10
	jur.BannedSecretID = bannedSecretID

	// 2. Create a Young NPC (< 30)
	youngNPC := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.SecretComponent](&world),
		ecs.ComponentID[components.Needs](&world),
	)

	youngIdent := (*components.Identity)(world.Get(youngNPC, ecs.ComponentID[components.Identity](&world)))
	youngIdent.Age = 20

	youngPos := (*components.Position)(world.Get(youngNPC, ecs.ComponentID[components.Position](&world)))
	youngPos.X = 12.0 // In radius
	youngPos.Y = 12.0

	youngSecrets := (*components.SecretComponent)(world.Get(youngNPC, ecs.ComponentID[components.SecretComponent](&world)))
	youngSecrets.Secrets = []components.Secret{{SecretID: bannedSecretID}}

	youngNeeds := (*components.Needs)(world.Get(youngNPC, ecs.ComponentID[components.Needs](&world)))
	youngNeeds.Food = 100.0

	// 3. Create an Old NPC (>= 60)
	oldNPC := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.SecretComponent](&world),
		ecs.ComponentID[components.Needs](&world),
	)

	oldIdent := (*components.Identity)(world.Get(oldNPC, ecs.ComponentID[components.Identity](&world)))
	oldIdent.Age = 65

	oldPos := (*components.Position)(world.Get(oldNPC, ecs.ComponentID[components.Position](&world)))
	oldPos.X = 15.0 // In radius
	oldPos.Y = 15.0

	oldSecrets := (*components.SecretComponent)(world.Get(oldNPC, ecs.ComponentID[components.SecretComponent](&world)))
	oldSecrets.Secrets = []components.Secret{{SecretID: bannedSecretID}}

	oldNeeds := (*components.Needs)(world.Get(oldNPC, ecs.ComponentID[components.Needs](&world)))
	oldNeeds.Food = 100.0

	// 4. Create a Physical Ledger Entity
	ledgerEnt := world.NewEntity(
		ecs.ComponentID[components.Ledger](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.LedgerComponent](&world),
	)

	ledgerPos := (*components.Position)(world.Get(ledgerEnt, ecs.ComponentID[components.Position](&world)))
	ledgerPos.X = 11.0
	ledgerPos.Y = 11.0

	ledgerComp := (*components.LedgerComponent)(world.Get(ledgerEnt, ecs.ComponentID[components.LedgerComponent](&world)))
	ledgerComp.Secrets = []uint32{bannedSecretID}

	// 5. Create an NPC outside of the jurisdiction (should be unaffected)
	safeNPC := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.SecretComponent](&world),
		ecs.ComponentID[components.Needs](&world),
	)

	safeIdent := (*components.Identity)(world.Get(safeNPC, ecs.ComponentID[components.Identity](&world)))
	safeIdent.Age = 65 // Old enough to execute

	safePos := (*components.Position)(world.Get(safeNPC, ecs.ComponentID[components.Position](&world)))
	safePos.X = 100.0 // Far out of radius
	safePos.Y = 100.0

	safeSecrets := (*components.SecretComponent)(world.Get(safeNPC, ecs.ComponentID[components.SecretComponent](&world)))
	safeSecrets.Secrets = []components.Secret{{SecretID: bannedSecretID}}

	safeNeeds := (*components.Needs)(world.Get(safeNPC, ecs.ComponentID[components.Needs](&world)))
	safeNeeds.Food = 100.0

	// 6. Simulate ticks
	for i := 0; i < 20; i++ { // System triggers on tick 20
		propagandaSystem.Update(&world)
	}

	// 7. Verify the Butterfly Effects

	// Assertion A: Young NPC is alive, but memory erased
	if !world.Alive(youngNPC) {
		t.Errorf("Expected Young NPC to survive propaganda erasure.")
	}
	ySecrets := (*components.SecretComponent)(world.Get(youngNPC, ecs.ComponentID[components.SecretComponent](&world)))
	if len(ySecrets.Secrets) != 0 {
		t.Errorf("Expected Young NPC to have Banned Secret erased, but found %d secrets", len(ySecrets.Secrets))
	}

	// Assertion B: Old NPC is dead (Needs.Food set to 0 to be executed by DeathSystem later)
	if !world.Alive(oldNPC) {
		t.Errorf("Expected Old NPC to still be in world (DeathSystem not run).")
	}
	oNeeds := (*components.Needs)(world.Get(oldNPC, ecs.ComponentID[components.Needs](&world)))
	if oNeeds.Food != 0 {
		t.Errorf("Expected Old NPC to have Food=0 (executed), but got %f", oNeeds.Food)
	}

	// Assertion C: Ledger burned (Removed from world)
	if world.Alive(ledgerEnt) {
		t.Errorf("Expected Physical Ledger to be destroyed (burned) by the state.")
	}

	// Assertion D: Safe NPC is unaffected
	if !world.Alive(safeNPC) {
		t.Errorf("Expected Safe NPC to survive.")
	}
	sNeeds := (*components.Needs)(world.Get(safeNPC, ecs.ComponentID[components.Needs](&world)))
	if sNeeds.Food == 0 {
		t.Errorf("Expected Safe NPC to avoid execution.")
	}
	sSecrets := (*components.SecretComponent)(world.Get(safeNPC, ecs.ComponentID[components.SecretComponent](&world)))
	if len(sSecrets.Secrets) == 0 {
		t.Errorf("Expected Safe NPC to keep banned secret.")
	}
}
