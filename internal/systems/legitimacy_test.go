package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 35.1: Sovereign Legitimacy Engine Testing
// The "Butterfly Effect" proving Legitimacy Score calculation ties to Economy (Treasury),
// Justice (Corruption), Public Sentiment (Hooks), and finally triggers Military Revolts.

func TestLegitimacySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// Register Components explicitly for Arche-Go determinism
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.JobComponent](&world)
	ecs.ComponentID[components.SecretComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.JurisdictionComponent](&world)
	ecs.ComponentID[components.CapitalComponent](&world)
	ecs.ComponentID[components.LegitimacyComponent](&world)
	ecs.ComponentID[components.TreasuryComponent](&world)
	ecs.ComponentID[components.EquipmentComponent](&world)

	// Create SparseHookGraph
	hookGraph := engine.NewSparseHookGraph()

	// Create Systems
	legitimacySystem := NewLegitimacySystem(&world, hookGraph)
	revoltSystem := NewMilitaryRevoltSystem(&world, hookGraph)

	// 1. Create a Capital Jurisdiction
	capEnt := world.NewEntity(
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.JurisdictionComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.CapitalComponent](&world),
		ecs.ComponentID[components.LegitimacyComponent](&world),
		ecs.ComponentID[components.TreasuryComponent](&world),
	)

	capIdent := (*components.Identity)(world.Get(capEnt, ecs.ComponentID[components.Identity](&world)))
	capIdent.ID = 100 // King/Capital ID

	capPos := (*components.Position)(world.Get(capEnt, ecs.ComponentID[components.Position](&world)))
	capPos.X = 10.0
	capPos.Y = 10.0

	capAff := (*components.Affiliation)(world.Get(capEnt, ecs.ComponentID[components.Affiliation](&world)))
	capAff.CityID = 1

	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, ecs.ComponentID[components.JurisdictionComponent](&world)))
	capJur.RadiusSquared = 100.0
	capJur.Corruption = 0 // Initially no corruption

	capTreasury := (*components.TreasuryComponent)(world.Get(capEnt, ecs.ComponentID[components.TreasuryComponent](&world)))
	capTreasury.Wealth = 10000.0 // Initially very wealthy

	capLegitimacy := (*components.LegitimacyComponent)(world.Get(capEnt, ecs.ComponentID[components.LegitimacyComponent](&world)))
	capLegitimacy.Score = 100

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
	guardSecrets.Secrets = []components.Secret{}

	// Step A: Assert stable state
	for i := 0; i < 50; i++ {
		legitimacySystem.Update(&world)
		revoltSystem.Update(&world)
	}

	if capLegitimacy.Score != 60 { // Base 50 + (10000/1000 = 10 bonus) - 0 corruption + 0 hooks
		t.Fatalf("Expected stable legitimacy of 60, got %d", capLegitimacy.Score)
	}

	if guardJob.JobID != components.JobGuard {
		t.Fatalf("Expected guard to remain loyal under stable legitimacy")
	}

	// Step B: Inject massive corruption and negative public sentiment to crush legitimacy
	// But introduce Phase 52.1 Auras of Legitimacy!
	capJur.Corruption = 20           // -40 penalty
	capTreasury.Wealth = 0.0         // 0 bonus
	hookGraph.AddHook(999, capIdent.ID, -500) // Someone hates the King -> -50 penalty

	// Add an ancient artifact to the King's entity
	// Let's use an 800 prestige artifact (+80 Legitimacy)
	world.Add(capEnt, ecs.ComponentID[components.EquipmentComponent](&world))
	capEquip := (*components.EquipmentComponent)(world.Get(capEnt, ecs.ComponentID[components.EquipmentComponent](&world)))
	capEquip.Equipped = true
	capEquip.Weapon = components.LegendComponent{
		Prestige: 800, // Grants +80 Legitimacy
	}

	// Re-get capLegitimacy since it was modified by another system maybe? Actually no,
	// wait, let's just make sure the component is retrieved properly and updated.
	// Oh! I added the EquipmentComponent. But `capLegitimacy.Score` might be 0 because
	// the `Update` loop was called for 50 ticks, but wait. `capLegitimacy` is a pointer
	// retrieved earlier from the world. If we added a component to `capEnt`, Arche-Go
	// might have reallocated the entity's memory! So the old `capLegitimacy` pointer is invalid!

	// Let's re-acquire the pointer.
	capLegitimacy = (*components.LegitimacyComponent)(world.Get(capEnt, ecs.ComponentID[components.LegitimacyComponent](&world)))

	// Run systems to update score. Legitimacy triggers every 50 ticks, Revolt every 10 ticks.
	// But wait, if revoltSystem triggers BEFORE legitimacySystem updates, the guard might revolt!
	// Legitimacy System needs to update first. Let's step tick-by-tick carefully.
	for i := 0; i < 50; i++ {
		legitimacySystem.Update(&world)
		// Don't run revolt system yet! Let Legitimacy update first.
	}

	// Expected Legitimacy: Base 50 + 0 (wealth) - 40 (corruption) - 50 (hooks) + 80 (Artifact) = 40 Legitimacy
	if capLegitimacy.Score != 40 {
		t.Fatalf("Expected legitimacy to be held up by artifact at 40, got %d", capLegitimacy.Score)
	}

	// Now we can safely run the revolt system. It shouldn't revolt because 40 >= 20.
	for i := 0; i < 50; i++ {
		revoltSystem.Update(&world)
	}

	// The Guard's job component might also have moved in memory. Re-acquire it.
	guardJob = (*components.JobComponent)(world.Get(guardNPC, ecs.ComponentID[components.JobComponent](&world)))

	// Assertion: The Guard did NOT revolt because the artifact is holding the state together
	if guardJob.JobID != components.JobGuard {
		t.Fatalf("Expected Guard to remain loyal due to ancient artifact aura, but became %d", guardJob.JobID)
	}

	// Step C: The Artifact is stolen or unequipped
	capEquip.Equipped = false

	// Re-acquire pointers again just in case memory shifted, though removing a flag won't do it.
	// But let's just make sure we get the newest job status.
	// We run Legitimacy System first to crash the score.
	for i := 0; i < 50; i++ {
		legitimacySystem.Update(&world)
	}

	// Expected Legitimacy: Base 50 + 0 (wealth) - 40 (corruption) - 50 (hooks) = -40 -> 0 Legitimacy
	if capLegitimacy.Score != 0 {
		t.Fatalf("Expected legitimacy to crash to 0 without the artifact, got %d", capLegitimacy.Score)
	}

	// Now run the Revolt System so the guard acts on the 0 legitimacy.
	for i := 0; i < 50; i++ {
		revoltSystem.Update(&world)
	}

	guardJob = (*components.JobComponent)(world.Get(guardNPC, ecs.ComponentID[components.JobComponent](&world)))

	// Re-acquire guardIdent to get the ID correctly
	guardIdent = (*components.Identity)(world.Get(guardNPC, ecs.ComponentID[components.Identity](&world)))
	capIdent = (*components.Identity)(world.Get(capEnt, ecs.ComponentID[components.Identity](&world)))

	// Assertion: Guard has dropped JobGuard and became a Bandit due to low legitimacy
	if guardJob.JobID != components.JobBandit {
		t.Fatalf("Expected Guard to drop JobGuard and become JobBandit upon Legitimacy crash, got %d", guardJob.JobID)
	}
	if guardJob.EmployerID != 0 {
		t.Fatalf("Expected Guard to sever employment with Capital, got %d", guardJob.EmployerID)
	}

	// Assertion: Guard gained a massive negative hook (-100) against the Capital/King
	hookVal := hookGraph.GetHook(guardIdent.ID, capIdent.ID)
	if hookVal != -100 {
		t.Fatalf("Expected Guard to gain -100 hook against Capital, got %d", hookVal)
	}
}
