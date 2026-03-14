package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

func TestPreacherSystem(t *testing.T) {
	// Setup world
	world := ecs.NewWorld()

	preacherSystem := NewPreacherSystem(&world)

	// Create Preacher
	preacherEnt := world.NewEntity()
	world.Add(preacherEnt,
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
	)

	preacherPos := (*components.Position)(world.Get(preacherEnt, ecs.ComponentID[components.Position](&world)))
	preacherJob := (*components.JobComponent)(world.Get(preacherEnt, ecs.ComponentID[components.JobComponent](&world)))
	preacherBelief := (*components.BeliefComponent)(world.Get(preacherEnt, ecs.ComponentID[components.BeliefComponent](&world)))

	preacherPos.X = 10.0
	preacherPos.Y = 10.0
	preacherJob.JobID = components.JobPreacher

	// Preacher believes in Belief 1 with high weight
	preacherBelief.Beliefs = append(preacherBelief.Beliefs, components.Belief{
		BeliefID: 1,
		Weight:   10,
	})

	// Create Target
	targetEnt := world.NewEntity()
	world.Add(targetEnt,
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.BeliefComponent](&world),
	)

	targetPos := (*components.Position)(world.Get(targetEnt, ecs.ComponentID[components.Position](&world)))
	targetBelief := (*components.BeliefComponent)(world.Get(targetEnt, ecs.ComponentID[components.BeliefComponent](&world)))

	// Position Target within preacher's range (distSq < 400.0)
	targetPos.X = 25.0 // distSq = 15*15 = 225.0
	targetPos.Y = 10.0

	// Target believes in Belief 2
	targetBelief.Beliefs = append(targetBelief.Beliefs, components.Belief{
		BeliefID: 2,
		Weight:   5,
	})

	// Tick the system to 50
	for i := 0; i < 50; i++ {
		preacherSystem.Update(&world)
	}

	// Verify target now has Belief 1
	foundBelief1 := false
	for _, b := range targetBelief.Beliefs {
		if b.BeliefID == 1 {
			foundBelief1 = true
			if b.Weight != 5 {
				t.Errorf("Expected Belief 1 weight to be 5, got %d", b.Weight)
			}
		}
		if b.BeliefID == 2 {
			if b.Weight != 4 { // Was 5, suppressed by 1
				t.Errorf("Expected Belief 2 weight to be suppressed to 4, got %d", b.Weight)
			}
		}
	}

	if !foundBelief1 {
		t.Errorf("Target did not receive Preacher's belief")
	}

	// Move target out of range
	targetPos.X = 100.0

	// Tick another 50 times
	for i := 0; i < 50; i++ {
		preacherSystem.Update(&world)
	}

	// Verify belief hasn't increased since it's out of range
	for _, b := range targetBelief.Beliefs {
		if b.BeliefID == 1 {
			if b.Weight != 5 {
				t.Errorf("Expected Belief 1 weight to remain 5 when out of range, got %d", b.Weight)
			}
		}
	}
}
