package systems

import (
	"testing"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestJusticeSystem_DetectionAndContraband(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewJusticeSystem(&world)

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
	sys := NewJusticeSystem(&world)

	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)

	// Create Capital with Jurisdiction to satisfy empty return check
	capEnt := world.NewEntity(posID, affID, jurID)
	capJur := (*components.JurisdictionComponent)(world.Get(capEnt, jurID))
	capJur.RadiusSquared = 10000.0 // Huge radius

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
