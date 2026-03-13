package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.3: Maritime Attrition & Piracy
// TestNavalPiracySystem_Deterministic ensures that the NavalPiracySystem assigns rogue entities
// identical path targets to high-wealth ShipComponents across seeded runs.

func TestNavalPiracySystem_Deterministic(t *testing.T) {
	setupWorld := func() (*ecs.World, ecs.Entity, ecs.Entity, ecs.Entity) {
		world := ecs.NewWorld()

		shipID := ecs.ComponentID[components.ShipComponent](&world)
		posID := ecs.ComponentID[components.Position](&world)
		payloadID := ecs.ComponentID[components.Payload](&world)
		npcID := ecs.ComponentID[components.NPC](&world)
		affilID := ecs.ComponentID[components.Affiliation](&world)
		pathID := ecs.ComponentID[components.Path](&world)

		// Ship 1: High Wealth (Far)
		ship1 := world.NewEntity(shipID, posID, payloadID)
		(*components.Position)(world.Get(ship1, posID)).X = 100
		(*components.Position)(world.Get(ship1, posID)).Y = 100
		(*components.Payload)(world.Get(ship1, payloadID)).Food = 500

		// Ship 2: Low Wealth (Close)
		ship2 := world.NewEntity(shipID, posID, payloadID)
		(*components.Position)(world.Get(ship2, posID)).X = 10
		(*components.Position)(world.Get(ship2, posID)).Y = 10
		(*components.Payload)(world.Get(ship2, payloadID)).Wood = 50

		// Rogue NPC (Pirate)
		rogue := world.NewEntity(npcID, affilID, posID, pathID)
		(*components.Affiliation)(world.Get(rogue, affilID)).CityID = 0
		(*components.Position)(world.Get(rogue, posID)).X = 0
		(*components.Position)(world.Get(rogue, posID)).Y = 0

		path := (*components.Path)(world.Get(rogue, pathID))
		path.TargetX = 0
		path.TargetY = 0

		return &world, ship1, ship2, rogue
	}

	runSystem := func(world *ecs.World) {
		sys := NewNavalPiracySystem()
		sys.Update(world)
	}

	engine.InitializeRNG([32]byte{1, 2, 3})
	world1, ship1_1, ship1_2, rogue1 := setupWorld()
	runSystem(world1)

	engine.InitializeRNG([32]byte{1, 2, 3})
	world2, ship2_1, ship2_2, rogue2 := setupWorld()
	runSystem(world2)

	// Verify Target assignment determinism
	pathID1 := ecs.ComponentID[components.Path](world1)
	path1 := (*components.Path)(world1.Get(rogue1, pathID1))

	pathID2 := ecs.ComponentID[components.Path](world2)
	path2 := (*components.Path)(world2.Get(rogue2, pathID2))

	if path1.TargetX != path2.TargetX || path1.TargetY != path2.TargetY {
		t.Errorf("Path targets mismatch: run1=(%f, %f), run2=(%f, %f)", path1.TargetX, path1.TargetY, path2.TargetX, path2.TargetY)
	}

	// Verify they targeted the correct ship based on scoring math
	// Ship 1 Score: 500 / (10000+10000) = 500 / 20000 = 0.025
	// Ship 2 Score: 50 / (100+100) = 50 / 200 = 0.25
	// So Ship 2 should be targeted as the score is higher!
	if path1.TargetX != 10 || path1.TargetY != 10 {
		t.Errorf("Expected path target to be (10, 10), got (%f, %f)", path1.TargetX, path1.TargetY)
	}

	_ = ship1_1
	_ = ship1_2
	_ = ship2_1
	_ = ship2_2
}
