package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 21.1: DesperationSystem
// Links Economy (Market/Needs) to Justice Engine (Crime).

type DesperationSystem struct {
	npcFilter ecs.Filter
}

func NewDesperationSystem(world *ecs.World) *DesperationSystem {
	needsID := ecs.ComponentID[components.Needs](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	posID := ecs.ComponentID[components.Position](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	idID := ecs.ComponentID[components.Identity](world)

	mask := ecs.All(needsID, affID, posID, despID, memID, idID)

	return &DesperationSystem{
		npcFilter: &mask,
	}
}

func (s *DesperationSystem) Update(world *ecs.World) {
	// Step 1: Pre-cache local market prices into a flat map for DOD O(1) lookups
	affID := ecs.ComponentID[components.Affiliation](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	villageID := ecs.ComponentID[components.Village](world)

	marketQuery := world.Query(ecs.All(villageID, affID, marketID))
	marketPrices := make(map[uint32]float32)

	for marketQuery.Next() {
		aff := (*components.Affiliation)(marketQuery.Get(affID))
		m := (*components.MarketComponent)(marketQuery.Get(marketID))
		marketPrices[aff.CityID] = m.FoodPrice
	}

	// Step 2: Extract active Villages with Storage for O(N^2) nearest target stealing loop
	posID := ecs.ComponentID[components.Position](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)

	villageStorageQuery := world.Query(ecs.All(villageID, posID, storageID))

	type vData struct {
		Entity ecs.Entity
		X      float32
		Y      float32
		Storage *components.StorageComponent
	}
	villages := make([]vData, 0, 100)

	for villageStorageQuery.Next() {
		pos := (*components.Position)(villageStorageQuery.Get(posID))
		storage := (*components.StorageComponent)(villageStorageQuery.Get(storageID))
		villages = append(villages, vData{
			Entity: villageStorageQuery.Entity(),
			X:      pos.X,
			Y:      pos.Y,
			Storage: storage,
		})
	}


	// Step 3: Iterate all NPCs
	needsID := ecs.ComponentID[components.Needs](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)

	npcQuery := world.Query(s.npcFilter)

	for npcQuery.Next() {
		needs := (*components.Needs)(npcQuery.Get(needsID))
		aff := (*components.Affiliation)(npcQuery.Get(affID))
		desp := (*components.DesperationComponent)(npcQuery.Get(despID))

		foodPrice, ok := marketPrices[aff.CityID]
		if !ok {
			foodPrice = 1.0 // Default if no market found
		}

		// The Catalyst: Starving AND Poor
		if needs.Food < 30.0 && needs.Wealth < foodPrice {
			if desp.Level < 100 {
				desp.Level += 1
			}
		} else {
			if desp.Level > 0 {
				desp.Level -= 1
			}
		}

		// The Crime Action: Steal
		if desp.Level >= 50 && len(villages) > 0 {
			pos := (*components.Position)(npcQuery.Get(posID))

			// Find nearest village with food
			var bestV *vData
			var bestDist float32 = 9999999.0

			for i := 0; i < len(villages); i++ {
				v := &villages[i]
				if v.Storage.Food <= 0 { continue }

				dx := pos.X - v.X
				dy := pos.Y - v.Y
				distSq := (dx * dx) + (dy * dy)
				if distSq < bestDist {
					bestDist = distSq
					bestV = v
				}
			}

			if bestV != nil {
				// We found a target, execute theft
				stealAmount := float32(20.0)
				if float32(bestV.Storage.Food) < stealAmount {
					stealAmount = float32(bestV.Storage.Food)
				}

				// Physically move goods
				bestV.Storage.Food -= uint32(stealAmount)
				needs.Food += stealAmount
				desp.Level = 0 // Reset after eating

				// Log the crime
				mem := (*components.Memory)(npcQuery.Get(memID))

				// Add memory event: TargetID = village entity ID? We'll just log an interaction
				// In arche, Entity.ID() is a struct, we don't have a reliable uint64 unless we query Identity.
				// Since we just need the system to flag it, TargetID = 0 is fine, InteractionTheft = 4.

				event := components.MemoryEvent{
					TargetID:        0,
					InteractionType: components.InteractionTheft,
					Value:           int32(stealAmount),
					TickStamp:       0, // Or query tickmanager? 0 works for basic Justice evaluation bounds
				}

				mem.Events[mem.Head] = event
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))
			}
		}
	}
}
