package systems

import (
	"log"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: DeathSystem
// Scans for any Entity where Needs.Food <= 0. If found, trigger the Despawn pipeline.

// itemSpawnData holds temporary data extracted from a dying entity to spawn a Legend Item
type itemSpawnData struct {
	posX     float32
	posY     float32
	nameID   uint32
	prestige uint32
}

type DeathSystem struct {
	filter       ecs.Filter
	toRemove     []ecs.Entity
	itemsToSpawn []itemSpawnData
	deadPos      []components.Position // Phase 20.3: Used for mapping trauma
}

// NewDeathSystem creates a new DeathSystem.
func NewDeathSystem(world *ecs.World) *DeathSystem {
	// Query entities that have Needs
	needsID := ecs.ComponentID[components.Needs](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	// Phase 05.3: Arche-Go Component Filters
	// Explicitly build Without(ruinID) to skip over ruins.
	mask := ecs.All(needsID).Without(ruinID)

	return &DeathSystem{
		filter:       &mask,
		toRemove:     make([]ecs.Entity, 0, 100),
		itemsToSpawn: make([]itemSpawnData, 0, 10),
		deadPos:      make([]components.Position, 0, 100),
	}
}

// Update executes the system logic per tick.
func (s *DeathSystem) Update(world *ecs.World) {
	needsID := ecs.ComponentID[components.Needs](world)
	legacyID := ecs.ComponentID[components.Legacy](world)
	identityID := ecs.ComponentID[components.Identity](world)
	positionID := ecs.ComponentID[components.Position](world)

	// Collect entities to remove to avoid modifying the world while iterating
	// Reset the slice length to 0, retaining capacity to avoid GC pressure
	s.toRemove = s.toRemove[:0]
	s.itemsToSpawn = s.itemsToSpawn[:0]
	s.deadPos = s.deadPos[:0]

	query := world.Query(s.filter)
	for query.Next() {
		needs := (*components.Needs)(query.Get(needsID))

		if needs.Food <= 0 {
			s.toRemove = append(s.toRemove, query.Entity())

			var posX, posY float32
			if query.Has(positionID) {
				pos := (*components.Position)(query.Get(positionID))
				posX = pos.X
				posY = pos.Y
				s.deadPos = append(s.deadPos, *pos)
			}

			// Phase 09.5: Item Inheritance logic
			if query.Has(legacyID) {
				legacy := (*components.Legacy)(query.Get(legacyID))
				if legacy.Prestige >= components.ExtremePrestigeThreshold {
					var nameID uint32 = 0
					if query.Has(identityID) {
						ident := (*components.Identity)(query.Get(identityID))
						registry := engine.GetSecretRegistry()
						nameID = registry.RegisterSecret(ident.Name)
					}

					s.itemsToSpawn = append(s.itemsToSpawn, itemSpawnData{
						posX:     posX,
						posY:     posY,
						prestige: legacy.Prestige,
						nameID:   nameID,
					})
				}
			}

			// log root causes to standard output
			// ecs.Entity formats safely to string via %v
			log.Printf("Entity %v despawned due to starvation (Food <= 0)", query.Entity())
		}
	}

	// Phase 20.3: Traumatic Traditions
	// Map dead positions to jurisdictions to increment Trauma
	if len(s.deadPos) > 0 {
		jurID := ecs.ComponentID[components.JurisdictionComponent](world)
		posID := ecs.ComponentID[components.Position](world)
		jurQuery := world.Query(ecs.All(jurID, posID))

		type jurData struct {
			comp *components.JurisdictionComponent
			x    float32
			y    float32
		}
		jurisdictions := make([]jurData, 0, 20)
		for jurQuery.Next() {
			jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))
			p := (*components.Position)(jurQuery.Get(posID))
			jurisdictions = append(jurisdictions, jurData{
				comp: jur,
				x:    p.X,
				y:    p.Y,
			})
		}

		for _, dp := range s.deadPos {
			for i := 0; i < len(jurisdictions); i++ {
				j := &jurisdictions[i]
				dx := dp.X - j.x
				dy := dp.Y - j.y
				if dx*dx+dy*dy <= j.comp.RadiusSquared {
					if j.comp.Trauma < 65535 {
						j.comp.Trauma++
					}
					break
				}
			}
		}
	}

	// Remove dead entities
	for _, entity := range s.toRemove {
		world.RemoveEntity(entity)
	}

	// Spawn legend items
	if len(s.itemsToSpawn) > 0 {
		itemID := ecs.ComponentID[components.ItemEntity](world)
		legendID := ecs.ComponentID[components.LegendComponent](world)
		posID := ecs.ComponentID[components.Position](world)

		for _, itemData := range s.itemsToSpawn {
			newEnt := world.NewEntity(itemID, legendID, posID)

			legend := (*components.LegendComponent)(world.Get(newEnt, legendID))
			legend.NameID = itemData.nameID
			legend.Prestige = itemData.prestige
			// History starts empty
			legend.History = make([]uint32, 0)

			pos := (*components.Position)(world.Get(newEnt, posID))
			pos.X = itemData.posX
			pos.Y = itemData.posY
		}
	}
}
