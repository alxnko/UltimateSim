package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.2: Dynamic Attrition - RustSystem
// Evaluates non-biological StorageComponent and Payload parameters (Iron, Wood, Stone).
// Decrements non-biological limits by 2% every 50 ticks to simulate rust and degradation.

type RustSystem struct {
	ticks uint64
}

// NewRustSystem creates a new RustSystem.
func NewRustSystem() *RustSystem {
	return &RustSystem{
		ticks: 0,
	}
}

// Update executes the system logic per tick.
func (s *RustSystem) Update(world *ecs.World) {
	s.ticks++
	if s.ticks%50 != 0 {
		return
	}

	storageID := ecs.ComponentID[components.StorageComponent](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	// Update StorageComponent
	storageFilter := ecs.All(storageID)
	storageQuery := world.Query(storageFilter)
	for storageQuery.Next() {
		storage := (*components.StorageComponent)(storageQuery.Get(storageID))

		if storage.Iron > 0 {
			newIron := uint32((uint64(storage.Iron) * 98) / 100)
			storage.Iron = newIron
		}
		if storage.Wood > 0 {
			newWood := uint32((uint64(storage.Wood) * 98) / 100)
			storage.Wood = newWood
		}
		// Stone typically doesn't rust or decay as fast, but including it for general attrition
		if storage.Stone > 0 {
			newStone := uint32((uint64(storage.Stone) * 98) / 100)
			storage.Stone = newStone
		}
	}

	// Update Payload
	payloadFilter := ecs.All(payloadID)
	payloadQuery := world.Query(payloadFilter)
	for payloadQuery.Next() {
		payload := (*components.Payload)(payloadQuery.Get(payloadID))

		if payload.Iron > 0 {
			newIron := uint32((uint64(payload.Iron) * 98) / 100)
			payload.Iron = newIron
		}
		if payload.Wood > 0 {
			newWood := uint32((uint64(payload.Wood) * 98) / 100)
			payload.Wood = newWood
		}
		if payload.Stone > 0 {
			newStone := uint32((uint64(payload.Stone) * 98) / 100)
			payload.Stone = newStone
		}
	}
}
