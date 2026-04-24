package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 63: The Refugee Crisis Engine (Butterfly Effect E2E Test)
// Validates that localized starvation -> Refugee Migration -> Xenophobia Grudge.

func TestRefugeeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(100, 4)
	hooks := engine.NewSparseHookGraph()

	refugeeSys := NewRefugeeSystem(&world, pathQueue)
	xenoSys := NewXenophobiaSystem(&world, hooks)

	// Component IDs
	posID := ecs.ComponentID[components.Position](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	storID := ecs.ComponentID[components.StorageComponent](&world)
	villID := ecs.ComponentID[components.Village](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	cultID := ecs.ComponentID[components.CultureComponent](&world)
	belID := ecs.ComponentID[components.BeliefComponent](&world)

	// 1. Create a starving NPC in City 1
	npcEnt := world.NewEntity(npcID, posID, affID, despID, identID, cultID, belID)

	ident := (*components.Identity)(world.Get(npcEnt, identID))
	ident.ID = 1

	pos := (*components.Position)(world.Get(npcEnt, posID))
	pos.X = 0
	pos.Y = 0

	aff := (*components.Affiliation)(world.Get(npcEnt, affID))
	aff.CityID = 100

	desp := (*components.DesperationComponent)(world.Get(npcEnt, despID))
	desp.Level = 85 // Critical desperation

	cult := (*components.CultureComponent)(world.Get(npcEnt, cultID))
	cult.LanguageID = 1 // Language 1

	// 2. Create a prosperous Village (City 2)
	villEnt := world.NewEntity(villID, posID, affID, storID)

	vPos := (*components.Position)(world.Get(villEnt, posID))
	vPos.X = 10
	vPos.Y = 10

	vAff := (*components.Affiliation)(world.Get(villEnt, affID))
	vAff.CityID = 200

	vStor := (*components.StorageComponent)(world.Get(villEnt, storID))
	vStor.Food = 500 // Very prosperous

	// 3. Create a Xenophobic citizen in City 2
	xenoEnt := world.NewEntity(npcID, posID, affID, identID, cultID, belID)

	xIdent := (*components.Identity)(world.Get(xenoEnt, identID))
	xIdent.ID = 2

	xPos := (*components.Position)(world.Get(xenoEnt, posID))
	xPos.X = 10
	xPos.Y = 10

	xAff := (*components.Affiliation)(world.Get(xenoEnt, affID))
	xAff.CityID = 200

	xCult := (*components.CultureComponent)(world.Get(xenoEnt, cultID))
	xCult.LanguageID = 2 // Different language!

	xBel := (*components.BeliefComponent)(world.Get(xenoEnt, belID))
	xBel.Beliefs = append(xBel.Beliefs, components.Belief{
		BeliefID: components.BeliefXenophobia,
		Weight:   100,
	})

	// === TEST EXECUTION ===

	// Step 1: RefugeeSystem ticks. NPC should become an AsylumSeeker.
	refugeeSys.tickCounter = 29
	refugeeSys.Update(&world)

	asylumID := ecs.ComponentID[components.AsylumSeekerComponent](&world)
	if !world.Has(npcEnt, asylumID) {
		t.Fatalf("NPC did not gain AsylumSeekerComponent")
	}

	aff = (*components.Affiliation)(world.Get(npcEnt, affID))
	if aff.CityID != 0 {
		t.Fatalf("NPC did not sever affiliation from original city")
	}

	asylum := (*components.AsylumSeekerComponent)(world.Get(npcEnt, asylumID))
	if asylum.TargetCityID != 200 || asylum.TargetX != 10 || asylum.TargetY != 10 {
		t.Fatalf("NPC did not target the prosperous city")
	}

	// Step 2: Teleport NPC to destination to simulate arrival
	pos = (*components.Position)(world.Get(npcEnt, posID))
	pos.X = 10
	pos.Y = 10

	// Step 3: RefugeeSystem ticks again. NPC should integrate.
	refugeeSys.tickCounter = 59
	refugeeSys.Update(&world)

	if world.Has(npcEnt, asylumID) {
		t.Fatalf("NPC did not lose AsylumSeekerComponent upon arrival")
	}

	aff = (*components.Affiliation)(world.Get(npcEnt, affID))
	if aff.CityID != 200 {
		t.Fatalf("NPC did not integrate into new city")
	}

	desp = (*components.DesperationComponent)(world.Get(npcEnt, despID))
	if desp.Level != 0 {
		t.Fatalf("NPC desperation was not reset after integration")
	}

	// Step 4: XenophobiaSystem ticks. Because the refugee (Language 1) is now physically
	// in the same location as the Xenophobe (Language 2), a massive grudge should spawn.
	xenoSys.tickCounter = 9
	xenoSys.Update(&world)

	grudge := hooks.GetHook(2, 1) // Xenophobe (2) hates Refugee (1)
	if grudge != -100 {
		t.Fatalf("Butterfly Effect Failed: Xenophobe did not generate -100 grudge against refugee. Got: %d", grudge)
	}
}
