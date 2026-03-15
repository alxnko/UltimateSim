package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 30.1: Ideological Economy (The Tithe Engine) Tests

func TestTitheSystem(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	beliefID := ecs.ComponentID[components.BeliefComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	npcID := ecs.ComponentID[components.NPC](&world)

	// Create a Preacher
	preacher := world.NewEntity(posID, jobID, beliefID, needsID)
	pPos := (*components.Position)(world.Get(preacher, posID))
	pPos.X, pPos.Y = 10, 10

	pJob := (*components.JobComponent)(world.Get(preacher, jobID))
	pJob.JobID = components.JobPreacher

	pBelief := (*components.BeliefComponent)(world.Get(preacher, beliefID))
	pBelief.Beliefs = []components.Belief{
		{BeliefID: 100, Weight: 50},
	}

	pNeeds := (*components.Needs)(world.Get(preacher, needsID))
	pNeeds.Wealth = 0

	// Create a devout NPC nearby
	npc1 := world.NewEntity(npcID, posID, beliefID, needsID)
	n1Pos := (*components.Position)(world.Get(npc1, posID))
	n1Pos.X, n1Pos.Y = 12, 12 // distSq = 8, < 25.0

	n1Belief := (*components.BeliefComponent)(world.Get(npc1, beliefID))
	n1Belief.Beliefs = []components.Belief{
		{BeliefID: 100, Weight: 50},
	}

	n1Needs := (*components.Needs)(world.Get(npc1, needsID))
	n1Needs.Wealth = 100

	// Create a devout NPC far away
	npc2 := world.NewEntity(npcID, posID, beliefID, needsID)
	n2Pos := (*components.Position)(world.Get(npc2, posID))
	n2Pos.X, n2Pos.Y = 20, 20 // distSq = 200, > 25.0

	n2Belief := (*components.BeliefComponent)(world.Get(npc2, beliefID))
	n2Belief.Beliefs = []components.Belief{
		{BeliefID: 100, Weight: 50},
	}

	n2Needs := (*components.Needs)(world.Get(npc2, needsID))
	n2Needs.Wealth = 100

	// Create an NPC of a different belief nearby
	npc3 := world.NewEntity(npcID, posID, beliefID, needsID)
	n3Pos := (*components.Position)(world.Get(npc3, posID))
	n3Pos.X, n3Pos.Y = 10, 11 // distSq = 1, < 25.0

	n3Belief := (*components.BeliefComponent)(world.Get(npc3, beliefID))
	n3Belief.Beliefs = []components.Belief{
		{BeliefID: 200, Weight: 50}, // Different BeliefID
	}

	n3Needs := (*components.Needs)(world.Get(npc3, needsID))
	n3Needs.Wealth = 100

	sys := NewTitheSystem(&world)

	// Tick the system 49 times (no tithe)
	for i := 0; i < 49; i++ {
		sys.Update(&world)
	}

	if pNeeds.Wealth != 0 {
		t.Errorf("Preacher wealth should be 0 before tick 50, got %f", pNeeds.Wealth)
	}
	if n1Needs.Wealth != 100 {
		t.Errorf("NPC1 wealth should be 100 before tick 50, got %f", n1Needs.Wealth)
	}

	// Tick the system the 50th time (tithe collected)
	sys.Update(&world)

	// NPC1 is near and shares belief -> pays 10% tithe (10.0)
	if n1Needs.Wealth != 90.0 {
		t.Errorf("Expected NPC1 wealth to be 90.0 after tithe, got %f", n1Needs.Wealth)
	}

	// NPC2 is far away -> no tithe
	if n2Needs.Wealth != 100.0 {
		t.Errorf("Expected NPC2 wealth to be 100.0, got %f", n2Needs.Wealth)
	}

	// NPC3 has different belief -> no tithe
	if n3Needs.Wealth != 100.0 {
		t.Errorf("Expected NPC3 wealth to be 100.0, got %f", n3Needs.Wealth)
	}

	// Preacher collects tithe from NPC1
	if pNeeds.Wealth != 10.0 {
		t.Errorf("Expected Preacher wealth to be 10.0 after tithes collected, got %f", pNeeds.Wealth)
	}
}
