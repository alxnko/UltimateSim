package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestDeforestationSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Set initial wood value
	mapGrid.Resources[5*10+5].WoodValue = 10

	sys := NewDeforestationSystem(mapGrid)

	// Create employer/village
	village := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.StorageComponent](&world),
	)

	vilIdent := (*components.Identity)(world.Get(village, ecs.ComponentID[components.Identity](&world)))
	vilIdent.ID = 100

	vilStorage := (*components.StorageComponent)(world.Get(village, ecs.ComponentID[components.StorageComponent](&world)))
	vilStorage.Wood = 0

	// Create lumberjack
	npc := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
	)

	pos := (*components.Position)(world.Get(npc, ecs.ComponentID[components.Position](&world)))
	pos.X = 5
	pos.Y = 5

	job := (*components.JobComponent)(world.Get(npc, ecs.ComponentID[components.JobComponent](&world)))
	job.JobID = components.JobLumberjack
	job.EmployerID = vilIdent.ID

	vitals := (*components.VitalsComponent)(world.Get(npc, ecs.ComponentID[components.VitalsComponent](&world)))
	vitals.Consciousness = 100
	vitals.Stamina = 100

	sys.Update(&world)

	if mapGrid.Resources[5*10+5].WoodValue != 9 {
		t.Errorf("Expected wood value to decrease to 9, got %d", mapGrid.Resources[5*10+5].WoodValue)
	}

	if vilStorage.Wood != 1 {
		t.Errorf("Expected village storage wood to increase to 1, got %d", vilStorage.Wood)
	}

	if vitals.Stamina != 95 {
		t.Errorf("Expected stamina to decrease to 95, got %f", vitals.Stamina)
	}
}

// Evolution: Phase 55 - The Ecological Collapse Engine (DeforestationSystem)
// Butterfly Effect Test
func TestDeforestationSystem_ButterflyEffect(t *testing.T) {
	// A lumberjack harvests wood. The wood goes into the city's storage.
	// This reduces the local tile's wood value to 0.
	// This proves interaction between DeforestationSystem, WinterHeatingSystem, and DesperationSystem.

	engine.InitializeRNG([32]byte{1, 2, 3}) // Initialize RNG

	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Set initial wood value
	mapGrid.Resources[5*10+5].WoodValue = 1

	sys := NewDeforestationSystem(mapGrid)
	cal := &engine.Calendar{Ticks: 0}
	cal.IsWinter = true
	whSys := NewWinterHeatingSystem(&world, cal)
	desSys := NewDesperationSystem(&world)

	// Create employer
	village := world.NewEntity(
		ecs.ComponentID[components.Village](&world),
		ecs.ComponentID[components.Identity](&world),
		ecs.ComponentID[components.StorageComponent](&world),
		ecs.ComponentID[components.PopulationComponent](&world),
		ecs.ComponentID[components.LoyaltyComponent](&world),
		ecs.ComponentID[components.Position](&world),
	)

	vilIdent := (*components.Identity)(world.Get(village, ecs.ComponentID[components.Identity](&world)))
	vilIdent.ID = 100

	vilStorage := (*components.StorageComponent)(world.Get(village, ecs.ComponentID[components.StorageComponent](&world)))
	vilStorage.Wood = 0
	vilStorage.Food = 100

	pop := (*components.PopulationComponent)(world.Get(village, ecs.ComponentID[components.PopulationComponent](&world)))
	pop.Count = 20 // Need 2 wood per tick. We only have 1 on the map.

	loyalty := (*components.LoyaltyComponent)(world.Get(village, ecs.ComponentID[components.LoyaltyComponent](&world)))
	loyalty.Value = 100

	vPos := (*components.Position)(world.Get(village, ecs.ComponentID[components.Position](&world)))
	vPos.X = 5
	vPos.Y = 5

	// Create lumberjack
	npc := world.NewEntity(
		ecs.ComponentID[components.NPC](&world),
		ecs.ComponentID[components.Position](&world),
		ecs.ComponentID[components.JobComponent](&world),
		ecs.ComponentID[components.VitalsComponent](&world),
		ecs.ComponentID[components.Affiliation](&world),
		ecs.ComponentID[components.Needs](&world),
		ecs.ComponentID[components.DesperationComponent](&world),
		ecs.ComponentID[components.Memory](&world),
		ecs.ComponentID[components.Identity](&world),
	)

	pos := (*components.Position)(world.Get(npc, ecs.ComponentID[components.Position](&world)))
	pos.X = 5
	pos.Y = 5

	job := (*components.JobComponent)(world.Get(npc, ecs.ComponentID[components.JobComponent](&world)))
	job.JobID = components.JobLumberjack
	job.EmployerID = vilIdent.ID

	vitals := (*components.VitalsComponent)(world.Get(npc, ecs.ComponentID[components.VitalsComponent](&world)))
	vitals.Consciousness = 100
	vitals.Stamina = 100
	vitals.Pain = 0

	needs := (*components.Needs)(world.Get(npc, ecs.ComponentID[components.Needs](&world)))
	needs.Food = 0 // Food 0 will trigger desperation
	needs.Safety = 0
	needs.Wealth = 0

	desp := (*components.DesperationComponent)(world.Get(npc, ecs.ComponentID[components.DesperationComponent](&world)))
	desp.Level = 0

	sys.Update(&world) // Harvest 1 wood, depletes tile, VilStorage gets 1 wood
	whSys.Update(&world) // Needs 2 wood, has 1. Wood drops to 0, village freezing crisis
	desSys.Update(&world) // Needs.Food/Wealth is low, Desperation increases

	if mapGrid.Resources[5*10+5].WoodValue != 0 {
		t.Errorf("Expected wood value to decrease to 0, got %d", mapGrid.Resources[5*10+5].WoodValue)
	}

	if vilStorage.Wood != 0 {
		t.Errorf("Expected village storage wood to drain to 0, got %d", vilStorage.Wood)
	}

	if loyalty.Value != 95 {
		t.Errorf("Expected loyalty to drop to 95 due to freezing crisis, got %d", loyalty.Value)
	}

	// Fast forward desperation
	for i := 0; i < 20; i++ {
		whSys.Update(&world)
		desSys.Update(&world)
	}

	if desp.Level <= 0 {
		t.Errorf("Expected desperation to increase due to low needs, got %d", desp.Level)
	}
}
