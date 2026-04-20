package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 61 - Biological Sabotage Engine Integration Test
// The "Butterfly Effect" proving Biological Sabotage ties Hook Grudges, Economy, and Disease logic.

func TestBiologicalSabotageSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	// 1. Initialize Components to prevent arche-go unregistered panics
	ecs.ComponentID[components.Village](&world)
	ecs.ComponentID[components.StorageComponent](&world)
	ecs.ComponentID[components.Affiliation](&world)
	ecs.ComponentID[components.Position](&world)
	ecs.ComponentID[components.AdministrationMarker](&world)
	ecs.ComponentID[components.Identity](&world)
	ecs.ComponentID[components.NPC](&world)
	ecs.ComponentID[components.CrimeMarker](&world)
	ecs.ComponentID[components.DiseaseEntity](&world)

	hooks := engine.NewSparseHookGraph()
	sys := NewBiologicalSabotageSystem(&world, hooks)

	var cityID uint32 = 42
	var rulerID uint64 = 999
	var npcID uint64 = 111

	// 2. Create the Ruler (Target)
	rulerEnt := world.NewEntity(
		ecs.ComponentID[components.AdministrationMarker](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Identity](&world),
	)
	rulerAff := (*components.Affiliation)(world.Get(rulerEnt, ecs.ComponentID[components.Affiliation](&world)))
	rulerAff.CityID = cityID
	rulerIdent := (*components.Identity)(world.Get(rulerEnt, ecs.ComponentID[components.Identity](&world)))
	rulerIdent.ID = rulerID

	// 3. Create the Village
	villageEnt := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Position](&world),
	)
	villageAff := (*components.Affiliation)(world.Get(villageEnt, ecs.ComponentID[components.Affiliation](&world)))
	villageAff.CityID = cityID

	villageStor := (*components.StorageComponent)(world.Get(villageEnt, ecs.ComponentID[components.StorageComponent](&world)))
	villageStor.Food = 100 // Starting food

	villagePos := (*components.Position)(world.Get(villageEnt, ecs.ComponentID[components.Position](&world)))
	villagePos.X = 10.0
	villagePos.Y = 10.0

	// 4. Create the Saboteur NPC
	npcEnt := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.Position](&world),
	)
	npcIdent := (*components.Identity)(world.Get(npcEnt, ecs.ComponentID[components.Identity](&world)))
	npcIdent.ID = npcID

	npcPos := (*components.Position)(world.Get(npcEnt, ecs.ComponentID[components.Position](&world)))
	npcPos.X = 10.5 // Within distance squared < 2.0 of village (dx=0.5, dy=0.5 -> distSq=0.5)
	npcPos.Y = 10.5

	// 5. Assign the deep negative hook (-150)
	hooks.AddHook(npcID, rulerID, -150)

	// 6. Run System (Tick 50 triggers evaluation)
	for i := 0; i < 50; i++ {
		sys.Update(&world)
	}

	// 7. Butterfly Effect Assertions

	// A. Food halving
	vStorAfter := (*components.StorageComponent)(world.Get(villageEnt, ecs.ComponentID[components.StorageComponent](&world)))
	if vStorAfter.Food != 50 {
		t.Errorf("Expected village food to be halved to 50, but got %d", vStorAfter.Food)
	}

	// B. Disease spawned
	diseaseQuery := world.Query(ecs.All(ecs.ComponentID[components.DiseaseEntity](&world), ecs.ComponentID[components.Position](&world)))
	foundDisease := false
	for diseaseQuery.Next() {
		dPos := (*components.Position)(diseaseQuery.Get(ecs.ComponentID[components.Position](&world)))
		if dPos.X == 10.0 && dPos.Y == 10.0 {
			foundDisease = true
			dComp := (*components.DiseaseEntity)(diseaseQuery.Get(ecs.ComponentID[components.DiseaseEntity](&world)))
			if dComp.Lethality < 80 {
				t.Errorf("Expected disease lethality to be >= 80, but got %d", dComp.Lethality)
			}
		}
	}
	if !foundDisease {
		t.Errorf("Expected a DiseaseEntity to be spawned at the village location.")
	}

	// C. CrimeMarker attached
	if !world.Has(npcEnt, ecs.ComponentID[components.CrimeMarker](&world)) {
		t.Errorf("Expected saboteur to receive a CrimeMarker.")
	} else {
		crime := (*components.CrimeMarker)(world.Get(npcEnt, ecs.ComponentID[components.CrimeMarker](&world)))
		if crime.Bounty != 500 {
			t.Errorf("Expected CrimeMarker bounty to be 500, got %d", crime.Bounty)
		}
	}

	// D. Hook Neutralized
	if hooks.GetHook(npcID, rulerID) != 0 {
		t.Errorf("Expected negative hook to be neutralized (0), got %d", hooks.GetHook(npcID, rulerID))
	}
}
