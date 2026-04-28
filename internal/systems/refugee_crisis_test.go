package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestRefugeeSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	pathQueue := engine.NewPathRequestQueue(10, 2)
	sys := NewRefugeeSystem(&world, pathQueue)

	// Components
	vID := ecs.ComponentID[components.Village](&world)
	tID := ecs.ComponentID[components.TreasuryComponent](&world)
	aID := ecs.ComponentID[components.Affiliation](&world)
	pID := ecs.ComponentID[components.Position](&world)
	dID := ecs.ComponentID[components.DesperationComponent](&world)
	identID := ecs.ComponentID[components.Identity](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	asylumID := ecs.ComponentID[components.AsylumSeekerComponent](&world)

	// Create prosperous village
	village := world.NewEntity(vID, tID, aID, pID)
	vPos := (*components.Position)(world.Get(village, pID))
	vPos.X = 10.0
	vPos.Y = 10.0

	vAff := (*components.Affiliation)(world.Get(village, aID))
	vAff.CityID = 2

	vTreas := (*components.TreasuryComponent)(world.Get(village, tID))
	vTreas.Wealth = 150.0

	// Create desperate NPC
	npc := world.NewEntity(dID, identID, pID, aID, pathID)
	nPos := (*components.Position)(world.Get(npc, pID))
	nPos.X = 0.0
	nPos.Y = 0.0

	nAff := (*components.Affiliation)(world.Get(npc, aID))
	nAff.CityID = 1

	nIdent := (*components.Identity)(world.Get(npc, identID))
	nIdent.ID = 100

	nDesp := (*components.DesperationComponent)(world.Get(npc, dID))
	nDesp.Level = 85

	// 1. Tick: NPC should acquire AsylumSeekerComponent targeting City 2
	sys.tickCounter = 9 // next update is 10
	sys.Update(&world)

	if !world.Has(npc, asylumID) {
		t.Fatalf("Expected NPC to acquire AsylumSeekerComponent")
	}

	asylum := (*components.AsylumSeekerComponent)(world.Get(npc, asylumID))
	if asylum.TargetCityID != 2 {
		t.Errorf("Expected AsylumSeekerComponent.TargetCityID to be 2, got %d", asylum.TargetCityID)
	}

	nPath := (*components.Path)(world.Get(npc, pathID))
	if !nPath.HasPath {
		t.Errorf("Expected NPC to have a path request")
	}
	if nPath.TargetX != 10.0 || nPath.TargetY != 10.0 {
		t.Errorf("Expected NPC path target to be (10.0, 10.0)")
	}

	// 2. Move NPC close to the target city
	nPos = (*components.Position)(world.Get(npc, pID))
	nPos.X = 9.0
	nPos.Y = 9.0

	// 3. Tick: NPC should assimilate
	sys.tickCounter = 19 // next update is 20
	sys.Update(&world)

	if world.Has(npc, asylumID) {
		t.Errorf("Expected NPC to lose AsylumSeekerComponent upon assimilation")
	}

	nAff = (*components.Affiliation)(world.Get(npc, aID))
	if nAff.CityID != 2 {
		t.Errorf("Expected NPC Affiliation.CityID to be 2, got %d", nAff.CityID)
	}

	nDesp = (*components.DesperationComponent)(world.Get(npc, dID))
	if nDesp.Level != 0 {
		t.Errorf("Expected NPC Desperation.Level to reset to 0, got %d", nDesp.Level)
	}
}
