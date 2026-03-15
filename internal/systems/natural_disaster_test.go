package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestNaturalDisasterSystem_Integration(t *testing.T) {
	world := ecs.NewWorld()
	grid := engine.NewMapGrid(20, 20)

	sys := NewNaturalDisasterSystem(&world, grid)
	// Force spawn for test
	sys.spawnChance = 1.0
	// Predict PRNG output for seed {1, 0, 0, ...}
	// To be truly deterministic, we should know the PRNG output or just verify the effect
	// The first random position with seed {1, 0...} needs to be hit.
	// Since we don't mock rand, we will let it hit a spot, find where it hit by looking for the wiped grid tiles,
	// but wait! The disaster gets despawned instantly.
	// We can instead surround the map with Villages and NPCs everywhere, then check that at least one got hit.

	posID := ecs.ComponentID[components.Position](&world)
	npcID := ecs.ComponentID[components.NPC](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	storageID := ecs.ComponentID[components.StorageComponent](&world)

	// Populate the entire grid with resources and entities
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			idx := y*20 + x
			grid.Resources[idx].WoodValue = 100
			grid.TileStates[idx].FootTraffic = 100

			// NPC
			npc := world.NewEntity()
			world.Add(npc, posID, npcID, vitalsID)
			pos := (*components.Position)(world.Get(npc, posID))
			pos.X = float32(x)
			pos.Y = float32(y)
			vitals := (*components.VitalsComponent)(world.Get(npc, vitalsID))
			vitals.Pain = 0

			// Village
			village := world.NewEntity()
			world.Add(village, posID, villageID, storageID)
			vPos := (*components.Position)(world.Get(village, posID))
			vPos.X = float32(x)
			vPos.Y = float32(y)
			storage := (*components.StorageComponent)(world.Get(village, storageID))
			storage.Food = 1000
		}
	}

	// Tick once to trigger disaster
	sys.Update(&world)

	// Verify MapGrid damage
	mapDamaged := false
	for i := 0; i < len(grid.Resources); i++ {
		if grid.Resources[i].WoodValue == 0 && grid.TileStates[i].FootTraffic == 0 {
			mapDamaged = true
			break
		}
	}

	if !mapDamaged {
		t.Errorf("Expected map grid to take damage (wood/traffic wiped)")
	}

	// Verify NPC damage
	npcQuery := world.Query(ecs.All(npcID, vitalsID))
	npcDamaged := false
	for npcQuery.Next() {
		vitals := (*components.VitalsComponent)(npcQuery.Get(vitalsID))
		if vitals.Pain > 0 {
			npcDamaged = true
			break
		}
	}

	if !npcDamaged {
		t.Errorf("Expected at least one NPC to take pain damage")
	}

	// Verify Village storage destruction
	villageQuery := world.Query(ecs.All(villageID, storageID))
	villageDamaged := false
	for villageQuery.Next() {
		storage := (*components.StorageComponent)(villageQuery.Get(storageID))
		if storage.Food == 0 {
			villageDamaged = true
			break
		}
	}

	if !villageDamaged {
		t.Errorf("Expected at least one Village to lose its storage")
	}

	// Determinism Test
	world2 := ecs.NewWorld()
	grid2 := engine.NewMapGrid(20, 20)
	sys2 := NewNaturalDisasterSystem(&world2, grid2)
	sys2.spawnChance = 1.0

	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			idx := y*20 + x
			grid2.Resources[idx].WoodValue = 100
			grid2.TileStates[idx].FootTraffic = 100
		}
	}

	sys2.Update(&world2)

	// Check if grid and grid2 have identical wiped tiles
	for i := 0; i < len(grid.Resources); i++ {
		if grid.Resources[i].WoodValue != grid2.Resources[i].WoodValue {
			t.Errorf("Determinism failed: Grid 1 and Grid 2 have different resource values at %d", i)
		}
		if grid.TileStates[i].FootTraffic != grid2.TileStates[i].FootTraffic {
			t.Errorf("Determinism failed: Grid 1 and Grid 2 have different foot traffic values at %d", i)
		}
	}
}
