package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 10.2: Bureaucratic Delay (Administrative Entropy)
// AdministrativeDecaySystem tests transit time of an OrderEntity against the
// targeted CityID's Loyalty base integer. If Decay > Loyalty, the order is intercepted
// by rebellious vassals and terminates prematurely.

type AdministrativeDecaySystem struct {
	Tick     uint64
	toRemove []ecs.Entity
}

func NewAdministrativeDecaySystem() *AdministrativeDecaySystem {
	return &AdministrativeDecaySystem{
		Tick:     0,
		toRemove: make([]ecs.Entity, 0, 100),
	}
}

func (s *AdministrativeDecaySystem) Update(world *ecs.World) {
	s.Tick++

	orderEntityID := ecs.ComponentID[components.OrderEntity](world)
	orderCompID := ecs.ComponentID[components.OrderComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)

	// Step 1: Pre-calculate Loyalty values for all valid target Cities to avoid nested map/loops.
	// This preserves DOD iteration speed for the main loop.
	loyaltyMap := make(map[uint32]uint32)

	cityQuery := world.Query(filter.All(villageID, identID, loyaltyID))
	for cityQuery.Next() {
		ident := (*components.Identity)(cityQuery.Get(identID))
		loyalty := (*components.LoyaltyComponent)(cityQuery.Get(loyaltyID))
		loyaltyMap[uint32(ident.ID)] = loyalty.Value
	}

	// Step 2: Iterate over all OrderEntities evaluating decay
	orderFilter := ecs.All(orderEntityID, orderCompID, posID)
	orderQuery := world.Query(orderFilter)

	s.toRemove = s.toRemove[:0] // Clear slice to reuse capacity

	for orderQuery.Next() {
		order := (*components.OrderComponent)(orderQuery.Get(orderCompID))

		// If current tick is less than creation tick, skip (should not happen, but defensive)
		if s.Tick <= order.CreationTick {
			continue
		}

		decay := s.Tick - order.CreationTick

		if loyaltyVal, exists := loyaltyMap[order.TargetCityID]; exists {
			if uint32(decay) > loyaltyVal {
				// The transit time exceeds loyalty. The order fails.
				s.toRemove = append(s.toRemove, orderQuery.Entity())
			}
		} else {
			// If target city doesn't exist anymore (e.g., destroyed), despawn the order
			s.toRemove = append(s.toRemove, orderQuery.Entity())
		}
	}

	// Step 3: Despawn failed orders outside the query loop to prevent ECS panics
	for _, e := range s.toRemove {
		world.RemoveEntity(e)
	}
}
