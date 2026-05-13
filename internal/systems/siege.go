package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by applying a SiegeMarker
// to Villages outnumbered by hostile NPCs during wars, organically spiking local
// MarketComponent food prices and draining LoyaltyComponent to simulate starvation.

type SiegeSystem struct {
	tickCounter uint64

	// Cache components
	villageID   ecs.ID
	posID       ecs.ID
	affID       ecs.ID
	warTrackerID ecs.ID
	npcID       ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	siegeID     ecs.ID

	// Slices for DOD structural changes
	toSiege   []siegeData
	toUnsiege []ecs.Entity
}

type siegeData struct {
	entity            ecs.Entity
	besiegerCountryID uint32
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		villageID:    ecs.ComponentID[components.Village](world),
		posID:        ecs.ComponentID[components.Position](world),
		affID:        ecs.ComponentID[components.Affiliation](world),
		warTrackerID: ecs.ComponentID[components.WarTrackerComponent](world),
		npcID:        ecs.ComponentID[components.NPC](world),
		marketID:     ecs.ComponentID[components.MarketComponent](world),
		loyaltyID:    ecs.ComponentID[components.LoyaltyComponent](world),
		siegeID:      ecs.ComponentID[components.SiegeMarker](world),
		toSiege:      make([]siegeData, 0, 50),
		toUnsiege:    make([]ecs.Entity, 0, 50),
	}
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%100 != 0 {
		return // Evaluate sieges every 100 ticks
	}

	s.toSiege = s.toSiege[:0]
	s.toUnsiege = s.toUnsiege[:0]

	// 1. Identify active wars and the defending countries
	activeWars := make(map[uint32]uint32) // targetCountryID -> attackerCountryID
	capQuery := world.Query(ecs.All(s.warTrackerID, s.affID))
	for capQuery.Next() {
		war := (*components.WarTrackerComponent)(capQuery.Get(s.warTrackerID))
		aff := (*components.Affiliation)(capQuery.Get(s.affID))
		if war.Active {
			activeWars[war.TargetCountryID] = aff.CountryID
		}
	}

	if len(activeWars) == 0 {
		// Remove all sieges if there are no wars
		siegeQuery := world.Query(ecs.All(s.siegeID))
		for siegeQuery.Next() {
			s.toUnsiege = append(s.toUnsiege, siegeQuery.Entity())
		}
		s.applyStructuralChanges(world)
		return
	}

	// 2. Pre-cache all NPC positions and affiliations for O(1) checks
	type npcCacheData struct {
		x         float32
		y         float32
		countryID uint32
	}
	npcs := make([]npcCacheData, 0, 1000)
	npcQuery := world.Query(ecs.All(s.npcID, s.posID, s.affID))
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		npcs = append(npcs, npcCacheData{
			x:         pos.X,
			y:         pos.Y,
			countryID: aff.CountryID,
		})
	}

	// 3. Evaluate Villages for Sieges
	villageQuery := world.Query(ecs.All(s.villageID, s.posID, s.affID, s.marketID, s.loyaltyID))
	for villageQuery.Next() {
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		pos := (*components.Position)(villageQuery.Get(s.posID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))

		vEnt := villageQuery.Entity()
		hasSiege := world.Has(vEnt, s.siegeID)

		attackerCountryID, isAtWar := activeWars[aff.CountryID]

		if !isAtWar {
			if hasSiege {
				s.toUnsiege = append(s.toUnsiege, vEnt)
			}
			continue
		}

		// Count defenders and attackers within a squared radius of 25.0 (5 tiles)
		defenders := 0
		attackers := 0

		for i := 0; i < len(npcs); i++ {
			n := &npcs[i]
			dx := pos.X - n.x
			dy := pos.Y - n.y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 {
				if n.countryID == aff.CountryID {
					defenders++
				} else if n.countryID == attackerCountryID {
					attackers++
				}
			}
		}

		// A siege occurs if attackers outnumber defenders
		if attackers > defenders && attackers > 0 {
			if !hasSiege {
				s.toSiege = append(s.toSiege, siegeData{
					entity:            vEnt,
					besiegerCountryID: attackerCountryID,
				})
			}
			// Apply economic and loyalty penalties (The Butterfly Effect)
			market.FoodPrice *= 1.5 // Skyrocket food prices
			if loyalty.Value > 5 {
				loyalty.Value -= 5 // Drain loyalty
			} else {
				loyalty.Value = 0
			}
		} else {
			if hasSiege {
				s.toUnsiege = append(s.toUnsiege, vEnt)
			}
		}
	}

	s.applyStructuralChanges(world)
}

func (s *SiegeSystem) applyStructuralChanges(world *ecs.World) {
	for _, e := range s.toUnsiege {
		if world.Alive(e) && world.Has(e, s.siegeID) {
			world.Remove(e, s.siegeID)
		}
	}

	for _, data := range s.toSiege {
		if world.Alive(data.entity) && !world.Has(data.entity, s.siegeID) {
			world.Add(data.entity, s.siegeID)
			marker := (*components.SiegeMarker)(world.Get(data.entity, s.siegeID))
			marker.BesiegerCountryID = data.besiegerCountryID
		}
	}
}
