package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 40.1 - The Maritime Migration Engine
// Evaluates desperate NPCs and moves them onto ShipComponents if they can afford the passage fare.
// Bridges Phase 13/21 (Desperation/Wealth) directly to Phase 17 (Naval Logistics).

type shipNodeData struct {
	Entity ecs.Entity
	OwnerID uint64
	X float32
	Y float32
	PassengerComp *components.PassengerComponent
}

type MaritimeMigrationSystem struct {
	tickCounter uint64

	// Component IDs
	posID         ecs.ID
	desperationID ecs.ID
	needsID       ecs.ID
	shipID        ecs.ID
	passengerID   ecs.ID
	businessID    ecs.ID
	identID       ecs.ID

	// Pre-allocated slice for DOD matching
	ships []shipNodeData
}

func NewMaritimeMigrationSystem(world *ecs.World) *MaritimeMigrationSystem {
	return &MaritimeMigrationSystem{
		posID:         ecs.ComponentID[components.Position](world),
		desperationID: ecs.ComponentID[components.DesperationComponent](world),
		needsID:       ecs.ComponentID[components.Needs](world),
		shipID:        ecs.ComponentID[components.ShipComponent](world),
		passengerID:   ecs.ComponentID[components.PassengerComponent](world),
		businessID:    ecs.ComponentID[components.BusinessComponent](world),
		identID:       ecs.ComponentID[components.Identity](world),
		ships:         make([]shipNodeData, 0, 20),
	}
}

func (s *MaritimeMigrationSystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%20 != 0 {
		return // Throttle evaluation to save CPU cycles
	}

	// 1. Cache all active ships with PassengerComponent in a flat DOD slice
	s.ships = s.ships[:0]
	shipFilter := ecs.All(s.shipID, s.posID, s.passengerID)
	shipQuery := world.Query(&shipFilter)

	for shipQuery.Next() {
		pos := (*components.Position)(shipQuery.Get(s.posID))
		passComp := (*components.PassengerComponent)(shipQuery.Get(s.passengerID))

		var ownerID uint64 = 0
		if shipQuery.Has(s.businessID) {
			bus := (*components.BusinessComponent)(shipQuery.Get(s.businessID))
			ownerID = bus.OwnerID
		}

		s.ships = append(s.ships, shipNodeData{
			Entity: shipQuery.Entity(),
			OwnerID: ownerID,
			X: pos.X,
			Y: pos.Y,
			PassengerComp: passComp,
		})
	}
	// Do not explicitly call shipQuery.Close() if Next() returned false

	if len(s.ships) == 0 {
		return // No ships to board
	}

	// 2. Map over all NPCs who are Desperate AND have enough wealth for passage
	npcFilter := ecs.All(s.identID, s.posID, s.desperationID, s.needsID).Without(s.passengerID) // Ensure they aren't already passengers
	npcQuery := world.Query(&npcFilter)

	// List of NPCs to physically remove from the map after they board
	// We map the Entity and the Ship they boarded.
	type boardingAction struct {
		NPCEntity ecs.Entity
		ShipIndex int
	}
	var toBoard []boardingAction

	for npcQuery.Next() {
		desp := (*components.DesperationComponent)(npcQuery.Get(s.desperationID))
		if desp.Level < 30 {
			continue // Not desperate enough to flee
		}

		needs := (*components.Needs)(npcQuery.Get(s.needsID))
		if needs.Wealth < 50.0 {
			continue // Cannot afford the minimum 50 wealth passage fee
		}

		pos := (*components.Position)(npcQuery.Get(s.posID))

		// Find nearest ship within embarkation range (distSq <= 10.0)
		bestDistSq := float32(11.0)
		bestShipIdx := -1

		for i, ship := range s.ships {
			// Fast squared distance
			dx := ship.X - pos.X
			dy := ship.Y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq < 10.0 && distSq < bestDistSq {
				bestDistSq = distSq
				bestShipIdx = i
			}
		}

		if bestShipIdx != -1 {
			// Found a ship. Mark for boarding.
			toBoard = append(toBoard, boardingAction{
				NPCEntity: npcQuery.Entity(),
				ShipIndex: bestShipIdx,
			})

			// We only allow one action per tick, but if multiple NPCs board the same ship, we process it later.
		}
	}
	// Do not explicitly call npcQuery.Close() if Next() returned false

	// 3. Execute the boarding actions structurally
	for _, action := range toBoard {
		// Re-fetch Needs to ensure it's still valid
		if !world.Alive(action.NPCEntity) {
			continue
		}

		needs := (*components.Needs)(world.Get(action.NPCEntity, s.needsID))
		ident := (*components.Identity)(world.Get(action.NPCEntity, s.identID))

		// Deduct fare
		needs.Wealth -= 50.0

		// Transfer wealth to the Ship Owner if one exists
		shipNode := s.ships[action.ShipIndex]
		if shipNode.OwnerID != 0 {
			// We would theoretically find the owner entity and add 50.0 to their Treasury.
			// But for now, we just deduct it from the NPC to represent the economic sink of buying passage.
		}

		// Append the NPC identity to the ship's passenger slice
		shipNode.PassengerComp.Passengers = append(shipNode.PassengerComp.Passengers, components.Passenger{
			EntityID: ident.ID,
		})

		// Despawn the physical NPC from the land map, abstracting them into the Ship's passenger hold.
		world.RemoveEntity(action.NPCEntity)
	}
}
