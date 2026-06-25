package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 31 - Flora, Fauna, // Phase 31: Flora, Fauna, & Animal Husbandry Animal Husbandry
// Herders tame wild animals. During local famines (high FoodPrice in their city),
// herders slaughter tamed animals to yield meat, bridging Ecosystem to Economy.

type HusbandrySystem struct {
	herderFilter ecs.Filter
	animalFilter ecs.Filter

	jobID      ecs.ID
	posID      ecs.ID
	needsID    ecs.ID
	identID    ecs.ID
	affID      ecs.ID
	animalID   ecs.ID
	tamedID    ecs.ID
	vitalsID   ecs.ID
	storageID  ecs.ID
	marketID   ecs.ID
	villageID  ecs.ID

	pathQueue *engine.PathRequestQueue
}

func NewHusbandrySystem(world *ecs.World, pathQueue *engine.PathRequestQueue) *HusbandrySystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	animalID := ecs.ComponentID[components.AnimalComponent](world)
	tamedID := ecs.ComponentID[components.TamedMarker](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	villageID := ecs.ComponentID[components.Village](world)

	hMask := ecs.All(jobID, posID, needsID, identID, affID)
	aMask := ecs.All(posID, animalID)

	return &HusbandrySystem{
		herderFilter: &hMask,
		animalFilter: &aMask,
		jobID:        jobID,
		posID:        posID,
		needsID:      needsID,
		identID:      identID,
		affID:        affID,
		animalID:     animalID,
		tamedID:      tamedID,
		vitalsID:     vitalsID,
		storageID:    storageID,
		marketID:     marketID,
		villageID:    villageID,
		pathQueue:    pathQueue,
	}
}

type animalData struct {
	entity ecs.Entity
	x      float32
	y      float32
	isTamed bool
	ownerID uint64
}

func (s *HusbandrySystem) Update(world *ecs.World) {
	// 1. Pre-cache market prices and storages for all cities
	type cityData struct {
		foodPrice float32
		entity    ecs.Entity
	}
	cities := make(map[uint32]cityData)

	marketQuery := world.Query(ecs.All(s.villageID, s.affID, s.marketID, s.storageID))
	for marketQuery.Next() {
		aff := (*components.Affiliation)(marketQuery.Get(s.affID))
		m := (*components.MarketComponent)(marketQuery.Get(s.marketID))
		cities[aff.CityID] = cityData{
			foodPrice: m.FoodPrice,
			entity:    marketQuery.Entity(),
		}
	}

	// 2. Pre-cache animals
	var animals []animalData
	aQuery := world.Query(s.animalFilter)
	for aQuery.Next() {
		pos := (*components.Position)(aQuery.Get(s.posID))
		isTamed := false
		var ownerID uint64
		if aQuery.Has(s.tamedID) {
			isTamed = true
			tamed := (*components.TamedMarker)(aQuery.Get(s.tamedID))
			ownerID = tamed.OwnerID
		}
		animals = append(animals, animalData{
			entity:  aQuery.Entity(),
			x:       pos.X,
			y:       pos.Y,
			isTamed: isTamed,
			ownerID: ownerID,
		})
	}

	if len(animals) == 0 {
		return
	}

	// Structural changes must be deferred
	var toAddTamed []struct { e ecs.Entity; ownerID uint64 }
	var toSlaughter []ecs.Entity
	var toAddFood []struct { cityID uint32; amount uint32 }

	hQuery := world.Query(s.herderFilter)
	for hQuery.Next() {
		job := (*components.JobComponent)(hQuery.Get(s.jobID))
		if job.JobID != components.JobHerder {
			continue
		}

		hPos := (*components.Position)(hQuery.Get(s.posID))
		ident := (*components.Identity)(hQuery.Get(s.identID))
		aff := (*components.Affiliation)(hQuery.Get(s.affID))

		cData, exists := cities[aff.CityID]
		famine := exists && cData.foodPrice > 10.0 // Starvation threshold

		// Find closest animal
		var bestAnimal *animalData
		var bestDistSq float32 = 9999999.0
		bestIndex := -1

		for i := range animals {
			a := &animals[i]
			if !world.Alive(a.entity) {
				continue
			}

			// If famine, we look for our OWN tamed animals to slaughter
			if famine {
				if !a.isTamed || a.ownerID != ident.ID {
					continue
				}
			} else {
				// Otherwise, look for wild animals to tame
				if a.isTamed {
					continue
				}
			}

			dx := hPos.X - a.x
			dy := hPos.Y - a.y
			distSq := dx*dx + dy*dy

			if distSq < bestDistSq {
				bestDistSq = distSq
				bestAnimal = a
				bestIndex = i
			}
		}

		if bestAnimal == nil {
			continue
		}

		// If adjacent, act
		if bestDistSq <= 4.0 {
			if famine {
				// Slaughter
				toSlaughter = append(toSlaughter, bestAnimal.entity)
				anim := (*components.AnimalComponent)(world.Get(bestAnimal.entity, s.animalID))
				toAddFood = append(toAddFood, struct{cityID uint32; amount uint32}{aff.CityID, anim.YieldMeat})
			} else {
				// Tame
				toAddTamed = append(toAddTamed, struct{e ecs.Entity; ownerID uint64}{bestAnimal.entity, ident.ID})
			}

			// Remove from consideration
			if bestIndex != -1 {
				animals[bestIndex] = animals[len(animals)-1]
				animals = animals[:len(animals)-1]
			}
		} else {
			// Pathfind
			if s.pathQueue != nil {
				s.pathQueue.Enqueue(engine.PathRequest{
					EntityID: ident.ID,
					StartX:   hPos.X,
					StartY:   hPos.Y,
					TargetX:  bestAnimal.x,
					TargetY:  bestAnimal.y,
					IsNaval:  false,
				})
			}
		}
	}

	// Apply structural changes
	for _, addT := range toAddTamed {
		if world.Alive(addT.e) && !world.Has(addT.e, s.tamedID) {
			world.Add(addT.e, s.tamedID)
			tm := (*components.TamedMarker)(world.Get(addT.e, s.tamedID))
			tm.OwnerID = addT.ownerID
		}
	}

	for _, sEntity := range toSlaughter {
		if world.Alive(sEntity) {
			world.RemoveEntity(sEntity)
		}
	}

	for _, addF := range toAddFood {
		cData, exists := cities[addF.cityID]
		if exists && world.Alive(cData.entity) {
			store := (*components.StorageComponent)(world.Get(cData.entity, s.storageID))
			store.Food += addF.amount
		}
	}
}
