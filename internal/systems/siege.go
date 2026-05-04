package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem evaluates geographic presence of hostile NPCs relative to allied NPCs
// around a village. If hostiles outnumber allies, a SiegeMarker is applied,
// organically spiking food prices (starvation) and dropping loyalty.

type SiegeSystem struct {
	// Filter and Component IDs
	villageFilter ecs.Filter
	npcFilter     ecs.Filter

	villageID    ecs.ID
	posID        ecs.ID
	affilID      ecs.ID
	marketID     ecs.ID
	loyaltyID    ecs.ID
	siegeID      ecs.ID
	npcID        ecs.ID
	warTrackerID ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	villageID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	siegeID := ecs.ComponentID[components.SiegeMarker](world)
	npcID := ecs.ComponentID[components.NPC](world)
	warTrackerID := ecs.ComponentID[components.WarTrackerComponent](world)

	vMask := ecs.All(villageID, posID, affilID, marketID, loyaltyID)
	nMask := ecs.All(npcID, posID, affilID)

	return &SiegeSystem{
		villageFilter: &vMask,
		npcFilter:     &nMask,
		villageID:     villageID,
		posID:         posID,
		affilID:       affilID,
		marketID:      marketID,
		loyaltyID:     loyaltyID,
		siegeID:       siegeID,
		npcID:         npcID,
		warTrackerID:  warTrackerID,
	}
}

type cachedNPC struct {
	x         float32
	y         float32
	countryID uint32
}

type cachedVillage struct {
	entity    ecs.Entity
	x         float32
	y         float32
	countryID uint32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	// 1. Build map of hostile countries (Attacker -> Defender and Defender -> Attacker)
	hostilities := make(map[uint32]map[uint32]bool)

	// Query for active WarTrackerComponents (usually on capitals/countries)
	// We need a custom filter for this specific query since it's just Affiliation + WarTracker
	warFilter := filter.All(s.affilID, s.warTrackerID)
	warQuery := world.Query(warFilter)
	for warQuery.Next() {
		affil := (*components.Affiliation)(warQuery.Get(s.affilID))
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warTrackerID))

		if war.Active {
			if hostilities[affil.CountryID] == nil {
				hostilities[affil.CountryID] = make(map[uint32]bool)
			}
			hostilities[affil.CountryID][war.TargetCountryID] = true

			// Wars are mutual for this purpose
			if hostilities[war.TargetCountryID] == nil {
				hostilities[war.TargetCountryID] = make(map[uint32]bool)
			}
			hostilities[war.TargetCountryID][affil.CountryID] = true
		}
	}
	// Query naturally exhausted, no Close() needed

	// 2. Cache all NPCs for spatial comparison
	var npcs []cachedNPC
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		affil := (*components.Affiliation)(npcQuery.Get(s.affilID))
		npcs = append(npcs, cachedNPC{
			x:         pos.X,
			y:         pos.Y,
			countryID: affil.CountryID,
		})
	}
	// No break used, auto-closes.

	// 3. Cache villages to avoid structural modification during iteration
	var villages []cachedVillage
	villageQuery := world.Query(s.villageFilter)
	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(s.posID))
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))

		villages = append(villages, cachedVillage{
			entity:    villageQuery.Entity(),
			x:         pos.X,
			y:         pos.Y,
			countryID: affil.CountryID,
		})
	}

	// 4. Evaluate Siege logic
	for _, v := range villages {
		hostileCount := 0
		alliedCount := 0

		// Check spatial presence of NPCs around the village
		for _, n := range npcs {
			dx := v.x - n.x
			dy := v.y - n.y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 {
				if n.countryID == v.countryID {
					alliedCount++
				} else if hostilities[n.countryID] != nil && hostilities[n.countryID][v.countryID] {
					hostileCount++
				}
			}
		}

		// Apply or Remove SiegeMarker based on presence
		isSieged := hostileCount > alliedCount && hostileCount > 0
		hasMarker := world.Has(v.entity, s.siegeID)

		if isSieged && !hasMarker {
			// Start Siege
			world.Add(v.entity, s.siegeID)
		} else if !isSieged && hasMarker {
			// End Siege
			world.Remove(v.entity, s.siegeID)
		}

		// Emergent Effects for active sieges
		if isSieged {
			// Fetch fresh pointers to avoid structural invalidation from ECS swap-and-pop
			market := (*components.MarketComponent)(world.Get(v.entity, s.marketID))
			loyalty := (*components.LoyaltyComponent)(world.Get(v.entity, s.loyaltyID))

			// Starvation economics: Food price spikes dramatically
			if market.FoodPrice < 500.0 {
				market.FoodPrice *= 1.2 // 20% spike per tick under siege
			}

			// Social collapse: Loyalty drains
			if loyalty.Value > 0 {
				loyalty.Value -= 1
			}
		}
	}
}
