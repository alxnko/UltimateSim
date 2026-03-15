package systems

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 22.1: The Corruption Engine (End-to-End Test)
// Proves the "Butterfly Effect" where a wealthy criminal bribes a guard,
// generating local Corruption, which directly causes the distant village
// to prematurely fracture from the country due to reduced administrative reach.
func TestCorruption_ButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()

	// Initialize Systems
	justiceSys := NewJusticeSystem(&world, engine.NewSparseHookGraph())
	adminSys := NewAdministrativeFractureSystem(&world)

	// Fetch Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	capID := ecs.ComponentID[components.CapitalComponent](&world)
	countryID := ecs.ComponentID[components.CountryComponent](&world)
	villID := ecs.ComponentID[components.Village](&world)

	// --- SETUP SCENE ---

	// 1. Capital Entity (with Jurisdiction)
	capEnt := world.NewEntity(posID, affID, jurID, capID, countryID)
	capPos := (*components.Position)(world.Get(capEnt, posID))
	capPos.X, capPos.Y = 0, 0

	capAff := (*components.Affiliation)(world.Get(capEnt, affID))
	capAff.CityID = 1
	capAff.CountryID = 1 // It is the capital of Country 1

	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	capJur.RadiusSquared = 100.0
	capJur.IllegalActionIDs = 1 << components.InteractionTheft
	capJur.Corruption = 0 // Starts clean

	// 2. Sub-Village (part of Country 1)
	// We place it at distSq = 20000.
	// MaxAdministrativeRange = 150, so Max distSq = 22500.
	// At distSq 20000, it is normally safely within the empire.
	villEnt := world.NewEntity(posID, affID, villID)
	villPos := (*components.Position)(world.Get(villEnt, posID))
	villPos.X, villPos.Y = 100.0, 100.0 // dx=100, dy=100 -> distSq=20000

	villAff := (*components.Affiliation)(world.Get(villEnt, affID))
	villAff.CityID = 2
	villAff.CountryID = 1

	// 3. The Guard (Enforcing Capital laws)
	guardEnt := world.NewEntity(posID, affID, jobID, pathID, velID)
	gPos := (*components.Position)(world.Get(guardEnt, posID))
	gPos.X, gPos.Y = 5, 5 // Near the criminal

	gAff := (*components.Affiliation)(world.Get(guardEnt, affID))
	gAff.CityID = 1 // Works for the Capital

	gJob := (*components.JobComponent)(world.Get(guardEnt, jobID))
	gJob.JobID = components.JobGuard

	gPath := (*components.Path)(world.Get(guardEnt, pathID))
	gPath.HasPath = false

	// 4. The Wealthy Criminal
	crimEnt := world.NewEntity(posID, affID, needsID, crimeID, velID)
	cPos := (*components.Position)(world.Get(crimEnt, posID))
	cPos.X, cPos.Y = 5, 5 // Extremely close to guard (distSq < 2.0)

	cAff := (*components.Affiliation)(world.Get(crimEnt, affID))
	cAff.CityID = 1 // Resident of the capital

	cNeeds := (*components.Needs)(world.Get(crimEnt, needsID))
	cNeeds.Wealth = 5000.0 // Very rich, easily affords bribe

	cCrime := (*components.CrimeMarker)(world.Get(crimEnt, crimeID))
	cCrime.Bounty = 100
	cCrime.CrimeLevel = 1

	// --- ACT 1: THE BRIBE (JusticeSystem) ---

	// The Guard intercepts the criminal. Since criminal is rich, they should bribe.
	justiceSys.Update(&world)

	// Verify Bribery Outcomes
	if world.Has(crimEnt, crimeID) {
		t.Errorf("Expected CrimeMarker to be removed due to bribery, but it remained.")
	}

	cNeedsAfter := (*components.Needs)(world.Get(crimEnt, needsID))
	if cNeedsAfter.Wealth >= 5000.0 {
		t.Errorf("Expected criminal wealth to decrease after bribe, got %v", cNeedsAfter.Wealth)
	}

	cAffAfter := (*components.Affiliation)(world.Get(crimEnt, affID))
	if cAffAfter.CityID == 0 {
		t.Errorf("Expected criminal NOT to be banished, but CityID was wiped.")
	}

	capJurAfter := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	if capJurAfter.Corruption == 0 {
		t.Errorf("Expected Capital Corruption to increment due to bribe.")
	}

	// --- ACT 2: SIMULATING SYSTEMIC ROT ---
	// Let's artificially boost Corruption to simulate years of this happening.
	capJurAfter.Corruption = 20

	// --- ACT 3: THE BUTTERFLY EFFECT (AdministrativeFractureSystem) ---

	// Normally, a village at distSq=20000 is safe (Max distSq=22500).
	// With Corruption=20, penalty = 20 * 0.1 = 2.0.
	// EffectiveDistSq = 20000 * (1.0 + 2.0) = 60000.
	// 60000 > 22500, so it should fracture!

	// Run system exactly 1000 times to trigger the tick modulus
	for i := 0; i < 1000; i++ {
		adminSys.Update(&world)
	}

	// Verify Fracture
	villAffAfter := (*components.Affiliation)(world.Get(villEnt, affID))
	if villAffAfter.CountryID != 0 {
		t.Errorf("Expected distant Village to secede due to Capital Corruption multiplier, but it remained loyal.")
	}
}
