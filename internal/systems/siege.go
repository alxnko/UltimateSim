package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by applying an 8-byte SiegeMarker
// to Villages outnumbered by hostile NPCs during wars, organically spiking local
// MarketComponent food prices and draining LoyaltyComponent to simulate starvation.

type SiegeSystem struct {
	// Component IDs
	posID       ecs.ID
	affID       ecs.ID
	npcID       ecs.ID
	villageID   ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	warID       ecs.ID
	siegeID     ecs.ID
	identID     ecs.ID

	// Filters
	warFilter     ecs.Filter
	npcFilter     ecs.Filter
	villageFilter ecs.Filter
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	npcID := ecs.ComponentID[components.NPC](world)
	villageID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	warID := ecs.ComponentID[components.WarTrackerComponent](world)
	siegeID := ecs.ComponentID[components.SiegeMarker](world)
	identID := ecs.ComponentID[components.Identity](world)

	warMask := ecs.All(warID, affID)
	npcMask := ecs.All(npcID, posID, affID)
	villageMask := ecs.All(villageID, posID, affID, marketID, loyaltyID)

	return &SiegeSystem{
		posID:         posID,
		affID:         affID,
		npcID:         npcID,
		villageID:     villageID,
		marketID:      marketID,
		loyaltyID:     loyaltyID,
		warID:         warID,
		siegeID:       siegeID,
		identID:       identID,
		warFilter:     &warMask,
		npcFilter:     &npcMask,
		villageFilter: &villageMask,
	}
}

type npcSiegeData struct {
	x         float32
	y         float32
	countryID uint32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	// 1. Gather active wars to know who is hostile to whom
	// Map from target country ID -> slice of attacking country IDs
	activeWars := make(map[uint32][]uint32)
	warQuery := world.Query(s.warFilter)
	for warQuery.Next() {
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warID))
		if !war.Active {
			continue
		}
		aff := (*components.Affiliation)(warQuery.Get(s.affID))
		attackerID := aff.CountryID
		targetID := war.TargetCountryID

		activeWars[targetID] = append(activeWars[targetID], attackerID)
	}

	if len(activeWars) == 0 {
		return // No active wars, skip
	}

	// 2. Pre-cache all NPC positions and affiliations into a flat slice for cache-friendly DOD iteration
	var npcs []npcSiegeData
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		npcs = append(npcs, npcSiegeData{
			x:         pos.X,
			y:         pos.Y,
			countryID: aff.CountryID,
		})
	}

	// Structural modifications queue
	type siegeChange struct {
		entity      ecs.Entity
		add         bool
		besiegerID  uint32
	}
	var changes []siegeChange

	// 3. Evaluate all Village entities
	villageQuery := world.Query(s.villageFilter)
	for villageQuery.Next() {
		villagePos := (*components.Position)(villageQuery.Get(s.posID))
		villageAff := (*components.Affiliation)(villageQuery.Get(s.affID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))
		hasSiege := villageQuery.Has(s.siegeID)

		villageCountryID := villageAff.CountryID
		attackers, isTarget := activeWars[villageCountryID]
		if !isTarget && !hasSiege {
			continue // Not a target of any war and no active siege
		}

		// Count defenders and hostiles within siege radius (distSq <= 25.0)
		defenders := 0
		hostiles := 0
		var mainBesieger uint32

		for _, npc := range npcs {
			dx := npc.x - villagePos.X
			dy := npc.y - villagePos.Y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 {
				if npc.countryID == villageCountryID {
					defenders++
				} else {
					// Check if NPC's country is attacking this village's country
					for _, attacker := range attackers {
						if npc.countryID == attacker {
							hostiles++
							mainBesieger = attacker
							break
						}
					}
				}
			}
		}

		// Evaluate siege state
		if hostiles > defenders && hostiles > 0 {
			if !hasSiege {
				changes = append(changes, siegeChange{
					entity:     villageQuery.Entity(),
					add:        true,
					besiegerID: mainBesieger,
				})
			}

			// Apply effects of siege
			// Food prices spike organically to simulate starvation
			market.FoodPrice += 10.0

			// Loyalty drains to simulate political collapse under starvation
			if loyalty.Value >= 5 {
				loyalty.Value -= 5
			} else {
				loyalty.Value = 0
			}

		} else {
			if hasSiege {
				changes = append(changes, siegeChange{
					entity: villageQuery.Entity(),
					add:    false,
				})
			}
		}
	}

	// 4. Apply structural changes deterministically
	for _, change := range changes {
		if !world.Alive(change.entity) {
			continue
		}
		if change.add {
			world.Add(change.entity, s.siegeID)
			// Re-fetch component pointer due to structural modification hazard
			siege := (*components.SiegeMarker)(world.Get(change.entity, s.siegeID))
			siege.BesiegerCountryID = change.besiegerID
		} else {
			world.Remove(change.entity, s.siegeID)
		}
	}
}
