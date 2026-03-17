package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 39.1: The Courier Interception Engine Testing (The Butterfly Effect)
// Tests that Bandits successfully intercept OrderEntities, preventing administrative execution,
// while simultaneously generating Justice System triggers and wealth accumulation.

func TestCourierInterceptionSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)
	needsID := ecs.ComponentID[components.Needs](&world)

	orderEntityID := ecs.ComponentID[components.OrderEntity](&world)
	orderCompID := ecs.ComponentID[components.OrderComponent](&world)
	crimeID := ecs.ComponentID[components.CrimeMarker](&world)

	// 1. Create a Bandit
	bandit := world.NewEntity(posID, jobID, memID, needsID)
	banditPos := (*components.Position)(world.Get(bandit, posID))
	banditJob := (*components.JobComponent)(world.Get(bandit, jobID))
	banditNeeds := (*components.Needs)(world.Get(bandit, needsID))

	banditPos.X = 10.0
	banditPos.Y = 10.0
	banditJob.JobID = components.JobBandit
	banditNeeds.Wealth = 0.0

	// 2. Create an OrderEntity directly on top of the Bandit (distSq = 0.0)
	order1 := world.NewEntity(posID, orderEntityID, orderCompID)
	order1Pos := (*components.Position)(world.Get(order1, posID))
	order1Comp := (*components.OrderComponent)(world.Get(order1, orderCompID))

	order1Pos.X = 10.0
	order1Pos.Y = 10.0
	order1Comp.TargetCityID = 42

	// 3. Create another OrderEntity far away (distSq = 100.0)
	order2 := world.NewEntity(posID, orderEntityID, orderCompID)
	order2Pos := (*components.Position)(world.Get(order2, posID))

	order2Pos.X = 20.0
	order2Pos.Y = 10.0

	// Initialize System
	sys := NewCourierInterceptionSystem(&world)

	// Run System
	sys.Update(&world)

	// --- ASSERTIONS ---

	// A. Did the bandit intercept the close order? (Order1 should be destroyed)
	if world.Alive(order1) {
		t.Errorf("Expected Order1 to be intercepted and destroyed by the Bandit.")
	}

	// B. Did the bandit ignore the far order? (Order2 should survive)
	if !world.Alive(order2) {
		t.Errorf("Expected Order2 to survive since it was out of range of the Bandit.")
	}

	// C. Did the bandit gain wealth from intercepting the state secrets?
	banditNeedsAfter := (*components.Needs)(world.Get(bandit, needsID))
	if banditNeedsAfter.Wealth != 50.0 {
		t.Errorf("Expected Bandit wealth to spike to 50.0 after intercepting state secrets, got %f", banditNeedsAfter.Wealth)
	}

	// D. The Butterfly Effect: Was the Justice System triggered? (CrimeMarker added)
	if !world.Has(bandit, crimeID) {
		t.Fatalf("Expected Bandit to receive a CrimeMarker for intercepting a royal courier.")
	}

	cm := (*components.CrimeMarker)(world.Get(bandit, crimeID))
	if cm.Bounty != 500 {
		t.Errorf("Expected a massive State Bounty of 500 for the crime, got %d", cm.Bounty)
	}

	// E. The Butterfly Effect: Did the Bandit log the InteractionTheft?
	banditMem := (*components.Memory)(world.Get(bandit, memID))

	// Check previous memory slot since head advanced
	prevHead := banditMem.Head - 1
	if prevHead > 255 {
		prevHead = uint8(len(banditMem.Events)) - 1
	}

	event := banditMem.Events[prevHead]
	if event.InteractionType != components.InteractionTheft {
		t.Errorf("Expected Bandit memory buffer to log InteractionTheft (4), got %d", event.InteractionType)
	}
	if event.Value != 42 {
		t.Errorf("Expected Bandit memory to record the TargetCityID (42) as the stolen data value, got %d", event.Value)
	}
}

// Ensure completely deterministic executions across multiple seeds
func TestCourierInterceptionSystem_Determinism(t *testing.T) {
	world1 := ecs.NewWorld()
	world2 := ecs.NewWorld()

	// Seed identical states
	var seedA [32]byte
	seedA[0] = 1
	engine.InitializeRNG(seedA)

	// Create components...
	posID1 := ecs.ComponentID[components.Position](&world1)
	jobID1 := ecs.ComponentID[components.JobComponent](&world1)
	memID1 := ecs.ComponentID[components.Memory](&world1)
	needsID1 := ecs.ComponentID[components.Needs](&world1)
	oEntityID1 := ecs.ComponentID[components.OrderEntity](&world1)
	oCompID1 := ecs.ComponentID[components.OrderComponent](&world1)

	b1 := world1.NewEntity(posID1, jobID1, memID1, needsID1)
	(*components.Position)(world1.Get(b1, posID1)).X = float32(engine.GetRandomInt() % 100)
	(*components.JobComponent)(world1.Get(b1, jobID1)).JobID = components.JobBandit

	o1 := world1.NewEntity(posID1, oEntityID1, oCompID1)
	(*components.Position)(world1.Get(o1, posID1)).X = (*components.Position)(world1.Get(b1, posID1)).X + 1.0

	// Identical seed
	var seedB [32]byte
	seedB[0] = 1
	engine.InitializeRNG(seedB)

	posID2 := ecs.ComponentID[components.Position](&world2)
	jobID2 := ecs.ComponentID[components.JobComponent](&world2)
	memID2 := ecs.ComponentID[components.Memory](&world2)
	needsID2 := ecs.ComponentID[components.Needs](&world2)
	oEntityID2 := ecs.ComponentID[components.OrderEntity](&world2)
	oCompID2 := ecs.ComponentID[components.OrderComponent](&world2)

	b2 := world2.NewEntity(posID2, jobID2, memID2, needsID2)
	(*components.Position)(world2.Get(b2, posID2)).X = float32(engine.GetRandomInt() % 100)
	(*components.JobComponent)(world2.Get(b2, jobID2)).JobID = components.JobBandit

	o2 := world2.NewEntity(posID2, oEntityID2, oCompID2)
	(*components.Position)(world2.Get(o2, posID2)).X = (*components.Position)(world2.Get(b2, posID2)).X + 1.0

	// Systems
	sys1 := NewCourierInterceptionSystem(&world1)
	sys2 := NewCourierInterceptionSystem(&world2)

	sys1.Update(&world1)
	sys2.Update(&world2)

	// Compare states
	if world1.Alive(o1) != world2.Alive(o2) {
		t.Fatalf("Determinism failure: State mismatch between World1 and World2 on Order destruction")
	}

	bNeeds1 := (*components.Needs)(world1.Get(b1, needsID1)).Wealth
	bNeeds2 := (*components.Needs)(world2.Get(b2, needsID2)).Wealth

	if bNeeds1 != bNeeds2 {
		t.Fatalf("Determinism failure: Wealth calculation drifted: %f vs %f", bNeeds1, bNeeds2)
	}
}
