package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
	"sort"
)

// Phase 31: Animal Husbandry Engine (HusbandrySystem)
// HusbandrySystem bridges Ecosystem to Economy by having JobHerder NPCs tame
// AnimalComponent entities. During local famines (assessed via MarketComponent),
// herders slaughter tamed animals for meat, transferring it to their employer's storage.

type HusbandrySystem struct {
	world *ecs.World

	// Component IDs
	npcID       ecs.ID
	jobID       ecs.ID
	animalID    ecs.ID
	tamedID     ecs.ID
	businessID  ecs.ID
	villageID   ecs.ID
	marketID    ecs.ID
	storageID   ecs.ID
	idID        ecs.ID

	// active employers lookup
	employers map[uint64]employerData
}

type employerData struct {
	Entity ecs.Entity
	Market *components.MarketComponent
	Storage *components.StorageComponent
}

// NewHusbandrySystem creates a new HusbandrySystem.
func NewHusbandrySystem(world *ecs.World) *HusbandrySystem {
	return &HusbandrySystem{
		world:      world,
		npcID:      ecs.ComponentID[components.NPC](world),
		jobID:      ecs.ComponentID[components.JobComponent](world),
		animalID:   ecs.ComponentID[components.AnimalComponent](world),
		tamedID:    ecs.ComponentID[components.TamedMarker](world),
		businessID: ecs.ComponentID[components.BusinessComponent](world),
		villageID:  ecs.ComponentID[components.Village](world),
		marketID:   ecs.ComponentID[components.MarketComponent](world),
		storageID:  ecs.ComponentID[components.StorageComponent](world),
		idID:       ecs.ComponentID[components.Identity](world),
		employers:  make(map[uint64]employerData),
	}
}

// Update executes the husbandry logic per tick.
func (s *HusbandrySystem) Update(w *ecs.World) {
	// 1. Pre-cache active employers (Villages and Businesses)
	clear(s.employers)

	// Villages
	vq := s.world.Query(filter.All(s.villageID, s.marketID, s.storageID, s.idID))
	for vq.Next() {
		id := (*components.Identity)(vq.Get(s.idID))
		market := (*components.MarketComponent)(vq.Get(s.marketID))
		storage := (*components.StorageComponent)(vq.Get(s.storageID))
		s.employers[id.ID] = employerData{
			Entity:  vq.Entity(),
			Market:  market,
			Storage: storage,
		}
	}

	// Businesses
	bq := s.world.Query(filter.All(s.businessID, s.marketID, s.storageID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		market := (*components.MarketComponent)(bq.Get(s.marketID))
		storage := (*components.StorageComponent)(bq.Get(s.storageID))
		s.employers[id.ID] = employerData{
			Entity:  bq.Entity(),
			Market:  market,
			Storage: storage,
		}
	}

	// 2. Iterate Herders to find out what employer IDs are active for herding
	herderEmployers := make(map[uint64]bool)
	hq := s.world.Query(filter.All(s.npcID, s.jobID))
	for hq.Next() {
		job := (*components.JobComponent)(hq.Get(s.jobID))
		if job.JobID == components.JobHerder {
			herderEmployers[job.EmployerID] = true
		}
	}

	// 3. Process Taming & Slaughtering
	// We need to defer structural changes (adding TamedMarker, removing Animal/Entity)
	type tameTask struct {
		Animal ecs.Entity
		Owner  uint64
	}
	var animalsToTame []tameTask
	var animalsToSlaughter []ecs.Entity

	aq := s.world.Query(filter.All(s.animalID))
	for aq.Next() {
		animalEnt := aq.Entity()
		animal := (*components.AnimalComponent)(aq.Get(s.animalID))

		if aq.Has(s.tamedID) {
			// Already tamed, check if slaughter is needed
			tamed := (*components.TamedMarker)(aq.Get(s.tamedID))

			// Is there an active herder for this owner?
			if !herderEmployers[tamed.OwnerID] {
				continue
			}

			ownerData, exists := s.employers[tamed.OwnerID]
			if !exists {
				continue
			}

			// Local famine check: food price > 10.0
			if ownerData.Market.FoodPrice > 10.0 {
				animalsToSlaughter = append(animalsToSlaughter, animalEnt)
				ownerData.Storage.Food += animal.YieldMeat
			}
		} else {
			// Not tamed, can be tamed if there's any active herder.
			if len(herderEmployers) > 0 {
				// Convert to slice and sort for determinism
				empIDs := make([]uint64, 0, len(herderEmployers))
				for empID := range herderEmployers {
					empIDs = append(empIDs, empID)
				}

				sort.Slice(empIDs, func(i, j int) bool {
					return empIDs[i] < empIDs[j]
				})

				// Assign to the lowest ID employer
				animalsToTame = append(animalsToTame, tameTask{Animal: animalEnt, Owner: empIDs[0]})
			}
		}
	}

	// 4. Apply deferred structural changes
	for _, task := range animalsToTame {
		if s.world.Alive(task.Animal) && !s.world.Has(task.Animal, s.tamedID) {
			s.world.Add(task.Animal, s.tamedID)
			tamed := (*components.TamedMarker)(s.world.Get(task.Animal, s.tamedID))
			tamed.OwnerID = task.Owner
		}
	}

	for _, e := range animalsToSlaughter {
		if s.world.Alive(e) {
			s.world.RemoveEntity(e)
		}
	}
}
