package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.3: Infrastructure Wear System (Desire Paths) Testing

func TestDesirePathCreation(t *testing.T) {
	// 1. Initialize world and components
	world := ecs.NewWorld()

	// Create a 10x10 map grid
	mapGrid := engine.NewMapGrid(10, 10)

	// Set tile (1,0) to Mountain biome to give it a high movement cost
	tileIndex := 1 // x=1, y=0
	mapGrid.Tiles[tileIndex].BiomeID = engine.BiomeMountain

	initialCost := engine.GetEffectiveMovementCost(engine.BiomeMountain, 0)
	if initialCost <= 1.0 {
		t.Fatalf("Expected Mountain biome to have > 1.0 cost, got %f", initialCost)
	}

	// 2. Spawn a moving entity that travels across the mountain tile
	e := world.NewEntity()
	posID := ecs.ComponentID[components.Position](&world)
	velID := ecs.ComponentID[components.Velocity](&world)
	pathID := ecs.ComponentID[components.Path](&world)

	world.Add(e, posID, velID, pathID)

	// Start at x=0.5, y=0.5 (Tile 0,0)
	pos := (*components.Position)(world.Get(e, posID))
	pos.X = 0.5
	pos.Y = 0.5

	// Move towards x=2.5, y=0.5 (Tile 2,0)
	path := (*components.Path)(world.Get(e, pathID))
	path.HasPath = true
	path.Nodes = []components.Position{
		{X: 2.5, Y: 0.5},
	}

	// 3. Initialize MovementSystem
	movSystem := NewMovementSystem(&world, mapGrid)

	// 4. Tick simulation until entity crosses the boundary into tile (1,0)
	// At base speed 1.0 on standard tile, crossing from 0.5 to 1.0 should take 1 tick.
	// We'll tick until pos.X >= 1.0
	maxTicks := 100
	ticks := 0
	crossedBoundary := false

	for ; ticks < maxTicks; ticks++ {
		movSystem.Update(&world)

		if pos.X >= 1.0 && !crossedBoundary {
			crossedBoundary = true
			break
		}
	}

	if !crossedBoundary {
		t.Fatalf("Entity never crossed tile boundary into x >= 1.0 after %d ticks (pos=%f)", maxTicks, pos.X)
	}

	// Since it crossed into Tile (1,0), that tile's FootTraffic should have incremented.
	if mapGrid.TileStates[tileIndex].FootTraffic != 1 {
		t.Errorf("Expected FootTraffic on tile %d to be 1, got %d", tileIndex, mapGrid.TileStates[tileIndex].FootTraffic)
	}

	// Ensure the starting tile (0,0) didn't get traffic incremented incorrectly
	if mapGrid.TileStates[0].FootTraffic != 0 {
		t.Errorf("Expected FootTraffic on starting tile 0 to be 0, got %d", mapGrid.TileStates[0].FootTraffic)
	}

	// Now artificially boost FootTraffic on the mountain tile to see movement cost reduction
	mapGrid.TileStates[tileIndex].FootTraffic = 5000
	reducedCost := engine.GetEffectiveMovementCost(engine.BiomeMountain, mapGrid.TileStates[tileIndex].FootTraffic)

	if reducedCost >= initialCost {
		t.Errorf("Expected movement cost to reduce with FootTraffic. Initial: %f, Reduced: %f", initialCost, reducedCost)
	}
}
