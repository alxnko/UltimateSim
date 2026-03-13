package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.2: Oceanic Pathfinding Deterministic Tests
func TestNavalRoutingSystem_Determinism(t *testing.T) {
	// Initialize two parallel ECS Worlds to verify exact determinism
	world1 := ecs.NewWorld()
	world2 := ecs.NewWorld()

	mapGrid1 := engine.NewMapGrid(100, 100)
	mapGrid2 := engine.NewMapGrid(100, 100)

	for i := range mapGrid1.Tiles {
		mapGrid1.Tiles[i].BiomeID = engine.BiomeOcean
		mapGrid2.Tiles[i].BiomeID = engine.BiomeOcean
	}

	queue1 := engine.NewPathRequestQueue(10, 1)
	queue2 := engine.NewPathRequestQueue(10, 1)

	queue1.StartWorkers()
	queue2.StartWorkers()
	defer queue1.Close()
	defer queue2.Close()

	calendar1 := &engine.Calendar{IsWinter: false, Ticks: 0}
	calendar2 := &engine.Calendar{IsWinter: false, Ticks: 0}

	sys1 := NewNavalRoutingSystem(&world1, mapGrid1, queue1, calendar1)
	sys2 := NewNavalRoutingSystem(&world2, mapGrid2, queue2, calendar2)

	// Set up Ship components for world1
	shipID1 := ecs.ComponentID[components.ShipComponent](&world1)
	posID1 := ecs.ComponentID[components.Position](&world1)
	pathID1 := ecs.ComponentID[components.Path](&world1)
	idID1 := ecs.ComponentID[components.Identity](&world1)

	e1 := world1.NewEntity(shipID1, posID1, pathID1, idID1)
	pos1 := (*components.Position)(world1.Get(e1, posID1))
	pos1.X = 10
	pos1.Y = 10
	path1 := (*components.Path)(world1.Get(e1, pathID1))
	path1.TargetX = 50
	path1.TargetY = 50
	ident1 := (*components.Identity)(world1.Get(e1, idID1))
	ident1.ID = 1

	// Set up Ship components for world2
	shipID2 := ecs.ComponentID[components.ShipComponent](&world2)
	posID2 := ecs.ComponentID[components.Position](&world2)
	pathID2 := ecs.ComponentID[components.Path](&world2)
	idID2 := ecs.ComponentID[components.Identity](&world2)

	e2 := world2.NewEntity(shipID2, posID2, pathID2, idID2)
	pos2 := (*components.Position)(world2.Get(e2, posID2))
	pos2.X = 10
	pos2.Y = 10
	path2 := (*components.Path)(world2.Get(e2, pathID2))
	path2.TargetX = 50
	path2.TargetY = 50
	ident2 := (*components.Identity)(world2.Get(e2, idID2))
	ident2.ID = 1

	// Tick 1: Route request should be registered synchronously
	sys1.Update(&world1)
	sys2.Update(&world2)

	// Tick 2: Evaluate synchronously in map
	sys1.Update(&world1)
	sys2.Update(&world2)

	// Verify both nodes have perfectly identical path results
	resPath1 := (*components.Path)(world1.Get(e1, pathID1))
	resPath2 := (*components.Path)(world2.Get(e2, pathID2))

	if len(resPath1.Nodes) == 0 {
		t.Fatalf("Expected path nodes to be generated, got 0")
	}

	if len(resPath1.Nodes) != len(resPath2.Nodes) {
		t.Fatalf("Deterministic mismatch in path lengths: %d vs %d", len(resPath1.Nodes), len(resPath2.Nodes))
	}

	for i := range resPath1.Nodes {
		if resPath1.Nodes[i].X != resPath2.Nodes[i].X || resPath1.Nodes[i].Y != resPath2.Nodes[i].Y {
			t.Errorf("Deterministic mismatch at node %d: %+v != %+v", i, resPath1.Nodes[i], resPath2.Nodes[i])
		}
	}
}

// Phase 17.2: Winter logic should prevent routing updates
func TestNavalRoutingSystem_WinterBlock(t *testing.T) {
	world := ecs.NewWorld()
	mapGrid := engine.NewMapGrid(10, 10)
	queue := engine.NewPathRequestQueue(10, 1)
	queue.StartWorkers()
	defer queue.Close()

	calendar := &engine.Calendar{IsWinter: true} // Frozen oceans prevent ships moving

	sys := NewNavalRoutingSystem(&world, mapGrid, queue, calendar)

	shipID := ecs.ComponentID[components.ShipComponent](&world)
	posID := ecs.ComponentID[components.Position](&world)
	pathID := ecs.ComponentID[components.Path](&world)
	idID := ecs.ComponentID[components.Identity](&world)

	e := world.NewEntity(shipID, posID, pathID, idID)
	pos := (*components.Position)(world.Get(e, posID))
	pos.X = 0
	pos.Y = 0
	path := (*components.Path)(world.Get(e, pathID))
	path.TargetX = 5
	path.TargetY = 5

	sys.Update(&world)
	sys.Update(&world) // Second tick evaluation

	// Should NOT have received a path because winter blocked the queue
	resPath := (*components.Path)(world.Get(e, pathID))
	if len(resPath.Nodes) > 0 {
		t.Errorf("Expected 0 nodes during winter blockage, got %d", len(resPath.Nodes))
	}
}
