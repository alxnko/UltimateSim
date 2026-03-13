package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.3: Maritime Attrition & Piracy
// TestStormSystem_Deterministic ensures that the StormSystem applies hull damage identically across seeded runs.

func TestStormSystem_Deterministic(t *testing.T) {
	setupWorld := func() (*ecs.World, *engine.MapGrid, ecs.Entity, ecs.Entity) {
		world := ecs.NewWorld()

		grid := engine.NewMapGrid(10, 10)
		seed := [32]byte{1, 2, 3, 4, 5}
		engine.InitializeRNG(seed)

		// Force tile to be ocean
		grid.Tiles[5*10+5].BiomeID = engine.BiomeOcean
		grid.Tiles[2*10+2].BiomeID = engine.BiomeGrassland

		shipID := ecs.ComponentID[components.ShipComponent](&world)
		posID := ecs.ComponentID[components.Position](&world)

		// Create ship on Ocean
		oceanShip := world.NewEntity(shipID, posID)
		(*components.Position)(world.Get(oceanShip, posID)).X = 5
		(*components.Position)(world.Get(oceanShip, posID)).Y = 5
		(*components.ShipComponent)(world.Get(oceanShip, shipID)).Hull = 100

		// Create ship on Land (should not take damage)
		landShip := world.NewEntity(shipID, posID)
		(*components.Position)(world.Get(landShip, posID)).X = 2
		(*components.Position)(world.Get(landShip, posID)).Y = 2
		(*components.ShipComponent)(world.Get(landShip, shipID)).Hull = 100

		return &world, grid, oceanShip, landShip
	}

	runSystem := func(world *ecs.World, grid *engine.MapGrid) {
		sys := NewStormSystem(grid)
		sys.stormChance = 0.5 // High chance for testing
		// Run for enough ticks to guarantee some storms
		for i := 0; i < 50; i++ {
			sys.Update(world)
		}
	}

	world1, grid1, oceanShip1, landShip1 := setupWorld()
	// Set the global seed explicitly before running system logic
	engine.InitializeRNG([32]byte{9, 9, 9})
	runSystem(world1, grid1)

	world2, grid2, oceanShip2, landShip2 := setupWorld()
	engine.InitializeRNG([32]byte{9, 9, 9})
	runSystem(world2, grid2)

	// Check land ships took no damage
	shipID1 := ecs.ComponentID[components.ShipComponent](world1)
	shipID2 := ecs.ComponentID[components.ShipComponent](world2)

	landHull1 := (*components.ShipComponent)(world1.Get(landShip1, shipID1)).Hull
	landHull2 := (*components.ShipComponent)(world2.Get(landShip2, shipID2)).Hull

	if landHull1 != 100 {
		t.Errorf("Land ship 1 took damage, expected 100, got %d", landHull1)
	}
	if landHull2 != 100 {
		t.Errorf("Land ship 2 took damage, expected 100, got %d", landHull2)
	}

	// Check ocean ships took identical damage or were both destroyed
	alive1 := world1.Alive(oceanShip1)
	alive2 := world2.Alive(oceanShip2)

	if alive1 != alive2 {
		t.Fatalf("Ocean ship alive mismatch: run1=%v, run2=%v", alive1, alive2)
	}

	if alive1 {
		oceanHull1 := (*components.ShipComponent)(world1.Get(oceanShip1, shipID1)).Hull
		oceanHull2 := (*components.ShipComponent)(world2.Get(oceanShip2, shipID2)).Hull
		if oceanHull1 != oceanHull2 {
			t.Errorf("Ocean hull mismatch: run1=%d, run2=%d", oceanHull1, oceanHull2)
		}
		if oceanHull1 == 100 {
			t.Logf("Note: Ocean ship took 0 damage over 50 ticks. Consider increasing storm chance for testing or increasing ticks.")
		}
	}
}
