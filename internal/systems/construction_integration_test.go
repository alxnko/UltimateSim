package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 59 - The Physical Construction Engine
func TestConstructionSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(100, 100)
	hooks := engine.NewSparseHookGraph()

	// Initialize systems
	constructionSystem := NewConstructionSystem(&world, pathQueue)
	laborCrisisSystem := NewLaborCrisisSystem(&world, hooks)

	// Components
	vID := ecs.ComponentID[components.Village](&world)
	tID := ecs.ComponentID[components.TreasuryComponent](&world)
	sID := ecs.ComponentID[components.StorageComponent](&world)
	dID := ecs.ComponentID[components.DemographicsComponent](&world)
	pID := ecs.ComponentID[components.Position](&world)
	aID := ecs.ComponentID[components.Affiliation](&world)
	popID := ecs.ComponentID[components.PopulationComponent](&world)

	// Create a wealthy village
	vEnt := world.NewEntity(vID, tID, sID, dID, pID, aID, popID)

	treas := (*components.TreasuryComponent)(world.Get(vEnt, tID))
	treas.Wealth = 2000.0 // Enough for 4 sites

	stor := (*components.StorageComponent)(world.Get(vEnt, sID))
	stor.Wood = 500
	stor.Stone = 500

	demo := (*components.DemographicsComponent)(world.Get(vEnt, dID))
	demo.PeakPopulation = 100 // Starting peak

	pop := (*components.PopulationComponent)(world.Get(vEnt, popID))
	pop.Count = 100 // At peak initially

	aff := (*components.Affiliation)(world.Get(vEnt, aID))
	aff.CityID = 1

	pos := (*components.Position)(world.Get(vEnt, pID))
	pos.X = 10.0
	pos.Y = 10.0

	// Create Builders
	npcID := ecs.ComponentID[components.NPC](&world)
	jID := ecs.ComponentID[components.JobComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)

	for i := 0; i < 4; i++ {
		bEnt := world.NewEntity(npcID, jID, pID, aID, identID)

		bJob := (*components.JobComponent)(world.Get(bEnt, jID))
		bJob.JobID = components.JobBuilder

		bAff := (*components.Affiliation)(world.Get(bEnt, aID))
		bAff.CityID = 1

		bPos := (*components.Position)(world.Get(bEnt, pID))
		bPos.X = 10.0
		bPos.Y = 10.0

		bIdent := (*components.Identity)(world.Get(bEnt, identID))
		bIdent.ID = uint64(i + 1)
	}

	// 1. Tick construction system to spawn sites
	// Needs multiple ticks to trigger s.tickCounter % 100 == 0
	for i := 0; i < 100; i++ {
		constructionSystem.Update(&world)
	}

	// Verify sites spawned
	siteCompID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	siteFilter := ecs.All(siteCompID)
	siteQuery := world.Query(siteFilter)
	siteCount := 0
	for siteQuery.Next() {
		siteCount++
	}
	if siteCount == 0 {
		t.Fatalf("Expected construction sites to be spawned, got 0")
	}

	// 2. Tick construction system heavily to let builders finish structures
	// Wood (50) + Stone (50) + Progress (100) = ~200 ticks per site minimum
	for i := 0; i < 300; i++ {
		constructionSystem.Update(&world)
	}

	// 3. Verify structures were completed and demographics changed
	structCompID := ecs.ComponentID[components.StructureComponent](&world)
	structFilter := ecs.All(structCompID)
	structQuery := world.Query(structFilter)
	structCount := 0
	for structQuery.Next() {
		structCount++
	}

	if structCount == 0 {
		t.Fatalf("Expected structures to be completed, got 0")
	}

	if demo.PeakPopulation <= 100 {
		t.Fatalf("Expected PeakPopulation to increase, stayed at %d", demo.PeakPopulation)
	}

	// 4. Test "Butterfly Effect": Labor Crisis System should trigger
	// Current population is 100. New peak is 100 + (4 * 50) = 300.
	// 100 is < 80% of 300 (which is 240).
	// So LaborCrisisSystem should flag an active labor crisis!

	// Create a dummy market to allow LaborCrisisSystem to function
	mID := ecs.ComponentID[components.MarketComponent](&world)
	world.Add(vEnt, mID)

	// Re-get pointers as Add invalidates them
	demo = (*components.DemographicsComponent)(world.Get(vEnt, dID))
	pop = (*components.PopulationComponent)(world.Get(vEnt, popID))



	// We might need an identity to avoid crash when laborCrisisSystem iterates NPCs
	ident := (*components.Identity)(world.Get(vEnt, ecs.ComponentID[components.Identity](&world)))
	if ident == nil {
		world.Add(vEnt, ecs.ComponentID[components.Identity](&world))
		demo = (*components.DemographicsComponent)(world.Get(vEnt, dID))
		pop = (*components.PopulationComponent)(world.Get(vEnt, popID))

	}
	for i := 0; i < 110; i++ { laborCrisisSystem.Update(&world) }
 // Run it once


	if !demo.LaborCrisisActive {
		t.Logf("Pop count: %d, Peak: %d", pop.Count, demo.PeakPopulation)

		t.Errorf("Expected LaborCrisisActive to be true due to housing bubble artificially raising peak population")
	}
}
