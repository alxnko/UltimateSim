package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestParasiteSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()
	sys := NewParasiteSystem(&world, hooks)

	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	parasiteID := ecs.ComponentID[components.ParasiteComponent](&world)

	// Create Victim
	victimEnt := world.NewEntity(posID, vitalsID, identID, needsID)
	vPos := (*components.Position)(world.Get(victimEnt, posID))
	vPos.X = 10.0
	vPos.Y = 10.0
	vVitals := (*components.VitalsComponent)(world.Get(victimEnt, vitalsID))
	vVitals.Blood = 100.0
	vVitals.Pain = 0.0
	vIdent := (*components.Identity)(world.Get(victimEnt, identID))
	vIdent.ID = 101

	// Create Parasite
	parasiteEnt := world.NewEntity(posID, needsID, identID, pathID, parasiteID)
	pPos := (*components.Position)(world.Get(parasiteEnt, posID))
	pPos.X = 10.0 // Right next to victim
	pPos.Y = 10.0
	pNeeds := (*components.Needs)(world.Get(parasiteEnt, needsID))
	pNeeds.Food = 10.0 // Hungry
	pIdent := (*components.Identity)(world.Get(parasiteEnt, identID))
	pIdent.ID = 202

	// Run Update
	sys.Update(&world)

	// Assertions

	// 1. Victim's blood drained, pain spiked
	if vVitals.Blood != 60.0 {
		t.Errorf("Expected victim blood to be 60.0, got %f", vVitals.Blood)
	}
	if vVitals.Pain != 40.0 {
		t.Errorf("Expected victim pain to be 40.0, got %f", vVitals.Pain)
	}

	// 2. Parasite's food restored
	if pNeeds.Food != 60.0 {
		t.Errorf("Expected parasite food to be 60.0, got %f", pNeeds.Food)
	}

	// 3. Grudge Hook added from victim to parasite
	hookVal := hooks.GetHook(vIdent.ID, pIdent.ID)
	if hookVal != -100 {
		t.Errorf("Expected hook from victim to parasite to be -100, got %d", hookVal)
	}

	// 4. Parasite marked with EsotericTrait
	if pIdent.BaseTraits&components.TraitEsoteric == 0 {
		t.Errorf("Expected parasite to be marked with TraitEsoteric")
	}
}
