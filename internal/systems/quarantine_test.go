package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 37.1: The Quarantine Engine Testing
// Demonstrates Butterfly Effect: Disease -> Quarantine -> NPC crosses border -> CrimeMarker -> Guard fines/banishes

func TestQuarantineButterflyEffect(t *testing.T) {
	world := ecs.NewWorld()
	hookGraph := engine.NewSparseHookGraph()

	// Instantiate Systems
	quarantineSys := NewQuarantineSystem(&world)
	justiceSys := NewJusticeSystem(&world, hookGraph)

	// Fetch IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](&world)
	quarID := ecs.ComponentID[components.QuarantineComponent](&world)
	diseaseID := ecs.ComponentID[components.DiseaseEntity](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	idID := ecs.ComponentID[components.Identity](&world)

	// 1. Create a Capital with Jurisdiction
	capital := world.NewEntity(posID, affID, jurID)
	capPos := (*components.Position)(world.Get(capital, posID))
	capPos.X, capPos.Y = 50, 50

	capAff := (*components.Affiliation)(world.Get(capital, affID))
	capAff.CityID = 1

	capJur := (*components.JurisdictionComponent)(world.Get(capital, jurID))
	capJur.RadiusSquared = 100.0 // Radius 10

	// Verify no quarantine active initially
	if world.Has(capital, quarID) {
		t.Fatalf("Expected no quarantine active initially")
	}

	// 2. Spawn a Disease inside the Jurisdiction
	disease := world.NewEntity(posID, diseaseID)
	dPos := (*components.Position)(world.Get(disease, posID))
	dPos.X, dPos.Y = 52, 52 // Inside the radius

	// 3. Update Quarantine System
	// We need to run it 20 times due to the modulo throttle in Update()
	for i := 0; i < 20; i++ {
		quarantineSys.Update(&world)
	}

	// Verify Quarantine was enacted
	if !world.Has(capital, quarID) {
		t.Fatalf("Expected QuarantineComponent to be added to Capital")
	}

	quarComp := (*components.QuarantineComponent)(world.Get(capital, quarID))
	if !quarComp.Active {
		t.Fatalf("Expected QuarantineComponent to be Active")
	}

	// 4. Create an NPC inside the Quarantine trying to leave
	npc := world.NewEntity(posID, affID, memID, pathID, needsID, velID, idID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X, npcPos.Y = 55, 55 // Inside radius

	npcPath := (*components.Path)(world.Get(npc, pathID))
	npcPath.TargetX = 70
	npcPath.TargetY = 70 // Outside radius (dx=20, dy=20 => distSq=800)

	npcNeeds := (*components.Needs)(world.Get(npc, needsID))
	npcNeeds.Wealth = 50.0 // Has some wealth for fines

	npcID := (*components.Identity)(world.Get(npc, idID))
	npcID.ID = 100

	npcAff := (*components.Affiliation)(world.Get(npc, affID))
	npcAff.CityID = 1

	// 5. Update Justice System
	justiceSys.Update(&world)

	// Verify NPC got a CrimeMarker for trying to leave Quarantine
	if !world.Has(npc, crimeID) {
		t.Fatalf("Expected NPC to receive a CrimeMarker for breaking Quarantine")
	}

	cMarker := (*components.CrimeMarker)(world.Get(npc, crimeID))
	if cMarker.Bounty != 100 { // Default Bounty from JusticeSystem
		t.Errorf("Expected Bounty 100, got %d", cMarker.Bounty)
	}

	// 6. Spawn a Guard nearby
	guard := world.NewEntity(posID, affID, jobID, pathID, velID, idID)
	gPos := (*components.Position)(world.Get(guard, posID))
	gPos.X, gPos.Y = 55, 54 // Adjacent to NPC (distSq < 2.0)

	gJob := (*components.JobComponent)(world.Get(guard, jobID))
	gJob.JobID = components.JobGuard

	gAff := (*components.Affiliation)(world.Get(guard, affID))
	gAff.CityID = 1

	gID := (*components.Identity)(world.Get(guard, idID))
	gID.ID = 200

	// 7. Update Justice System to enforce punishment
	justiceSys.Update(&world)

	// Verify Punishment (Banishment & Fine & Hook)
	if world.Has(npc, crimeID) {
		t.Errorf("Expected CrimeMarker to be removed after punishment")
	}

	// Wait, we need to fetch the pointer again because arche structural changes might invalidate it
	npcNeedsNew := (*components.Needs)(world.Get(npc, needsID))
	if npcNeedsNew.Wealth != 0.0 { // 50 - 100 = 0 (clamped)
		t.Errorf("Expected NPC wealth to be 0 after fine, got %f", npcNeedsNew.Wealth)
	}

	// Wait, we need to fetch the pointer again because arche structural changes might invalidate it
	// Phase 45 introduced Penal Labor which prevents banishment if fine is unpaid.
	// In this test, Wealth=50, Bounty=100 -> UnpaidFine=50.
	// Therefore, the NPC is sent to Penal Labor instead of Banishment.
	penalID := ecs.ComponentID[components.PenalLaborComponent](&world)
	if !world.Has(npc, penalID) {
		t.Errorf("Expected NPC to be sentenced to Penal Labor due to unpaid fine")
	} else {
		penal := (*components.PenalLaborComponent)(world.Get(npc, penalID))
		if penal.RemainingSentence != 250 { // 50 * 5 = 250 ticks
			t.Errorf("Expected 250 ticks of penal labor, got %d", penal.RemainingSentence)
		}
	}

	npcAffNew := (*components.Affiliation)(world.Get(npc, affID))
	if npcAffNew.GuildID != 1 {
		t.Errorf("Expected NPC's GuildID to be seized by the state (1), got %d", npcAffNew.GuildID)
	}

	hookScore := hookGraph.GetHook(100, 200)
	if hookScore != -50 {
		t.Errorf("Expected NPC to form a -50 Blood Feud hook against the Guard, got %d", hookScore)
	}
}
