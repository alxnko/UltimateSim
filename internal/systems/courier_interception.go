package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 39.1: The Courier Interception Engine
// Bandits pathing the map actively intercept traversing OrderEntities (state couriers).
// The order is destroyed (preventing state execution) and the Bandit logs InteractionTheft
// triggering the Justice System (Phase 18) and Blood Feuds.

type CourierInterceptionSystem struct {
	banditFilter ecs.Filter
	orderFilter  ecs.Filter
	toRemove     []ecs.Entity
	toPunish     []ecs.Entity
}

func NewCourierInterceptionSystem(world *ecs.World) *CourierInterceptionSystem {
	posID := ecs.ComponentID[components.Position](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)

	banditMask := ecs.All(posID, jobID, memID, needsID)

	orderEntityID := ecs.ComponentID[components.OrderEntity](world)
	orderCompID := ecs.ComponentID[components.OrderComponent](world)

	orderMask := ecs.All(posID, orderEntityID, orderCompID)

	return &CourierInterceptionSystem{
		banditFilter: &banditMask,
		orderFilter:  &orderMask,
		toRemove:     make([]ecs.Entity, 0, 100),
		toPunish:     make([]ecs.Entity, 0, 100),
	}
}

func (s *CourierInterceptionSystem) Update(world *ecs.World) {
	s.toRemove = s.toRemove[:0]
	s.toPunish = s.toPunish[:0]

	// Step 1: Cache OrderEntities for DOD fast loop
	posID := ecs.ComponentID[components.Position](world)
	orderCompID := ecs.ComponentID[components.OrderComponent](world)

	type oData struct {
		Entity ecs.Entity
		X      float32
		Y      float32
		Order  *components.OrderComponent
	}

	orders := make([]oData, 0, 100)
	orderQuery := world.Query(s.orderFilter)
	for orderQuery.Next() {
		pos := (*components.Position)(orderQuery.Get(posID))
		order := (*components.OrderComponent)(orderQuery.Get(orderCompID))
		orders = append(orders, oData{
			Entity: orderQuery.Entity(),
			X:      pos.X,
			Y:      pos.Y,
			Order:  order,
		})
	}

	if len(orders) == 0 {
		return
	}

	// Step 2: Iterate through Bandits
	jobID := ecs.ComponentID[components.JobComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)

	banditQuery := world.Query(s.banditFilter)
	for banditQuery.Next() {
		job := (*components.JobComponent)(banditQuery.Get(jobID))
		if job.JobID != components.JobBandit {
			continue
		}

		pos := (*components.Position)(banditQuery.Get(posID))

		// Find nearest Order
		var bestO *oData
		var bestDist float32 = 9999999.0
		var bestIdx int = -1

		for i := 0; i < len(orders); i++ {
			o := &orders[i]

			// Check if already removed
			marked := false
			for _, rm := range s.toRemove {
				if rm == o.Entity {
					marked = true
					break
				}
			}
			if marked {
				continue
			}

			dx := pos.X - o.X
			dy := pos.Y - o.Y
			distSq := (dx * dx) + (dy * dy)

			if distSq < bestDist {
				bestDist = distSq
				bestO = o
				bestIdx = i
			}
		}

		// Intercept if close enough
		if bestO != nil && bestDist <= 2.0 {
			mem := (*components.Memory)(banditQuery.Get(memID))
			needs := (*components.Needs)(banditQuery.Get(needsID))

			// Log crime (Theft of State Secrets)
			event := components.MemoryEvent{
				TargetID:        0, // Target is OrderEntity, 0 is fine
				InteractionType: components.InteractionTheft,
				Value:           int32(bestO.Order.TargetCityID), // Secret data value
				TickStamp:       0,
			}
			mem.Events[mem.Head] = event
			mem.Head = (mem.Head + 1) % uint8(len(mem.Events))

			// Bandit gains wealth via bribery/blackmail potential from stolen orders
			needs.Wealth += 50.0

			// Mark order for removal and bandit for punishment
			s.toRemove = append(s.toRemove, bestO.Entity)
			s.toPunish = append(s.toPunish, banditQuery.Entity())

			// Remove from cache
			if bestIdx != -1 {
				orders[bestIdx] = orders[len(orders)-1]
				orders = orders[:len(orders)-1]
			}
		}
	}

	// Step 3: Mutate locked world
	for _, e := range s.toRemove {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}

	crimeID := ecs.ComponentID[components.CrimeMarker](world)
	for _, e := range s.toPunish {
		if world.Alive(e) {
			if !world.Has(e, crimeID) {
				world.Add(e, crimeID)
			}
			cm := (*components.CrimeMarker)(world.Get(e, crimeID))
			cm.CrimeLevel = 2 // Higher level crime for intercepting state orders
			cm.Bounty = 500   // Massive state bounty
		}
	}
}
