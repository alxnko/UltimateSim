package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 17.3: Maritime Attrition & Piracy
// NavalPiracySystem handles Rogue entities (no CityID) seeking out high-wealth ShipComponents
// aggressively to path towards them and execute piracy.

type targetShip struct {
	entity ecs.Entity
	x      float32
	y      float32
	wealth uint32
}

type NavalPiracySystem struct {
	targetShips []targetShip // Pre-allocated slice for DOD iteration and O(1) matching
}

func NewNavalPiracySystem() *NavalPiracySystem {
	return &NavalPiracySystem{
		targetShips: make([]targetShip, 0, 100),
	}
}

func (s *NavalPiracySystem) Update(world *ecs.World) {
	shipID := ecs.ComponentID[components.ShipComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	shipFilter := ecs.All(shipID, posID, payloadID)
	shipQuery := world.Query(shipFilter)

	s.targetShips = s.targetShips[:0]

	// Extract all ships to flat array for O(N*M) calculation without nested lock
	for shipQuery.Next() {
		pos := (*components.Position)(shipQuery.Get(posID))
		payload := (*components.Payload)(shipQuery.Get(payloadID))

		wealth := payload.Food + payload.Wood + payload.Stone + payload.Iron
		if wealth > 0 { // Only target ships with actual cargo
			s.targetShips = append(s.targetShips, targetShip{
				entity: shipQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				wealth: wealth,
			})
		}
	}

	if len(s.targetShips) == 0 {
		return
	}

	npcID := ecs.ComponentID[components.NPC](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	pathID := ecs.ComponentID[components.Path](world)

	// We query NPCs that are pirates (no City affiliation)
	rogueFilter := ecs.All(npcID, affilID, posID, pathID)
	rogueQuery := world.Query(rogueFilter)

	for rogueQuery.Next() {
		affil := (*components.Affiliation)(rogueQuery.Get(affilID))
		if affil.CityID != 0 {
			continue // Only rogue entities (CityID == 0) become pirates
		}

		roguePos := (*components.Position)(rogueQuery.Get(posID))
		path := (*components.Path)(rogueQuery.Get(pathID))

		var bestTarget targetShip
		var bestScore float32 = -1.0 // Higher is better (wealth / distance^2)

		// O(1) loop over flat ships slice
		for _, ship := range s.targetShips {
			dx := ship.x - roguePos.X
			dy := ship.y - roguePos.Y
			distSq := dx*dx + dy*dy

			if distSq == 0 {
				distSq = 1 // Prevent division by zero
			}

			// Value score: wealth over squared distance
			score := float32(ship.wealth) / distSq
			if score > bestScore {
				bestScore = score
				bestTarget = ship
			}
		}

		if bestScore > 0 {
			// Update path target
			path.TargetX = bestTarget.x
			path.TargetY = bestTarget.y
		}
	}
}
