package systems_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.1: Maritime Reach DOD Constraints Verification
func TestNavalSpawningSystem_Deterministic(t *testing.T) {
	world1 := ecs.NewWorld()
	world2 := ecs.NewWorld()

	// Setup systems
	sys1 := systems.NewNavalSpawningSystem()
	sys2 := systems.NewNavalSpawningSystem()

	// Register Component IDs World 1
	vID := ecs.ComponentID[components.Village](&world1)
	pID := ecs.ComponentID[components.Position](&world1)
	portID := ecs.ComponentID[components.PortComponent](&world1)
	caravanID := ecs.ComponentID[components.Caravan](&world1)
	velID := ecs.ComponentID[components.Velocity](&world1)
	payloadID := ecs.ComponentID[components.Payload](&world1)
	shipID := ecs.ComponentID[components.ShipComponent](&world1)
	passID := ecs.ComponentID[components.PassengerComponent](&world1)
	pathID := ecs.ComponentID[components.Path](&world1)

	// Register Component IDs World 2
	vID2 := ecs.ComponentID[components.Village](&world2)
	pID2 := ecs.ComponentID[components.Position](&world2)
	portID2 := ecs.ComponentID[components.PortComponent](&world2)
	caravanID2 := ecs.ComponentID[components.Caravan](&world2)
	velID2 := ecs.ComponentID[components.Velocity](&world2)
	payloadID2 := ecs.ComponentID[components.Payload](&world2)
	shipID2 := ecs.ComponentID[components.ShipComponent](&world2)
	passID2 := ecs.ComponentID[components.PassengerComponent](&world2)
	pathID2 := ecs.ComponentID[components.Path](&world2)

	// Avoid un-used variable errors
	_ = passID
	_ = pathID
	_ = passID2
	_ = pathID2

	// --- World 1 Setup ---
	// Village with Port
	v1 := world1.NewEntity(vID, pID, portID)
	vp1 := (*components.Position)(world1.Get(v1, pID))
	vp1.X, vp1.Y = 10, 10

	// Arrived Caravan
	c1 := world1.NewEntity(caravanID, pID, velID, payloadID)
	cp1 := (*components.Position)(world1.Get(c1, pID))
	cp1.X, cp1.Y = 10, 10
	cv1 := (*components.Velocity)(world1.Get(c1, velID))
	cv1.X, cv1.Y = 0, 0
	cpld1 := (*components.Payload)(world1.Get(c1, payloadID))
	cpld1.Wood = 50

	// Moving Caravan
	c2 := world1.NewEntity(caravanID, pID, velID, payloadID)
	cp2 := (*components.Position)(world1.Get(c2, pID))
	cp2.X, cp2.Y = 10, 10
	cv2 := (*components.Velocity)(world1.Get(c2, velID))
	cv2.X, cv2.Y = 1, 0 // Non-zero velocity

	// --- World 2 Setup ---
	// Village with Port
	v2 := world2.NewEntity(vID2, pID2, portID2)
	vp2 := (*components.Position)(world2.Get(v2, pID2))
	vp2.X, vp2.Y = 10, 10

	// Arrived Caravan
	c1_2 := world2.NewEntity(caravanID2, pID2, velID2, payloadID2)
	cp1_2 := (*components.Position)(world2.Get(c1_2, pID2))
	cp1_2.X, cp1_2.Y = 10, 10
	cv1_2 := (*components.Velocity)(world2.Get(c1_2, velID2))
	cv1_2.X, cv1_2.Y = 0, 0
	cpld1_2 := (*components.Payload)(world2.Get(c1_2, payloadID2))
	cpld1_2.Wood = 50

	// Moving Caravan
	c2_2 := world2.NewEntity(caravanID2, pID2, velID2, payloadID2)
	cp2_2 := (*components.Position)(world2.Get(c2_2, pID2))
	cp2_2.X, cp2_2.Y = 10, 10
	cv2_2 := (*components.Velocity)(world2.Get(c2_2, velID2))
	cv2_2.X, cv2_2.Y = 1, 0

	// Run systems
	sys1.Update(&world1)
	sys2.Update(&world2)

	// Verify World 1
	if world1.Alive(c1) {
		t.Errorf("Expected Caravan 1 to be despawned")
	}
	if !world1.Alive(c2) {
		t.Errorf("Expected Caravan 2 to still be alive (was moving)")
	}

	// Check for Ship creation in World 1
	filter1 := ecs.All(shipID, pID, payloadID)
	query1 := world1.Query(filter1)
	shipCount1 := 0
	var woodAmount1 uint32
	for query1.Next() {
		shipCount1++
		pld := (*components.Payload)(query1.Get(payloadID))
		woodAmount1 = pld.Wood
	}
	if shipCount1 != 1 {
		t.Errorf("Expected exactly 1 Ship in World 1, got %d", shipCount1)
	}
	if woodAmount1 != 50 {
		t.Errorf("Expected transferred payload to be 50 Wood, got %d", woodAmount1)
	}

	// Verify World 2 (Determinism)
	if world2.Alive(c1_2) {
		t.Errorf("Expected Caravan 1 to be despawned in World 2")
	}
	if !world2.Alive(c2_2) {
		t.Errorf("Expected Caravan 2 to still be alive in World 2 (was moving)")
	}

	filter2 := ecs.All(shipID2, pID2, payloadID2)
	query2 := world2.Query(filter2)
	shipCount2 := 0
	var woodAmount2 uint32
	for query2.Next() {
		shipCount2++
		pld := (*components.Payload)(query2.Get(payloadID2))
		woodAmount2 = pld.Wood
	}
	if shipCount2 != 1 {
		t.Errorf("Expected exactly 1 Ship in World 2, got %d", shipCount2)
	}
	if woodAmount2 != 50 {
		t.Errorf("Expected transferred payload to be 50 Wood in World 2, got %d", woodAmount2)
	}
}
