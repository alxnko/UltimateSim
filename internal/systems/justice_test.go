package systems

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestJusticeSystem_DetectionAndContraband(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewJusticeSystem(&world, engine.NewSparseHookGraph())

	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	contraID := ecs.ComponentID[components.ContrabandComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	// Create Capital with Jurisdiction
	capEnt := world.NewEntity(posID, affID, jurID, contraID)
	capPos := (*components.Position)(world.Get(capEnt, posID))
	capPos.X, capPos.Y = 10, 10

	capAff := (*components.Affiliation)(world.Get(capEnt, affID))
	capAff.CityID = 1

	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	capJur.RadiusSquared = 100.0 // Radius 10
	capJur.IllegalActionIDs = 1 << components.InteractionAssault // Assault is illegal

	capContra := (*components.ContrabandComponent)(world.Get(capEnt, contraID))
	capContra.Contraband = 1 << components.ItemIron // Iron is contraband

	// Create NPC 1: Commits Assault inside Jurisdiction
	npc1 := world.NewEntity(posID, affID, memID, storID)
	npc1Pos := (*components.Position)(world.Get(npc1, posID))
	npc1Pos.X, npc1Pos.Y = 15, 15 // Inside radius (dx=5, dy=5, distSq=50)

	npc1Mem := (*components.Memory)(world.Get(npc1, memID))
	npc1Mem.Events[0] = components.MemoryEvent{InteractionType: components.InteractionAssault}

	// Create NPC 2: Commits Assault outside Jurisdiction
	npc2 := world.NewEntity(posID, affID, memID, storID)
	npc2Pos := (*components.Position)(world.Get(npc2, posID))
	npc2Pos.X, npc2Pos.Y = 30, 30 // Outside radius (dx=20, dy=20, distSq=800)

	npc2Mem := (*components.Memory)(world.Get(npc2, memID))
	npc2Mem.Events[0] = components.MemoryEvent{InteractionType: components.InteractionAssault}

	// Create NPC 3: Carries Contraband inside Jurisdiction
	npc3 := world.NewEntity(posID, affID, memID, storID)
	npc3Pos := (*components.Position)(world.Get(npc3, posID))
	npc3Pos.X, npc3Pos.Y = 12, 12 // Inside radius

	npc3Stor := (*components.StorageComponent)(world.Get(npc3, storID))
	npc3Stor.Iron = 10 // Contraband!

	// Run system
	sys.Update(&world)

	// Verify NPC 1 is caught
	if !world.Has(npc1, crimeID) {
		t.Errorf("Expected NPC1 to have CrimeMarker for assault inside jurisdiction")
	}

	// Verify NPC 2 is safe
	if world.Has(npc2, crimeID) {
		t.Errorf("Expected NPC2 to NOT have CrimeMarker since outside jurisdiction")
	}

	// Verify NPC 3 is caught for contraband
	if !world.Has(npc3, crimeID) {
		t.Errorf("Expected NPC3 to have CrimeMarker for contraband inside jurisdiction")
	}
}

func TestJusticeSystem_Sentencing(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewJusticeSystem(&world, engine.NewSparseHookGraph())

	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)

	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)

	// Create Capital with Jurisdiction to satisfy empty return check
	capEnt := world.NewEntity(posID, affID, jurID, treasuryID)
	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	capJur.RadiusSquared = 10000.0 // Huge radius
	capAff := (*components.Affiliation)(world.Get(capEnt, affID))
	capAff.CityID = 1

	capTreasury := (*components.TreasuryComponent)(world.Get(capEnt, treasuryID))
	capTreasury.Wealth = 500.0

	// Create Criminal (Tagged)
	// We do not give them Memory or Storage in this test, so the detection step
	// skips generating new markers for them, but we pre-tag them for the Guard step.
	criminal := world.NewEntity(posID, affID, crimeID, needsID, velID, idID)
	cPos := (*components.Position)(world.Get(criminal, posID))
	cPos.X, cPos.Y = 10, 10

	cAff := (*components.Affiliation)(world.Get(criminal, affID))
	cAff.CityID = 1

	cNeeds := (*components.Needs)(world.Get(criminal, needsID))
	// Set wealth low enough to avoid bribery (bribe threshold = Bounty * 2.0 = 200.0)
	cNeeds.Wealth = 150.0

	cCrime := (*components.CrimeMarker)(world.Get(criminal, crimeID))
	cCrime.Bounty = 100

	// Create Guard far away
	guard1 := world.NewEntity(posID, affID, jobID, pathID, velID)
	g1Pos := (*components.Position)(world.Get(guard1, posID))
	g1Pos.X, g1Pos.Y = 50, 50

	g1Job := (*components.JobComponent)(world.Get(guard1, jobID))
	g1Job.JobID = components.JobGuard

	g1Aff := (*components.Affiliation)(world.Get(guard1, affID))
	g1Aff.CityID = 1

	// Run system - Guard should target Criminal
	sys.Update(&world)

	if !world.Has(criminal, crimeID) {
		t.Errorf("Criminal should still have CrimeMarker since Guard is far")
	}

	g1Path := (*components.Path)(world.Get(guard1, pathID))
	if g1Path.TargetX != 10 || g1Path.TargetY != 10 {
		t.Errorf("Guard1 did not target criminal properly")
	}

	// Move Guard adjacent to Criminal to trigger punishment
	g1Pos.X, g1Pos.Y = 11, 10

	// Run system - Guard executes punishment
	sys.Update(&world)

	// Check execution
	if world.Has(criminal, crimeID) {
		t.Errorf("Criminal should no longer have CrimeMarker (punished)")
	}

	// cNeeds pointer might have been invalidated if archetype changed during Update
	cNeedsNew := (*components.Needs)(world.Get(criminal, needsID))

	if cNeedsNew.Wealth != 50.0 {
		t.Errorf("Criminal should have been fined 100 wealth, got %f", cNeedsNew.Wealth)
	}

	cAffNew := (*components.Affiliation)(world.Get(criminal, affID))
	if cAffNew.CityID != 0 {
		t.Errorf("Criminal should have been banished (CityID set to 0), got %d", cAffNew.CityID)
	}

	cVelNew := (*components.Velocity)(world.Get(criminal, velID))
	if cVelNew.X == 0 && cVelNew.Y == 0 {
		t.Errorf("Criminal should be fleeing (velocity set), got X: %f Y: %f", cVelNew.X, cVelNew.Y)
	}
}

func TestJusticeSystem_CarceralResentmentAndBlackmail(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewJusticeSystem(&world, hooks)

	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	secID := ecs.ComponentID[components.SecretComponent](&world)

	// Create Capital with Jurisdiction
	capEnt := world.NewEntity(posID, affID, jurID)
	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	capJur.RadiusSquared = 10000.0 // Huge radius

	// Create Criminal A (Poor, will be punished)
	crimA := world.NewEntity(posID, affID, crimeID, needsID, velID, idID, secID)
	cAPos := (*components.Position)(world.Get(crimA, posID))
	cAPos.X, cAPos.Y = 10, 10
	cAAff := (*components.Affiliation)(world.Get(crimA, affID))
	cAAff.CityID = 1
	cANeeds := (*components.Needs)(world.Get(crimA, needsID))
	cANeeds.Wealth = 50.0 // Too poor for 200 bribe
	cACrime := (*components.CrimeMarker)(world.Get(crimA, crimeID))
	cACrime.Bounty = 100
	cAId := (*components.Identity)(world.Get(crimA, idID))
	cAId.ID = 1001

	// Create Criminal B (Rich, will bribe)
	crimB := world.NewEntity(posID, affID, crimeID, needsID, velID, idID, secID)
	cBPos := (*components.Position)(world.Get(crimB, posID))
	cBPos.X, cBPos.Y = 12, 10
	cBAff := (*components.Affiliation)(world.Get(crimB, affID))
	cBAff.CityID = 1
	cBNeeds := (*components.Needs)(world.Get(crimB, needsID))
	cBNeeds.Wealth = 500.0 // Rich enough for 200 bribe
	cBCrime := (*components.CrimeMarker)(world.Get(crimB, crimeID))
	cBCrime.Bounty = 100
	cBId := (*components.Identity)(world.Get(crimB, idID))
	cBId.ID = 1002

	// Create Guard 1 next to Crim A
	guardA := world.NewEntity(posID, affID, jobID, pathID, velID, idID)
	gAPos := (*components.Position)(world.Get(guardA, posID))
	gAPos.X, gAPos.Y = 10, 10
	gAJob := (*components.JobComponent)(world.Get(guardA, jobID))
	gAJob.JobID = components.JobGuard
	gAAff := (*components.Affiliation)(world.Get(guardA, affID))
	gAAff.CityID = 1
	gAId := (*components.Identity)(world.Get(guardA, idID))
	gAId.ID = 2001

	// Create Guard 2 next to Crim B
	guardB := world.NewEntity(posID, affID, jobID, pathID, velID, idID)
	gBPos := (*components.Position)(world.Get(guardB, posID))
	gBPos.X, gBPos.Y = 12, 10
	gBJob := (*components.JobComponent)(world.Get(guardB, jobID))
	gBJob.JobID = components.JobGuard
	gBAff := (*components.Affiliation)(world.Get(guardB, affID))
	gBAff.CityID = 1
	gBId := (*components.Identity)(world.Get(guardB, idID))
	gBId.ID = 2002

	// Run system
	sys.Update(&world)

	// --- Asserts for Criminal A (Punished) ---
	if world.Has(crimA, crimeID) {
		t.Errorf("Criminal A should no longer have CrimeMarker (punished)")
	}
	hookA := hooks.GetHook(1001, 2001)
	if hookA != -50 {
		t.Errorf("Expected Criminal A to have -50 hook on Guard A, got %d", hookA)
	}

	// --- Asserts for Criminal B (Bribed) ---
	if world.Has(crimB, crimeID) {
		t.Errorf("Criminal B should no longer have CrimeMarker (bribed/cleared)")
	}
	hookB := hooks.GetHook(1002, 2002)
	if hookB != 50 {
		t.Errorf("Expected Criminal B to have 50 hook on Guard B, got %d", hookB)
	}

	// Check if secret was generated
	cBSecNew := (*components.SecretComponent)(world.Get(crimB, secID))
	if len(cBSecNew.Secrets) == 0 {
		t.Errorf("Expected Criminal B to have generated a Secret about Guard B")
	} else {
		sec := cBSecNew.Secrets[0]
		if sec.OriginID != 1002 {
			t.Errorf("Expected secret origin ID to be Criminal B (1002)")
		}
		// Check registry
		registry := engine.GetSecretRegistry()
		text, exists := registry.GetSecret(sec.SecretID)
		if !exists || text != "guard_2002_corrupted" {
			t.Errorf("Expected rumor 'guard_2002_corrupted' in registry, got '%s'", text)
		}
	}
}
