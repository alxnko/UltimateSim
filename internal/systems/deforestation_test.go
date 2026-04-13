package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 55: The Ecological Collapse Engine (DeforestationSystem)
// E2E Test proving the Butterfly Effect between Geography and Economy

func TestDeforestationSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)

	// Setup a MapGrid tile with Forest and Wood
	tileIdx := 5*10 + 5
	mapGrid.Tiles[tileIdx].BiomeID = engine.BiomeTemperateDeciduousForest
	mapGrid.Resources[tileIdx].WoodValue = 5

	sys := NewDeforestationSystem(&world, mapGrid)

	// Create a Village with a storage component
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	posID := ecs.ComponentID[components.Position](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	jobID := ecs.ComponentID[components.JobComponent](&world)

	villageEntity := world.NewEntity()
	world.Add(villageEntity, villageID, storageID, idID)

	vIdent := (*components.Identity)(world.Get(villageEntity, idID))
	vIdent.ID = 12345

	vStorage := (*components.StorageComponent)(world.Get(villageEntity, storageID))
	vStorage.Wood = 0

	// Create an NPC Lumberjack employed by the Village
	npcEntity := world.NewEntity()
	world.Add(npcEntity, npcID, posID, jobID)

	npcPos := (*components.Position)(world.Get(npcEntity, posID))
	npcPos.X = 5.0
	npcPos.Y = 5.0

	npcJob := (*components.JobComponent)(world.Get(npcEntity, jobID))
	npcJob.JobID = components.JobLumberjack
	npcJob.EmployerID = 12345

	// Run system. It processes every 60 ticks.
	for i := 0; i < 60; i++ {
		sys.Update()
	}

	// Verify Wood extraction (1 unit per 60 ticks)
	if mapGrid.Resources[tileIdx].WoodValue != 4 {
		t.Errorf("Expected MapGrid WoodValue to be 4, got %d", mapGrid.Resources[tileIdx].WoodValue)
	}

	if vStorage.Wood != 1 {
		t.Errorf("Expected Village Storage Wood to be 1, got %d", vStorage.Wood)
	}

	// Run until deforestation (4 more cycles of 60 ticks)
	for i := 0; i < 240; i++ {
		sys.Update()
	}

	if mapGrid.Resources[tileIdx].WoodValue != 0 {
		t.Errorf("Expected MapGrid WoodValue to be 0, got %d", mapGrid.Resources[tileIdx].WoodValue)
	}

	if vStorage.Wood != 5 {
		t.Errorf("Expected Village Storage Wood to be 5, got %d", vStorage.Wood)
	}

	if mapGrid.Tiles[tileIdx].BiomeID != engine.BiomeGrassland {
		t.Errorf("Expected MapGrid BiomeID to degrade to Grassland, got %d", mapGrid.Tiles[tileIdx].BiomeID)
	}
}

// Determinism test
func TestDeforestationSystem_Deterministic(t *testing.T) {
	world1 := ecs.NewWorld()
	grid1 := engine.NewMapGrid(10, 10)
	sys1 := NewDeforestationSystem(&world1, grid1)

	world2 := ecs.NewWorld()
	grid2 := engine.NewMapGrid(10, 10)
	sys2 := NewDeforestationSystem(&world2, grid2)

	// Setup same conditions
	setup := func(w *ecs.World, g *engine.MapGrid) {
		idx := 2*10 + 2
		g.Tiles[idx].BiomeID = engine.BiomeTemperateDeciduousForest
		g.Resources[idx].WoodValue = 10

		vID := ecs.ComponentID[components.Village](w)
		sID := ecs.ComponentID[components.StorageComponent](w)
		iID := ecs.ComponentID[components.Identity](w)
		pID := ecs.ComponentID[components.Position](w)
		nID := ecs.ComponentID[components.NPC](w)
		jID := ecs.ComponentID[components.JobComponent](w)

		ve := w.NewEntity()
		w.Add(ve, vID, sID, iID)
		(*components.Identity)(w.Get(ve, iID)).ID = 1
		(*components.StorageComponent)(w.Get(ve, sID)).Wood = 0

		ne := w.NewEntity()
		w.Add(ne, nID, pID, jID)
		pos := (*components.Position)(w.Get(ne, pID))
		pos.X, pos.Y = 2.0, 2.0
		job := (*components.JobComponent)(w.Get(ne, jID))
		job.JobID = components.JobLumberjack
		job.EmployerID = 1
	}

	setup(&world1, grid1)
	setup(&world2, grid2)

	for i := 0; i < 300; i++ {
		sys1.Update()
		sys2.Update()
	}

	if grid1.Resources[22].WoodValue != grid2.Resources[22].WoodValue {
		t.Errorf("Determinism failed: WoodValue mismatch")
	}

	// Query to get the entities back since we cannot construct them directly
	var s1, s2 *components.StorageComponent

	q1 := world1.Query(filter.All(ecs.ComponentID[components.Village](&world1)))
	for q1.Next() {
		s1 = (*components.StorageComponent)(q1.Get(ecs.ComponentID[components.StorageComponent](&world1)))
	}

	q2 := world2.Query(filter.All(ecs.ComponentID[components.Village](&world2)))
	for q2.Next() {
		s2 = (*components.StorageComponent)(q2.Get(ecs.ComponentID[components.StorageComponent](&world2)))
	}

	if s1.Wood != s2.Wood {
		t.Errorf("Determinism failed: Storage mismatch %d vs %d", s1.Wood, s2.Wood)
	}
}
