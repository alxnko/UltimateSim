package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.1: Maritime Reach & Naval Logistics
// NavalSpawningSystem converts an overland CaravanEntity arriving at a PortComponent
// into a ShipComponent entity, transferring the Payload and Passengers.

type caravanData struct {
	entity  ecs.Entity
	pos     *components.Position
	payload *components.Payload
}

type NavalSpawningSystem struct {
	toConvert []caravanData // Pre-allocated slice for deterministic DOD iteration
}

func NewNavalSpawningSystem() *NavalSpawningSystem {
	return &NavalSpawningSystem{
		toConvert: make([]caravanData, 0, 100),
	}
}

func (s *NavalSpawningSystem) Update(world *ecs.World) {
	// Step 1: Pre-calculate all Ports locations
	villageID := ecs.ComponentID[components.Village](world)
	portID := ecs.ComponentID[components.PortComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	portFilter := ecs.All(villageID, portID, posID)
	portQuery := world.Query(portFilter)

	// Combine X and Y into a single uint64 key to ensure O(1) DOD hash matching
	portLocations := make(map[uint64]bool)
	for portQuery.Next() {
		pos := (*components.Position)(portQuery.Get(posID))
		key := (uint64(pos.X) << 32) | uint64(pos.Y)
		portLocations[key] = true
	}

	// Step 2: Iterate over Caravans
	caravanID := ecs.ComponentID[components.Caravan](world)
	velID := ecs.ComponentID[components.Velocity](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	caravanFilter := ecs.All(caravanID, posID, velID, payloadID)
	caravanQuery := world.Query(caravanFilter)

	s.toConvert = s.toConvert[:0]

	for caravanQuery.Next() {
		vel := (*components.Velocity)(caravanQuery.Get(velID))
		if vel.X != 0 || vel.Y != 0 {
			continue // Only convert when stopped (arrived)
		}

		pos := (*components.Position)(caravanQuery.Get(posID))
		key := (uint64(pos.X) << 32) | uint64(pos.Y)

		if portLocations[key] {
			payload := (*components.Payload)(caravanQuery.Get(payloadID))
			s.toConvert = append(s.toConvert, caravanData{
				entity:  caravanQuery.Entity(),
				pos:     pos,
				payload: payload,
			})
		}
	}

	// Entity Instantiation & Removal outside the loop
	if len(s.toConvert) == 0 {
		return
	}

	shipID := ecs.ComponentID[components.ShipComponent](world)
	passengerID := ecs.ComponentID[components.PassengerComponent](world)
	pathID := ecs.ComponentID[components.Path](world)

	for _, c := range s.toConvert {
		// Spawn new ship entity
		shipEntity := world.NewEntity(shipID, posID, velID, payloadID, passengerID, pathID)

		// Set Position
		newPos := (*components.Position)(world.Get(shipEntity, posID))
		*newPos = *c.pos

		// Set Velocity
		newVel := (*components.Velocity)(world.Get(shipEntity, velID))
		newVel.X = 0
		newVel.Y = 0

		// Set Payload
		newPayload := (*components.Payload)(world.Get(shipEntity, payloadID))
		*newPayload = *c.payload

		// Set Passengers
		newPassenger := (*components.PassengerComponent)(world.Get(shipEntity, passengerID))
		newPassenger.Passengers = make([]components.Passenger, 0)

		// Initialize Routing Path
		newPath := (*components.Path)(world.Get(shipEntity, pathID))
		newPath.HasPath = false
		newPath.Nodes = make([]components.Position, 0)

		// Despawn old Caravan
		world.RemoveEntity(c.entity)
	}
}
