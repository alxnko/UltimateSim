package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 65 - The Physical Sanitation Engine
// Tests that Death leaves a Corpse, Corpses cause Stress, and Corpses decay into Disease.
func TestSanitationSystem_Integration(t *testing.T) {
	engine.InitializeRNG([32]byte{42})

	world := ecs.NewWorld()

	// Register all needed components
	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	sanityID := ecs.ComponentID[components.SanityComponent](&world)
	corpseID := ecs.ComponentID[components.CorpseComponent](&world)
	diseaseID := ecs.ComponentID[components.DiseaseEntity](&world)

	deathSys := NewDeathSystem(&world, nil)
	sanitationSys := NewSanitationSystem(&world)
	mentalBreakSys := NewMentalBreakSystem(&world)

	// Entity 1: The Starving NPC
	dyingNPC := world.NewEntity(posID, needsID, vitalsID)
	dPos := (*components.Position)(world.Get(dyingNPC, posID))
	dPos.X = 10
	dPos.Y = 10

	needs := (*components.Needs)(world.Get(dyingNPC, needsID))
	needs.Food = 0 // Starving, will trigger DeathSystem

	// Entity 2: The Innocent Bystander
	bystander := world.NewEntity(posID, sanityID, vitalsID)
	bPos := (*components.Position)(world.Get(bystander, posID))
	bPos.X = 12
	bPos.Y = 12 // DistSq = 8, within the 25.0 radius

	sanity := (*components.SanityComponent)(world.Get(bystander, sanityID))
	sanity.MaxStress = 5.0
	sanity.Stress = 0.0

	// 1. Trigger Death
	deathSys.Update(&world)

	if world.Alive(dyingNPC) {
		t.Fatal("Expected dying NPC to be dead")
	}

	// 2. Verify Corpse Spawned
	corpseQuery := world.Query(ecs.All(corpseID, posID))
	corpseCount := 0
	var corpseEnt ecs.Entity
	for corpseQuery.Next() {
		corpseCount++
		corpseEnt = corpseQuery.Entity()
	}
	if corpseCount != 1 {
		t.Fatalf("Expected 1 corpse to spawn, got %d", corpseCount)
	}

	// 3. Trigger Sanitation (Psychological Impact)
	// Corpse decays 1 per tick. Stress increases 0.5 per tick for bystander.
	for i := 0; i < 11; i++ { // Need stress to pass 5.0. 11 ticks = 5.5 stress
		sanitationSys.Update(&world)
	}

	sanity = (*components.SanityComponent)(world.Get(bystander, sanityID))
	if sanity.Stress < 5.0 {
		t.Fatalf("Expected bystander stress to spike above 5.0, got %f", sanity.Stress)
	}

	// 4. Trigger Mental Break
	// Need to sync tickCounter for MentalBreakSystem (it triggers every 10 ticks)
	for i := 0; i < 10; i++ {
		mentalBreakSys.Update(&world)
	}

	sanity = (*components.SanityComponent)(world.Get(bystander, sanityID))
	if sanity.BreakState == components.BreakNormal {
		t.Fatal("Expected bystander to suffer a mental break from corpse stress")
	}

	// 5. Trigger Biological Impact (Disease Spawn)
	// Corpse max decay is 100. We've done 11 ticks. Need 89 more.
	for i := 0; i < 90; i++ {
		sanitationSys.Update(&world)
	}

	if world.Alive(corpseEnt) {
		t.Fatal("Expected corpse to be removed after full decay")
	}

	diseaseQuery := world.Query(ecs.All(diseaseID, posID))
	diseaseCount := 0
	for diseaseQuery.Next() {
		diseaseCount++
	}

	if diseaseCount != 1 {
		t.Fatalf("Expected 1 DiseaseEntity to spawn from decayed corpse, got %d", diseaseCount)
	}
}
