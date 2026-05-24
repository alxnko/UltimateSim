package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem bridges macro-politics (WarTrackerComponent) with micro-economics (MarketComponent)
// and politics (LoyaltyComponent). It organically spawns a SiegeMarker on a village when it is
// outnumbered by hostile NPCs from a besieging country.

type SiegeSystem struct {
	tickCounter uint64

	// Component IDs
	posID       ecs.ID
	affID       ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	warID       ecs.ID
	villageID   ecs.ID
	siegeID     ecs.ID
	popID       ecs.ID

	// Filters
	villageFilter ecs.Filter
	npcFilter     ecs.Filter
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	s := &SiegeSystem{
		posID:     ecs.ComponentID[components.Position](world),
		affID:     ecs.ComponentID[components.Affiliation](world),
		marketID:  ecs.ComponentID[components.MarketComponent](world),
		loyaltyID: ecs.ComponentID[components.LoyaltyComponent](world),
		warID:     ecs.ComponentID[components.WarTrackerComponent](world),
		villageID: ecs.ComponentID[components.Village](world),
		siegeID:   ecs.ComponentID[components.SiegeMarker](world),
		popID:     ecs.ComponentID[components.PopulationComponent](world),
	}

	s.villageFilter = filter.All(s.posID, s.affID, s.villageID, s.marketID, s.loyaltyID, s.popID)
	f := filter.All(s.posID, s.affID).Without(s.villageID)
	s.npcFilter = &f

	return s
}

type structuralChange struct {
	entity ecs.Entity
	cID    uint32
}

type siegeChange struct {
	entity            ecs.Entity
	besiegerCountryID uint32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// 1. Gather all active wars and the attacking countries.
	// Map of TargetCountryID -> []BesiegerCountryID
	activeWars := make(map[uint32][]uint32)
	warQuery := world.Query(filter.All(s.warID, s.affID))
	for warQuery.Next() {
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warID))
		aff := (*components.Affiliation)(warQuery.Get(s.affID))
		if war.Active {
			activeWars[war.TargetCountryID] = append(activeWars[war.TargetCountryID], aff.CountryID)
		}
	}

	// We cannot simply return if len(activeWars) == 0, because we must clear existing sieges
	// if the wars ended. So we proceed, but hostiles list will be empty.

	// 2. Identify Hostile NPCs (Belonging to a BesiegerCountryID)
	// We'll cache their positions and affiliations to avoid locking the world during the double loop.
	type npcData struct {
		x         float32
		y         float32
		countryID uint32
	}
	var hostiles []npcData
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		pos := (*components.Position)(npcQuery.Get(s.posID))

		// Is this NPC from a besieging country?
		isBesieger := false
		for _, attackers := range activeWars {
			for _, besiegerID := range attackers {
				if aff.CountryID == besiegerID {
					isBesieger = true
					break
				}
			}
			if isBesieger {
				break
			}
		}

		if isBesieger {
			hostiles = append(hostiles, npcData{
				x:         pos.X,
				y:         pos.Y,
				countryID: aff.CountryID,
			})
		}
	}

	// 3. Evaluate Villages for Sieges
	var toAddSiege []siegeChange

	villageQuery := world.Query(s.villageFilter)
	for villageQuery.Next() {
		entity := villageQuery.Entity()
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		pos := (*components.Position)(villageQuery.Get(s.posID))
		pop := (*components.PopulationComponent)(villageQuery.Get(s.popID))

		// Check if this village is a target in any active war
		attackers, isTarget := activeWars[aff.CountryID]

		hostileCount := 0
		primaryAttackerID := uint32(0)

		if isTarget {
			// Count hostile NPCs near this village
			for _, hostile := range hostiles {
				isHostileAttacker := false
				for _, a := range attackers {
					if hostile.countryID == a {
						isHostileAttacker = true
						if primaryAttackerID == 0 {
							primaryAttackerID = a
						}
						break
					}
				}

				if isHostileAttacker {
					dx := pos.X - hostile.x
					dy := pos.Y - hostile.y
					distSq := dx*dx + dy*dy
					if distSq < 25.0 {
						hostileCount++
					}
				}
			}
		}

		// Determine if besieged (e.g. at least 3 hostile NPCs and outnumbers 10% of population)
		// Or simply outnumbered
		if isTarget && hostileCount >= 3 && hostileCount >= int(pop.Count)/10 {
			if !world.Has(entity, s.siegeID) {
				toAddSiege = append(toAddSiege, siegeChange{
					entity:            entity,
					besiegerCountryID: primaryAttackerID,
				})
			}
		} else {
			if world.Has(entity, s.siegeID) {
				// Has siege marker but no longer outnumbered -> remove
				// Defer removal to avoid locked world panic
				toAddSiege = append(toAddSiege, siegeChange{
					entity:            entity,
					besiegerCountryID: 0, // Indicator to remove
				})
			}
		}
	}

	// 4. Apply structural changes (Add SiegeMarkers)
	for _, change := range toAddSiege {
		if change.besiegerCountryID > 0 {
			world.Add(change.entity, s.siegeID)
			marker := (*components.SiegeMarker)(world.Get(change.entity, s.siegeID))
			marker.BesiegerCountryID = change.besiegerCountryID
		} else {
			world.Remove(change.entity, s.siegeID)
		}
	}

	// 5. Process all sieged villages
	s.processExistingSieges(world)
}

func (s *SiegeSystem) processExistingSieges(world *ecs.World) {
	query := world.Query(filter.All(s.villageID, s.siegeID, s.marketID, s.loyaltyID))
	for query.Next() {
		market := (*components.MarketComponent)(query.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(query.Get(s.loyaltyID))

		// Siege drastically inflates food prices due to starvation/hoarding
		market.FoodPrice += 0.5

		// The populace loses faith in their ruler's ability to protect them
		if loyalty.Value > 0 {
			loyalty.Value -= 1
		}
	}
}
