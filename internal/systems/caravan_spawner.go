package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.1 & 13.1: The Caravan Entity & Price Discovery
// Spawns CaravanEntity if a VillageEntity processes extreme local disparity signals
// mapped via MarketComponent pricing boundaries.

// cache values + the entity handle, not component pointers — GC corruption
// class, see banditry.go. Storage is written through, so it is re-fetched via
// world.Get at use time; position is read-only and copied by value.
type villageData struct {
	entity ecs.Entity
	x      float32
	y      float32
}

type CaravanSpawnerSystem struct {
	toSpawn []villageData
}

func NewCaravanSpawnerSystem() *CaravanSpawnerSystem {
	return &CaravanSpawnerSystem{
		toSpawn: make([]villageData, 0, 100),
	}
}

func (s *CaravanSpawnerSystem) Update(world *ecs.World) {
	villageID := ecs.ComponentID[components.Village](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	filter := ecs.All(villageID, storageID, marketID, posID)
	query := world.Query(filter)

	s.toSpawn = s.toSpawn[:0] // Clear slice to reuse capacity

	for query.Next() {
		market := (*components.MarketComponent)(query.Get(marketID))
		pos := (*components.Position)(query.Get(posID))

		// Phase 13.1: Market logic bounds trigger
		// Float thresholds dictating extreme famine or need
		if market.FoodPrice > 10.0 {
			// Copy the values out of the Next loop; storage is re-fetched by
			// entity handle at spawn time to deduct Wood
			s.toSpawn = append(s.toSpawn, villageData{
				entity: query.Entity(),
				x:      pos.X,
				y:      pos.Y,
			})
		}
	}

	// Entity Bind & Instantiation outside Next loop to prevent concurrent modifications
	// Check if we need to spawn anything before doing ID lookups
	if len(s.toSpawn) == 0 {
		return
	}

	caravanID := ecs.ComponentID[components.Caravan](world)
	velID := ecs.ComponentID[components.Velocity](world)
	payloadID := ecs.ComponentID[components.Payload](world)
	pathID := ecs.ComponentID[components.Path](world)

	for _, v := range s.toSpawn {
		if !world.Alive(v.entity) {
			continue
		}
		// Re-fetch after any prior structural change (NewEntity below) — cached
		// component pointers do not survive archetype moves.
		storage := (*components.StorageComponent)(world.Get(v.entity, storageID))

		// Calculate potential payload limit
		var woodToTransfer uint32 = 0
		if storage.Wood > 50 {
			woodToTransfer = 50
			storage.Wood -= 50
		}

		// Instantiate a CaravanEntity
		caravanEntity := world.NewEntity(caravanID, posID, velID, payloadID, pathID)

		// Set Position (copying from Village)
		newPos := (*components.Position)(world.Get(caravanEntity, posID))
		newPos.X = v.x
		newPos.Y = v.y

		// Set Velocity (initialize)
		newVel := (*components.Velocity)(world.Get(caravanEntity, velID))
		newVel.X = 0
		newVel.Y = 0

		// Set Payload
		newPayload := (*components.Payload)(world.Get(caravanEntity, payloadID))
		newPayload.Wood = woodToTransfer
		newPayload.Stone = 0
		newPayload.Iron = 0
		newPayload.Food = 0

		// Initialize Routing Path
		newPath := (*components.Path)(world.Get(caravanEntity, pathID))
		newPath.HasPath = false
		newPath.Nodes = make([]components.Position, 0)
	}
}
