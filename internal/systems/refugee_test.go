package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// TestRefugeeSystem_Integration verifies the Phase 63 E2E "Butterfly Effect".
// Starving NPCs spawn AsylumSeekers, path to wealthy city, and assimilate.
func TestRefugeeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()

	refugeeSys := NewRefugeeSystem(&world)

	posID := ecs.ComponentID[components.Position](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	despID := ecs.ComponentID[components.DesperationComponent](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	vilID := ecs.ComponentID[components.Village](&world)
	treasID := ecs.ComponentID[components.TreasuryComponent](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	// Create starving village NPC
	npc := world.NewEntity(posID, needsID, despID, affID, pathID)
	npcPos := (*components.Position)(world.Get(npc, posID))
	npcPos.X = 0.0
	npcPos.Y = 0.0

	npcNeeds := (*components.Needs)(world.Get(npc, needsID))
	npcNeeds.Wealth = 5.0

	npcDesp := (*components.DesperationComponent)(world.Get(npc, despID))
	npcDesp.Level = 85

	npcAff := (*components.Affiliation)(world.Get(npc, affID))
	npcAff.CityID = 1 // Starving city

	// Create wealthy city
	city := world.NewEntity(posID, vilID, affID, treasID)
	cityPos := (*components.Position)(world.Get(city, posID))
	cityPos.X = 10.0
	cityPos.Y = 10.0

	cityAff := (*components.Affiliation)(world.Get(city, affID))
	cityAff.CityID = 2

	cityTreas := (*components.TreasuryComponent)(world.Get(city, treasID))
	cityTreas.Wealth = 500.0

	// Tick 1: NPC hits extreme desperation, spawns AsylumSeekerComponent
	refugeeSys.Update(&world)

	asylumID := ecs.ComponentID[components.AsylumSeekerComponent](&world)
	if !world.Has(npc, asylumID) {
		t.Fatalf("Expected NPC to receive AsylumSeekerComponent")
	}

	npcAff = (*components.Affiliation)(world.Get(npc, affID))
	if npcAff.CityID != 0 {
		t.Errorf("Expected NPC CityID to be cleared (abandoned state), got %d", npcAff.CityID)
	}

	asc := (*components.AsylumSeekerComponent)(world.Get(npc, asylumID))
	if asc.TargetCityID != 2 {
		t.Errorf("Expected AsylumSeeker to target CityID 2, got %d", asc.TargetCityID)
	}

	// Re-fetch pointer because Add invalidated it!
	npcPos = (*components.Position)(world.Get(npc, posID))

	// Move NPC to target city (distSq < 4.0)
	npcPos.X = 9.0
	npcPos.Y = 9.0

	// Tick 2: Assimilation
	refugeeSys.Update(&world)

	if world.Has(npc, asylumID) {
		t.Errorf("Expected AsylumSeekerComponent to be removed upon arrival")
	}

	npcAff = (*components.Affiliation)(world.Get(npc, affID))
	if npcAff.CityID != 2 {
		t.Errorf("Expected NPC to assimilate to CityID 2, got %d", npcAff.CityID)
	}
}
