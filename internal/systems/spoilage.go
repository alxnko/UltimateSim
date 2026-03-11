package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.2: Dynamic Attrition - SpoilageSystem
// Evaluates biological StorageComponent and Payload trackers (Food).
// Decrements Food limits by 5% every 10 ticks to simulate spoilage.

type SpoilageSystem struct {
	ticks uint64
}

// NewSpoilageSystem creates a new SpoilageSystem.
func NewSpoilageSystem() *SpoilageSystem {
	return &SpoilageSystem{
		ticks: 0,
	}
}

// Update executes the system logic per tick.
func (s *SpoilageSystem) Update(world *ecs.World) {
	s.ticks++
	if s.ticks%10 != 0 {
		return
	}

	storageID := ecs.ComponentID[components.StorageComponent](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	// Update StorageComponent (e.g., Villages)
	storageFilter := ecs.All(storageID)
	storageQuery := world.Query(storageFilter)
	for storageQuery.Next() {
		storage := (*components.StorageComponent)(storageQuery.Get(storageID))
		if storage.Food > 0 {
			// 5% spoilage: cast to uint64 to prevent overflow on very large integers
			newFood := uint32((uint64(storage.Food) * 95) / 100)
			storage.Food = newFood
		}
	}

	// Update Payload (e.g., Caravans)
	payloadFilter := ecs.All(payloadID)
	payloadQuery := world.Query(payloadFilter)
	for payloadQuery.Next() {
		payload := (*components.Payload)(payloadQuery.Get(payloadID))
		if payload.Food > 0 {
			// 5% spoilage
			newFood := uint32((uint64(payload.Food) * 95) / 100)
			payload.Food = newFood
		}
	}
}
