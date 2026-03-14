package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.2 & 14: The Genesis Spawner Tests

func TestNPCSpawner_E2E(t *testing.T) {
	// Initialize deterministic RNG
	engine.InitializeRNG([32]byte{1, 2, 3, 4})

	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(20, 20)

	// Set up the grid
	// By default, everything is empty (0), which is BiomeOcean.
	// We'll set a few tiles to Grassland to make them habitable.
	habitableCount := 0
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if x >= 5 && x < 15 && y >= 5 && y < 15 {
				mapGrid.SetTile(x, y, engine.TileData{BiomeID: engine.BiomeGrassland})
				habitableCount++
			}
		}
	}

	spawner := systems.NewNPCSpawnerSystem(&world, mapGrid)

	// Tick once to spawn
	spawner.Update(&world)

	// Verify exactly 100 entities are spawned (20 families * 5 individuals)
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	idID := ecs.ComponentID[components.Identity](&world)
	genID := ecs.ComponentID[components.GenomeComponent](&world)
	legID := ecs.ComponentID[components.Legacy](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)

	query := world.Query(ecs.All(posID, velID, idID, genID, legID, needsID, npcID, affID))
	count := 0

	for query.Next() {
		count++
		pos := (*components.Position)(query.Get(posID))
		id := (*components.Identity)(query.Get(idID))
		gen := (*components.GenomeComponent)(query.Get(genID))
		needs := (*components.Needs)(query.Get(needsID))
		aff := (*components.Affiliation)(query.Get(affID))

		// Verify placement is on a habitable tile
		tile := mapGrid.GetTile(int(pos.X), int(pos.Y))
		if tile.BiomeID == engine.BiomeOcean {
			t.Errorf("Entity spawned on an uninhabitable tile at (%f, %f)", pos.X, pos.Y)
		}

		// Basic bounds checks
		if id.ID == 0 {
			t.Errorf("Entity ID not set")
		}
		if needs.Food != 1000.0 {
			t.Errorf("Needs not initialized correctly")
		}
		if gen.Strength > 100 {
			t.Errorf("Genetics out of bounds")
		}
		if aff.FamilyID == 0 {
			t.Errorf("FamilyID not set")
		}
	}

	if count != 100 {
		t.Fatalf("Expected exactly 100 entities to be spawned, got %d", count)
	}

	// Tick again and verify count doesn't increase
	spawner.Update(&world)
	query2 := world.Query(ecs.All(posID))
	count2 := 0
	for query2.Next() {
		count2++
	}
	if count2 != 100 {
		t.Fatalf("Expected count to remain 100, got %d", count2)
	}
}

func TestNPCSpawner_Deterministic(t *testing.T) {
	runSim := func() float32 {
		engine.InitializeRNG([32]byte{42})
		world := ecs.NewWorld()
		mapGrid := engine.NewMapGrid(20, 20)

		for y := 0; y < 20; y++ {
			for x := 0; x < 20; x++ {
				mapGrid.SetTile(x, y, engine.TileData{BiomeID: engine.BiomeGrassland})
			}
		}

		spawner := systems.NewNPCSpawnerSystem(&world, mapGrid)
		spawner.Update(&world)

		posID := ecs.ComponentID[components.Position](&world)
		query := world.Query(ecs.All(posID))

		var sumX float32 = 0
		for query.Next() {
			pos := (*components.Position)(query.Get(posID))
			sumX += pos.X
		}
		return sumX
	}

	result1 := runSim()
	result2 := runSim()

	if result1 != result2 {
		t.Fatalf("Determinism check failed: Run 1 total X = %f, Run 2 total X = %f", result1, result2)
	}
}
