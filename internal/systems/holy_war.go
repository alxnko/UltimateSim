package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 20.1: Ideological Warfare (HolyWarSystem)
// Holy Wars generate aggressive physical Entity spawns targeting rival storage clusters.

type HolyWarSystem struct {
	tickCounter uint64

	// Component IDs
	identID       ecs.ID
	posID         ecs.ID
	velID         ecs.ID
	villageID     ecs.ID
	beliefID      ecs.ID
	storageID     ecs.ID
	crusaderID    ecs.ID
	crusadeCompID ecs.ID
}

func NewHolyWarSystem(world *ecs.World) *HolyWarSystem {
	return &HolyWarSystem{
		identID:       ecs.ComponentID[components.Identity](world),
		posID:         ecs.ComponentID[components.Position](world),
		velID:         ecs.ComponentID[components.Velocity](world),
		villageID:     ecs.ComponentID[components.Village](world),
		beliefID:      ecs.ComponentID[components.BeliefComponent](world),
		storageID:     ecs.ComponentID[components.StorageComponent](world),
		crusaderID:    ecs.ComponentID[components.CrusaderEntity](world),
		crusadeCompID: ecs.ComponentID[components.CrusadeComponent](world),
	}
}

func (s *HolyWarSystem) Update(world *ecs.World) {
	s.tickCounter++

	// 1. Process active CrusaderEntity units
	var toRemove []ecs.Entity

	// We only want Crusader entities that are active on the board
	crusaderFilter := ecs.All(s.crusaderID, s.crusadeCompID, s.posID, s.velID)
	crusaderQuery := world.Query(&crusaderFilter)

	// Cache values + entity handles, not component pointers — GC corruption class, see banditry.go.
	// Velocity writes re-fetch through the entity handle at use time.
	type crusaderData struct {
		entity       ecs.Entity
		x            float32
		y            float32
		targetCityID uint32
	}

	var activeCrusaders []crusaderData
	for crusaderQuery.Next() {
		pos := (*components.Position)(crusaderQuery.Get(s.posID))
		crusade := (*components.CrusadeComponent)(crusaderQuery.Get(s.crusadeCompID))
		activeCrusaders = append(activeCrusaders, crusaderData{
			entity:       crusaderQuery.Entity(),
			x:            pos.X,
			y:            pos.Y,
			targetCityID: crusade.TargetCityID,
		})
	}

	// Calculate target city pos
	villageFilter := ecs.All(s.villageID, s.identID, s.posID, s.storageID)
	villageQuery := world.Query(&villageFilter)

	// Cache values + entity handles, not component pointers — GC corruption class, see banditry.go.
	// Storage writes re-fetch through the entity handle at use time.
	type villageData struct {
		entity ecs.Entity
		id     uint64
		x      float32
		y      float32
	}
	var villages []villageData
	for villageQuery.Next() {
		ident := (*components.Identity)(villageQuery.Get(s.identID))
		pos := (*components.Position)(villageQuery.Get(s.posID))
		villages = append(villages, villageData{
			entity: villageQuery.Entity(),
			id:     ident.ID,
			x:      pos.X,
			y:      pos.Y,
		})
	}

	// Resolve active crusader movement and combat
	for _, crusader := range activeCrusaders {
		var targetVillage *villageData
		for vi := range villages {
			if uint32(villages[vi].id) == crusader.targetCityID {
				targetVillage = &villages[vi]
				break
			}
		}

		if targetVillage == nil {
			// Target city destroyed or no longer exists, despawn crusader
			toRemove = append(toRemove, crusader.entity)
			continue
		}

		// Move towards target
		dx := targetVillage.x - crusader.x
		dy := targetVillage.y - crusader.y
		distSq := dx*dx + dy*dy

		// Reached the city
		if distSq < 4.0 {
			// Deal damage to storage (re-fetched at use time via the entity handle)
			if world.Alive(targetVillage.entity) {
				storage := (*components.StorageComponent)(world.Get(targetVillage.entity, s.storageID))
				if storage.Food > 50 {
					storage.Food -= 50
				} else {
					storage.Food = 0
				}
				if storage.Wood > 50 {
					storage.Wood -= 50
				} else {
					storage.Wood = 0
				}
			}
			// Despawn crusader
			toRemove = append(toRemove, crusader.entity)
		} else if world.Alive(crusader.entity) {
			// Pure DOD math movement (velocity re-fetched at use time via the entity handle)
			vel := (*components.Velocity)(world.Get(crusader.entity, s.velID))
			speed := float32(0.5)
			if dx > 0 {
				vel.X = speed
			} else if dx < 0 {
				vel.X = -speed
			} else {
				vel.X = 0
			}

			if dy > 0 {
				vel.Y = speed
			} else if dy < 0 {
				vel.Y = -speed
			} else {
				vel.Y = 0
			}
		}
	}

	// Remove processed Crusaders safely outside the loop
	for _, e := range toRemove {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}

	// 2. Spawn logic runs every 1000 ticks
	if s.tickCounter%1000 != 0 {
		return
	}

	// Iterate through all villages to check for opposing beliefs
	cityFilter := ecs.All(s.villageID, s.identID, s.posID, s.beliefID)
	cityQuery := world.Query(&cityFilter)

	// Cache values, not component pointers — GC corruption class, see banditry.go.
	// The beliefs slice header points at Go-heap backing memory, not archetype storage.
	type cityBeliefData struct {
		entity  ecs.Entity
		id      uint64
		x       float32
		y       float32
		beliefs []components.Belief
	}

	var cities []cityBeliefData
	for cityQuery.Next() {
		ident := (*components.Identity)(cityQuery.Get(s.identID))
		pos := (*components.Position)(cityQuery.Get(s.posID))
		belief := (*components.BeliefComponent)(cityQuery.Get(s.beliefID))
		cities = append(cities, cityBeliefData{
			entity:  cityQuery.Entity(),
			id:      ident.ID,
			x:       pos.X,
			y:       pos.Y,
			beliefs: belief.Beliefs,
		})
	}

	// Check cities against each other
	var spawnCrusaders []struct {
		StartX       float32
		StartY       float32
		TargetCityID uint32
	}

	for i := 0; i < len(cities); i++ {
		cityA := cities[i]
		if len(cityA.beliefs) == 0 {
			continue
		}

		// Find cityA's dominant belief
		var dominantBeliefA uint32
		var maxWeightA int32 = -1
		for _, b := range cityA.beliefs {
			if b.Weight > maxWeightA {
				maxWeightA = b.Weight
				dominantBeliefA = b.BeliefID
			}
		}

		if dominantBeliefA == 0 {
			continue
		}

		for j := 0; j < len(cities); j++ {
			if i == j {
				continue
			}
			cityB := cities[j]
			if len(cityB.beliefs) == 0 {
				continue
			}

			// Find cityB's dominant belief
			var dominantBeliefB uint32
			var maxWeightB int32 = -1
			for _, b := range cityB.beliefs {
				if b.Weight > maxWeightB {
					maxWeightB = b.Weight
					dominantBeliefB = b.BeliefID
				}
			}

			// If beliefs differ and they are within range, start Holy War
			dx := cityA.x - cityB.x
			dy := cityA.y - cityB.y
			distSq := dx*dx + dy*dy

			// Within holy war range (radius 100.0)
			if distSq < 10000.0 && dominantBeliefA != dominantBeliefB {
				// cityA sends crusaders to attack cityB
				spawnCrusaders = append(spawnCrusaders, struct {
					StartX       float32
					StartY       float32
					TargetCityID uint32
				}{
					StartX:       cityA.x,
					StartY:       cityA.y,
					TargetCityID: uint32(cityB.id),
				})
			}
		}
	}

	// Spawn Crusaders outside loop
	for _, spawn := range spawnCrusaders {
		e := world.NewEntity(s.crusaderID, s.crusadeCompID, s.posID, s.velID)
		pos := (*components.Position)(world.Get(e, s.posID))
		crusade := (*components.CrusadeComponent)(world.Get(e, s.crusadeCompID))

		pos.X = spawn.StartX
		pos.Y = spawn.StartY
		crusade.TargetCityID = spawn.TargetCityID
	}
}
