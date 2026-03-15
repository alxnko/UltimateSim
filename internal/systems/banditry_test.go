package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestBanditrySystem_Integration tests the "Butterfly Effect":
// Desperation/Economy -> Banditry -> Logistics (Caravan Destroyed) -> Justice (CrimeMarker)
func TestBanditrySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	sys := NewBanditrySystem(&world)

	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)

	cPosID := ecs.ComponentID[components.Position](&world)
	caravanID := ecs.ComponentID[components.Caravan](&world)
	payloadID := ecs.ComponentID[components.Payload](&world)

	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	// 1. Create a Desperate NPC
	npc := world.NewEntity(posID, needsID, despID, memID, jobID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X, npcPos.Y = 10, 10

	npcNeeds := (*components.Needs)(world.Get(npc, needsID))
	npcNeeds.Food = 0
	npcNeeds.Wealth = 0

	npcDesp := (*components.DesperationComponent)(world.Get(npc, despID))
	npcDesp.Level = 60 // Over 50 triggers banditry conversion

	npcJob := (*components.JobComponent)(world.Get(npc, jobID))
	npcJob.JobID = components.JobFarmer // Initial job

	// 2. Create a Caravan nearby
	caravan := world.NewEntity(cPosID, caravanID, payloadID)
	cPos := (*components.Position)(world.Get(caravan, cPosID))
	cPos.X, cPos.Y = 11, 10 // distSq = 1.0 (< 2.0 triggers robbery)

	cPayload := (*components.Payload)(world.Get(caravan, payloadID))
	cPayload.Food = 100
	cPayload.Wood = 50

	// 3. Run System Update
	sys.Update(&world)

	// 4. Verification: Butterfly Effect checks

	// A. Did the NPC become a Bandit?
	newJob := (*components.JobComponent)(world.Get(npc, jobID))
	if newJob.JobID != components.JobBandit {
		t.Errorf("Expected NPC to convert to JobBandit (7), got %d", newJob.JobID)
	}

	// B. Was the Caravan destroyed?
	if world.Alive(caravan) {
		t.Errorf("Expected Caravan to be destroyed after robbery")
	}

	// C. Did the Bandit receive the payload?
	newNeeds := (*components.Needs)(world.Get(npc, needsID))
	if newNeeds.Food != 100 {
		t.Errorf("Expected Bandit to receive 100 Food, got %f", newNeeds.Food)
	}
	if newNeeds.Wealth != 50 {
		t.Errorf("Expected Bandit to receive 50 Wealth (converted from Wood), got %f", newNeeds.Wealth)
	}

	// D. Did Desperation reset?
	newDesp := (*components.DesperationComponent)(world.Get(npc, despID))
	if newDesp.Level != 0 {
		t.Errorf("Expected Desperation to reset to 0, got %d", newDesp.Level)
	}

	// E. Was Justice triggered? (CrimeMarker added)
	if !world.Has(npc, crimeID) {
		t.Errorf("Expected Bandit to receive a CrimeMarker")
	} else {
		cm := (*components.CrimeMarker)(world.Get(npc, crimeID))
		if cm.Bounty != 250 {
			t.Errorf("Expected Bandit Bounty to be 250, got %d", cm.Bounty)
		}
	}

	// F. Was Memory updated with Theft?
	mem := (*components.Memory)(world.Get(npc, memID))
	hasTheftMem := false
	for _, event := range mem.Events {
		if event.InteractionType == components.InteractionTheft {
			hasTheftMem = true
			if event.Value != 100 {
				t.Errorf("Expected theft memory value to be 100 (Food), got %d", event.Value)
			}
			break
		}
	}
	if !hasTheftMem {
		t.Errorf("Expected Memory to contain InteractionTheft")
	}
}
