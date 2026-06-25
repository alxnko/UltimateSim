package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestCannibalismSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	sanityID := ecs.ComponentID[components.SanityComponent](&world)
	corpseID := ecs.ComponentID[components.CorpseComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	parasiteID := ecs.ComponentID[components.ParasiteComponent](&world)
	memID := ecs.ComponentID[components.Memory](&world)

	pathQueue := engine.NewPathRequestQueue(100, 0)
	sys := NewCannibalismSystem(&world, pathQueue)

	// Create a starving, desperate NPC next to a corpse
	npcEnt := world.NewEntity(posID, needsID, despID, sanityID, identID, memID)

	nPos := (*components.Position)(world.Get(npcEnt, posID))
	nPos.X = 10.0
	nPos.Y = 10.0

	nNeeds := (*components.Needs)(world.Get(npcEnt, needsID))
	nNeeds.Food = 10.0

	nDesp := (*components.DesperationComponent)(world.Get(npcEnt, despID))
	nDesp.Level = 80

	nSanity := (*components.SanityComponent)(world.Get(npcEnt, sanityID))
	nSanity.MaxStress = 100.0

	// Create the corpse
	corpseEnt := world.NewEntity(posID, corpseID)
	cPos := (*components.Position)(world.Get(corpseEnt, posID))
	cPos.X = 11.0
	cPos.Y = 10.0

	// Tick the system multiple times to ensure deterministic consumption
	for i := 0; i < 5; i++ {
		sys.Update(&world)
	}

	// 1. Assert corpse is destroyed
	if world.Alive(corpseEnt) {
		t.Errorf("Expected corpse to be consumed and destroyed, but it is still alive")
	}

	// 2. Assert needs restored
	updatedNeeds := (*components.Needs)(world.Get(npcEnt, needsID))
	if updatedNeeds.Food != 60.0 {
		t.Errorf("Expected food to be 60.0 after consuming corpse, got %v", updatedNeeds.Food)
	}

	// 3. Assert psychological trauma
	updatedSanity := (*components.SanityComponent)(world.Get(npcEnt, sanityID))
	if updatedSanity.Stress != 50.0 {
		t.Errorf("Expected stress to spike to 50.0, got %v", updatedSanity.Stress)
	}

	// 4. Assert desperation reset
	updatedDesp := (*components.DesperationComponent)(world.Get(npcEnt, despID))
	if updatedDesp.Level != 0 {
		t.Errorf("Expected desperation to reset to 0, got %v", updatedDesp.Level)
	}

	// 5. Assert parasite symbiosis triggered
	if !world.Has(npcEnt, parasiteID) {
		t.Errorf("Expected NPC to contract ParasiteComponent after cannibalism")
	} else {
		parasite := (*components.ParasiteComponent)(world.Get(npcEnt, parasiteID))
		if parasite.BloodSatiety != 100.0 || !parasite.IsHidden {
			t.Errorf("Expected Parasite to be fully sated and hidden")
		}
	}

	// 6. Assert esoteric identity trait
	ident := (*components.Identity)(world.Get(npcEnt, identID))
	if ident.BaseTraits&components.TraitEsoteric == 0 {
		t.Errorf("Expected NPC to gain TraitEsoteric")
	}

	// 7. Assert esoteric memory log
	mem := (*components.Memory)(world.Get(npcEnt, memID))
	foundEsotericMem := false
	for _, e := range mem.Events {
		if e.InteractionType == components.InteractionEsoteric {
			foundEsotericMem = true
			break
		}
	}
	if !foundEsotericMem {
		t.Errorf("Expected InteractionEsoteric to be logged in memory")
	}
}
