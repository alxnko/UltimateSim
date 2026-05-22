package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by identifying villages
// overwhelmed by hostile NPCs during an active war. It applies a SiegeMarker,
// which dynamically spikes local MarketComponent food prices and drains LoyaltyComponent.

type SiegeSystem struct {
	tickCounter uint64

	// Component IDs
	posID       ecs.ID
	affID       ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	villageID   ecs.ID
	siegeID     ecs.ID
	warID       ecs.ID
	capID       ecs.ID
	npcID       ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		posID:     ecs.ComponentID[components.Position](world),
		affID:     ecs.ComponentID[components.Affiliation](world),
		marketID:  ecs.ComponentID[components.MarketComponent](world),
		loyaltyID: ecs.ComponentID[components.LoyaltyComponent](world),
		villageID: ecs.ComponentID[components.Village](world),
		siegeID:   ecs.ComponentID[components.SiegeMarker](world),
		warID:     ecs.ComponentID[components.WarTrackerComponent](world),
		capID:     ecs.ComponentID[components.CapitalComponent](world),
		npcID:     ecs.ComponentID[components.NPC](world),
	}
}

type siegeNpcData struct {
	x         float32
	y         float32
	countryID uint32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%100 != 0 {
		return
	}

	// 1. Identify active wars to know which countries are hostile
	// Mapping: attackerCountryID -> targetCountryID
	activeWars := make(map[uint32]uint32)
	warQuery := world.Query(ecs.All(s.capID, s.affID, s.warID))
	for warQuery.Next() {
		aff := (*components.Affiliation)(warQuery.Get(s.affID))
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warID))
		if war.Active && war.TargetCountryID != 0 {
			activeWars[aff.CountryID] = war.TargetCountryID
		}
	}

	if len(activeWars) == 0 {
		// No active wars, remove all SiegeMarkers
		siegeQuery := world.Query(ecs.All(s.villageID, s.siegeID))
		var toRemove []ecs.Entity
		for siegeQuery.Next() {
			toRemove = append(toRemove, siegeQuery.Entity())
		}
		for _, e := range toRemove {
			world.Remove(e, s.siegeID)
		}
		return
	}

	// 2. Pre-cache all NPC positions and affiliations for O(1) DOD scanning
	npcQuery := world.Query(ecs.All(s.npcID, s.posID, s.affID))
	var npcs []siegeNpcData
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		npcs = append(npcs, siegeNpcData{
			x:         pos.X,
			y:         pos.Y,
			countryID: aff.CountryID,
		})
	}

	// 3. Evaluate each village
	villageQuery := world.Query(ecs.All(s.villageID, s.posID, s.affID, s.marketID, s.loyaltyID))
	var markersToAdd []struct{ entity ecs.Entity; besiegerID uint32 }
	var markersToRemove []ecs.Entity

	for villageQuery.Next() {
		vPos := (*components.Position)(villageQuery.Get(s.posID))
		vAff := (*components.Affiliation)(villageQuery.Get(s.affID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))

		defenderCount := 0

		// To track hostiles per attacking country
		hostilesByCountry := make(map[uint32]int)

		for i := 0; i < len(npcs); i++ {
			npc := npcs[i]

			// Spatial check (radius squared <= 25.0, e.g. 5 tiles)
			dx := vPos.X - npc.x
			dy := vPos.Y - npc.y
			distSq := dx*dx + dy*dy
			if distSq > 25.0 {
				continue
			}

			if npc.countryID == vAff.CountryID {
				defenderCount++
			} else {
				// Check if this NPC's country is at war with the village's country
				if activeWars[npc.countryID] == vAff.CountryID {
					hostilesByCountry[npc.countryID]++
				}
			}
		}

		// Determine if the village is under siege by any specific country
		isBesieged := false
		var dominantBesieger uint32

		for besiegerID, hostileCount := range hostilesByCountry {
			if hostileCount > defenderCount {
				isBesieged = true
				dominantBesieger = besiegerID
				break // Just pick the first overwhelming force
			}
		}

		hasMarker := world.Has(villageQuery.Entity(), s.siegeID)

		if isBesieged {
			if !hasMarker {
				markersToAdd = append(markersToAdd, struct{ entity ecs.Entity; besiegerID uint32 }{villageQuery.Entity(), dominantBesieger})
			} else {
				// If already has marker, maybe update besiegerID, but mainly apply consequences
				market.FoodPrice += 5.0
				if market.FoodPrice > 100.0 {
					market.FoodPrice = 100.0
				}
				if loyalty.Value > 0 {
					loyalty.Value -= 1
				}
			}
		} else {
			if hasMarker {
				markersToRemove = append(markersToRemove, villageQuery.Entity())
			}
		}
	}

	// Apply structural changes outside the main query loop
	for _, e := range markersToRemove {
		if world.Alive(e) && world.Has(e, s.siegeID) {
			world.Remove(e, s.siegeID)
		}
	}

	for _, add := range markersToAdd {
		if world.Alive(add.entity) && !world.Has(add.entity, s.siegeID) {
			world.Add(add.entity, s.siegeID)
			marker := (*components.SiegeMarker)(world.Get(add.entity, s.siegeID))
			marker.BesiegerCountryID = add.besiegerID

			// Initial spike on being placed
			market := (*components.MarketComponent)(world.Get(add.entity, s.marketID))
			loyalty := (*components.LoyaltyComponent)(world.Get(add.entity, s.loyaltyID))

			market.FoodPrice += 10.0
			if market.FoodPrice > 100.0 {
				market.FoodPrice = 100.0
			}
			if loyalty.Value > 0 {
				loyalty.Value -= 5
			}
		}
	}
}
