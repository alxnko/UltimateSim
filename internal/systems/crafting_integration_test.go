package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 60 - The Physical Crafting Engine
func TestCraftingSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(100, 100)

	// Initialize systems
	craftingSystem := NewCraftingSystem(&world, pathQueue)
	constructionSystem := NewConstructionSystem(&world, pathQueue)

	// Create Employer (Village)
	vID := ecs.ComponentID[components.Village](&world)
	tID := ecs.ComponentID[components.TreasuryComponent](&world)
	sID := ecs.ComponentID[components.StorageComponent](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	pID := ecs.ComponentID[components.Position](&world)
	aID := ecs.ComponentID[components.Affiliation](&world)
	dID := ecs.ComponentID[components.DemographicsComponent](&world)

	villageEnt := world.NewEntity(vID, tID, sID, idID, pID, aID, dID)

	vIdent := (*components.Identity)(world.Get(villageEnt, idID))
	vIdent.ID = 1001

	vTreas := (*components.TreasuryComponent)(world.Get(villageEnt, tID))
	vTreas.Wealth = 0.0 // Starts broke

	vStor := (*components.StorageComponent)(world.Get(villageEnt, sID))
	vStor.Iron = 20 // Has iron to forge

	vAff := (*components.Affiliation)(world.Get(villageEnt, aID))
	vAff.CityID = 1001

	vPos := (*components.Position)(world.Get(villageEnt, pID))
	vPos.X = 10.0
	vPos.Y = 10.0

	// Create Workbench
	wbID := ecs.ComponentID[components.WorkbenchComponent](&world)
	wbEnt := world.NewEntity(wbID)

	wb := (*components.WorkbenchComponent)(world.Get(wbEnt, wbID))
	wb.EmployerID = 1001
	wb.X = 15.0
	wb.Y = 15.0

	// Create Artisan
	npcID := ecs.ComponentID[components.NPC](&world)
	jID := ecs.ComponentID[components.JobComponent](&world)

	artisanEnt := world.NewEntity(npcID, jID, pID, idID)

	aPos := (*components.Position)(world.Get(artisanEnt, pID))
	aPos.X = 10.0
	aPos.Y = 10.0

	aJob := (*components.JobComponent)(world.Get(artisanEnt, jID))
	aJob.JobID = components.JobArtisan
	aJob.EmployerID = 1001

	aIdent := (*components.Identity)(world.Get(artisanEnt, idID))
	aIdent.ID = 2002

	// Run Crafting System
	// Artisan starts at 10,10 and workbench is at 15,15
	// It will take some ticks to walk there and craft
	for i := 0; i < 50; i++ {
		craftingSystem.Update(&world)
	}

	// Iron should be consumed, wealth should be generated
	vStor = (*components.StorageComponent)(world.Get(villageEnt, sID))
	vTreas = (*components.TreasuryComponent)(world.Get(villageEnt, tID))

	if vStor.Iron >= 20 {
		t.Fatalf("Expected Iron to be consumed, got %d", vStor.Iron)
	}

	if vTreas.Wealth <= 0 {
		t.Fatalf("Expected Wealth to be generated, got %f", vTreas.Wealth)
	}

	// We'll give them enough iron to cross the 500 wealth threshold
	vStor.Iron = 100
	for i := 0; i < 200; i++ {
		craftingSystem.Update(&world)
	}

	vTreas = (*components.TreasuryComponent)(world.Get(villageEnt, tID))
	if vTreas.Wealth <= 500 {
		t.Fatalf("Expected Wealth to cross 500 threshold, got %f", vTreas.Wealth)
	}

	// Now run the Construction System. It should detect the wealth > 500 and spawn a construction site.
	// Tick 100 times to ensure s.tickCounter % 100 == 0 triggers in ConstructionSystem
	for i := 0; i < 100; i++ {
		constructionSystem.Update(&world)
	}

	// Verify Construction Site was spawned
	siteCompID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	siteFilter := ecs.All(siteCompID)
	siteQuery := world.Query(siteFilter)

	siteSpawned := false
	for siteQuery.Next() {
		siteSpawned = true
		break
	}

	if !siteSpawned {
		t.Fatalf("Butterfly Effect Failed: Crafting generated wealth, but ConstructionSystem did not spawn a site.")
	}
}
