package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/stretchr/testify/assert"
)

func TestForgerySystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	// Add systems
	forgerySys := NewForgerySystem(&world, hooks)
	bloodFeudSys := NewBloodFeudSystem(&world, hooks)

	// Create a wealthy, low-intellect Business Owner
	ownerBuilder := ecs.NewBuilder(&world,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)
	ownerEntity := ownerBuilder.New()

	ownerIdent := (*components.Identity)(world.Get(ownerEntity, ecs.ComponentID[components.Identity](&world)))
	ownerIdent.ID = 100

	ownerGenome := (*components.GenomeComponent)(world.Get(ownerEntity, ecs.ComponentID[components.GenomeComponent](&world)))
	ownerGenome.Intellect = 50

	ownerPos := (*components.Position)(world.Get(ownerEntity, ecs.ComponentID[components.Position](&world)))
	ownerPos.X = 10.0
	ownerPos.Y = 10.0

	ownerNeeds := (*components.Needs)(world.Get(ownerEntity, ecs.ComponentID[components.Needs](&world)))
	ownerNeeds.Food = 100.0

	// Create a Business owned by the low-intellect owner
	busBuilder := ecs.NewBuilder(&world,
		ecs.ComponentID[components.BusinessComponent](&world),
	)
	busEntity := busBuilder.New()
	busComp := (*components.BusinessComponent)(world.Get(busEntity, ecs.ComponentID[components.BusinessComponent](&world)))
	busComp.OwnerID = 100

	// Create a desperate, high-intellect Forger
	forgerBuilder := ecs.NewBuilder(&world,
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.GenomeComponent](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)
	forgerEntity := forgerBuilder.New()

	forgerIdent := (*components.Identity)(world.Get(forgerEntity, ecs.ComponentID[components.Identity](&world)))
	forgerIdent.ID = 200

	forgerGenome := (*components.GenomeComponent)(world.Get(forgerEntity, ecs.ComponentID[components.GenomeComponent](&world)))
	forgerGenome.Intellect = 150

	forgerDesp := (*components.DesperationComponent)(world.Get(forgerEntity, ecs.ComponentID[components.DesperationComponent](&world)))
	forgerDesp.Level = 80 // Desperate

	forgerPos := (*components.Position)(world.Get(forgerEntity, ecs.ComponentID[components.Position](&world)))
	forgerPos.X = 10.5
	forgerPos.Y = 10.5

	forgerNeeds := (*components.Needs)(world.Get(forgerEntity, ecs.ComponentID[components.Needs](&world)))
	forgerNeeds.Food = 100.0

	forgerMem := (*components.Memory)(world.Get(forgerEntity, ecs.ComponentID[components.Memory](&world)))

	// Advance ticks to trigger forgery
	for i := 0; i < 100; i++ {
		forgerySys.Update(&world)
	}

	// 1. Check if Forgery succeeded
	assert.Equal(t, uint64(200), busComp.OwnerID, "Business ownership should have transferred to Forger (ID 200)")

	// 2. Check Memory Event
	memEventFound := false
	for _, ev := range forgerMem.Events {
		if ev.InteractionType == components.InteractionTheft && ev.TargetID == 100 {
			memEventFound = true
			break
		}
	}
	assert.True(t, memEventFound, "Forger should have logged InteractionTheft against Owner in Memory")

	// 3. Check SparseHookGraph Edge
	hookStrength := hooks.GetHook(100, 200) // Owner -> Forger
	assert.Equal(t, -100, hookStrength, "Owner should have a massive -100 grudge against the Forger")

	// 4. Test Butterfly Effect: BloodFeudSystem triggering
	// We run BloodFeudSystem. With a -100 hook, the owner should attempt a murder
	bloodFeudSys.Update(&world)

	// Since they are close (distSq < 4.0 for murder), BloodFeud should execute
	// InteractionMurder and attach a CombatMarker
	combatMarkerID := ecs.ComponentID[components.CombatMarker](&world)
	hasCombatMarker := world.Has(ownerEntity, combatMarkerID)

	if hasCombatMarker {
		marker := (*components.CombatMarker)(world.Get(ownerEntity, combatMarkerID))
		assert.Equal(t, uint64(200), marker.TargetID, "Owner should have a CombatMarker targeting the Forger")
	} else {
		t.Errorf("Owner did not receive a CombatMarker after BloodFeud execution")
	}
}
