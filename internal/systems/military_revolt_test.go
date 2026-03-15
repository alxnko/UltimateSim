package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 27.1: The Military Revolt Engine Testing
// The "Butterfly Effect" proving Military Revolt System ties to Information Leakage, Justice, and Blood Feuds.

func TestMilitaryRevoltSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.SecretComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.CapitalComponent](&world)

	// Create SparseHookGraph
	hookGraph := engine.NewSparseHookGraph()

	// Create Military Revolt System
	revoltSystem := NewMilitaryRevoltSystem(&world, hookGraph)

	// 1. Create a Capital Jurisdiction with a Banned Secret ID
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

	// 2. Create a Guard NPC working for the Capital
	guardNPC := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.SecretComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)

	guardIdent := (*components.Identity)(world.Get(guardNPC, ecs.ComponentID[components.Identity](&world)))
	guardIdent.ID = 200 // Guard ID

	guardPos := (*components.Position)(world.Get(guardNPC, ecs.ComponentID[components.Position](&world)))
	guardPos.X = 12.0 // In radius
	guardPos.Y = 12.0

	guardJob := (*components.JobComponent)(world.Get(guardNPC, ecs.ComponentID[components.JobComponent](&world)))
	guardJob.JobID = components.JobGuard
	guardJob.EmployerID = capIdent.ID

	guardAff := (*components.Affiliation)(world.Get(guardNPC, ecs.ComponentID[components.Affiliation](&world)))
	guardAff.CityID = 1

	guardSecrets := (*components.SecretComponent)(world.Get(guardNPC, ecs.ComponentID[components.SecretComponent](&world)))
	// Initially no secrets

	// 3. Create another Guard without the secret
	loyalGuardNPC := world.NewEntity(
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.SecretComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)

	loyalIdent := (*components.Identity)(world.Get(loyalGuardNPC, ecs.ComponentID[components.Identity](&world)))
	loyalIdent.ID = 300 // Loyal Guard ID

	loyalPos := (*components.Position)(world.Get(loyalGuardNPC, ecs.ComponentID[components.Position](&world)))
	loyalPos.X = 13.0
	loyalPos.Y = 13.0

	loyalJob := (*components.JobComponent)(world.Get(loyalGuardNPC, ecs.ComponentID[components.JobComponent](&world)))
	loyalJob.JobID = components.JobGuard
	loyalJob.EmployerID = capIdent.ID

	loyalAff := (*components.Affiliation)(world.Get(loyalGuardNPC, ecs.ComponentID[components.Affiliation](&world)))
	loyalAff.CityID = 1

	// Give loyal guard some irrelevant secret
	loyalSecrets := (*components.SecretComponent)(world.Get(loyalGuardNPC, ecs.ComponentID[components.SecretComponent](&world)))
	loyalSecrets.Secrets = []components.Secret{{SecretID: 42}}

	// 4. Simulate ticks (without the banned secret)
	for i := 0; i < 10; i++ { // System triggers on tick 10
		revoltSystem.Update(&world)
	}

	// Assert guards are still guards and no hooks exist
	if guardJob.JobID != components.JobGuard {
		t.Errorf("Expected guard to remain a Guard before learning secret.")
	}
	hookVal := hookGraph.GetHook(guardIdent.ID, capIdent.ID)
	if hookVal != 0 {
		t.Errorf("Expected no hook against Capital before learning secret, got %d", hookVal)
	}

	// 5. Inject the Banned Secret (Simulating GossipDistributionSystem)
	guardSecrets.Secrets = append(guardSecrets.Secrets, components.Secret{SecretID: bannedSecretID})

	// 6. Simulate ticks again
	for i := 0; i < 10; i++ {
		revoltSystem.Update(&world)
	}

	// 7. Verify the Butterfly Effects

	// Assertion A: Guard has dropped JobGuard and became a Bandit
	if guardJob.JobID != components.JobBandit {
		t.Errorf("Expected Guard to drop JobGuard and become JobBandit upon learning BannedSecretID, got %d", guardJob.JobID)
	}
	if guardJob.EmployerID != 0 {
		t.Errorf("Expected Guard to sever employment with Capital, got %d", guardJob.EmployerID)
	}

	// Assertion B: Guard gained a massive negative hook (-100) against the Capital/King
	hookVal = hookGraph.GetHook(guardIdent.ID, capIdent.ID)
	if hookVal != -100 {
		t.Errorf("Expected Guard to gain -100 hook against Capital, got %d", hookVal)
	}

	// Assertion C: Loyal Guard is unaffected
	if loyalJob.JobID != components.JobGuard {
		t.Errorf("Expected Loyal Guard to remain a Guard.")
	}
	loyalHookVal := hookGraph.GetHook(loyalIdent.ID, capIdent.ID)
	if loyalHookVal != 0 {
		t.Errorf("Expected Loyal Guard to have no hook against Capital.")
	}
}
