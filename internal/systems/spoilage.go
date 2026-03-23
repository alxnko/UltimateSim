package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 09.2: Dynamic Attrition - SpoilageSystem
// Evaluates biological StorageComponent and Payload trackers (Food).
// Decrements Food limits by 5% every 10 ticks to simulate spoilage.
// Evolution: Phase 48 - The Ecological Rot & Plague Bridge

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
	posID := ecs.ComponentID[components.Position](world)
	diseaseID := ecs.ComponentID[components.DiseaseEntity](world)

	// Keep track of new diseases to spawn to avoid modifying ECS while querying
	type plagueSpawn struct {
		x, y      float32
		lethality uint8
	}
	var newPlagues []plagueSpawn

	// Update StorageComponent (e.g., Villages)
	// We check for StorageComponent alone to support tests that don't add Position,
	// but we only attempt plague spawn if Position is present.
	storageFilter := ecs.All(storageID)
	storageQuery := world.Query(storageFilter)
	for storageQuery.Next() {
		storage := (*components.StorageComponent)(storageQuery.Get(storageID))
		if storage.Food > 0 {
			// 5% spoilage: cast to uint64 to prevent overflow on very large integers
			newFood := uint32((uint64(storage.Food) * 95) / 100)

			// Force at least 1 unit to spoil if Food > 0, mimicking existing test behavior
			if newFood == storage.Food {
			    newFood--
			}

			spoiledAmount := storage.Food - newFood
			storage.Food = newFood

			// Phase 48: The Ecological Rot & Plague Bridge
			// Massive spoilage (e.g., hoarding during famine) triggers DiseaseEntity
			if spoiledAmount > 500 && world.Has(storageQuery.Entity(), posID) {
				pos := (*components.Position)(storageQuery.Get(posID))
				// 1% chance per tick to spawn a plague if severe rot happens
				if engine.GetRandomFloat32() < 0.01 {
					lethality := uint8(20) // Base lethality
					// Scale lethality with the amount of rot
					if spoiledAmount > 2000 {
						lethality = 50
					} else if spoiledAmount > 1000 {
						lethality = 30
					}

					newPlagues = append(newPlagues, plagueSpawn{
						x:         pos.X,
						y:         pos.Y,
						lethality: lethality,
					})
				}
			}
		}
	}

	// Update Payload (e.g., Caravans)
	// Caravans typically don't cause map-level plagues as they move, but they still spoil.
	payloadFilter := ecs.All(payloadID)
	payloadQuery := world.Query(payloadFilter)
	for payloadQuery.Next() {
		payload := (*components.Payload)(payloadQuery.Get(payloadID))
		if payload.Food > 0 {
			// 5% spoilage
			newFood := uint32((uint64(payload.Food) * 95) / 100)

			if newFood == payload.Food {
			    newFood--
			}
			payload.Food = newFood
		}
	}

	// Spawn the new Plagues outside the query loop
	for _, p := range newPlagues {
		ent := world.NewEntity(posID, diseaseID)
		pos := (*components.Position)(world.Get(ent, posID))
		pos.X = p.x
		pos.Y = p.y

		disease := (*components.DiseaseEntity)(world.Get(ent, diseaseID))
		// Phase 10.3: Plagues need a unique Disease ID
		disease.ID = uint32(engine.GetRandomInt())
		disease.Lethality = p.lethality
	}
}
