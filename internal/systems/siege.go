package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem evaluates active wars and physically maps outnumbering hostile NPCs to Villages.
// If a Village is outnumbered, it applies a SiegeMarker, drastically spiking local food prices
// and draining loyalty to simulate blockade starvation without needing explicit combat.

type SiegeSystem struct {
	tickCounter uint64

	// Cached Component IDs
	capID       ecs.ID
	warID       ecs.ID
	villageID   ecs.ID
	posID       ecs.ID
	affID       ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	siegeID     ecs.ID
	npcID       ecs.ID
	identID     ecs.ID
}

// IsExpensive returns true as it does N^2 spatial evaluations.
func (s *SiegeSystem) IsExpensive() bool {
	return true
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		capID:     ecs.ComponentID[components.CapitalComponent](world),
		warID:     ecs.ComponentID[components.WarTrackerComponent](world),
		villageID: ecs.ComponentID[components.Village](world),
		posID:     ecs.ComponentID[components.Position](world),
		affID:     ecs.ComponentID[components.Affiliation](world),
		marketID:  ecs.ComponentID[components.MarketComponent](world),
		loyaltyID: ecs.ComponentID[components.LoyaltyComponent](world),
		siegeID:   ecs.ComponentID[components.SiegeMarker](world),
		npcID:     ecs.ComponentID[components.NPC](world),
		identID:   ecs.ComponentID[components.Identity](world),
	}
}

type siegeVillageData struct {
	entity    ecs.Entity
	x         float32
	y         float32
	countryID uint32
	market    *components.MarketComponent
	loyalty   *components.LoyaltyComponent
	isSieged  bool
}

type siegeNPCData struct {
	x         float32
	y         float32
	countryID uint32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run every 50 ticks to simulate the slow pace of a siege
	if s.tickCounter%50 != 0 {
		return
	}

	// 1. Identify active wars
	capQuery := world.Query(ecs.All(s.capID, s.warID, s.affID))
	activeWars := make(map[uint32]uint32) // map[TargetCountryID]AttackerCountryID
	for capQuery.Next() {
		war := (*components.WarTrackerComponent)(capQuery.Get(s.warID))
		aff := (*components.Affiliation)(capQuery.Get(s.affID))
		if war.Active && war.TargetCountryID != 0 {
			activeWars[war.TargetCountryID] = aff.CountryID
		}
	}

	// Always need to evaluate sieges, because even if no active wars, we might need to clear existing sieges.
	// But let's check if there are existing sieges first.

	// 2. Pre-cache all NPCs mapped to CountryID to avoid nested queries
	npcQuery := world.Query(ecs.All(s.npcID, s.posID, s.affID))
	var npcs []siegeNPCData
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		if aff.CountryID != 0 {
			npcs = append(npcs, siegeNPCData{
				x:         pos.X,
				y:         pos.Y,
				countryID: aff.CountryID,
			})
		}
	}

	// 3. Pre-cache Villages
	villageQuery := world.Query(ecs.All(s.villageID, s.posID, s.affID, s.marketID, s.loyaltyID))
	var villages []siegeVillageData
	for villageQuery.Next() {
		ent := villageQuery.Entity()
		pos := (*components.Position)(villageQuery.Get(s.posID))
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))
		isSieged := world.Has(ent, s.siegeID)

		villages = append(villages, siegeVillageData{
			entity:    ent,
			x:         pos.X,
			y:         pos.Y,
			countryID: aff.CountryID,
			market:    market,
			loyalty:   loyalty,
			isSieged:  isSieged,
		})
	}

	// Lists for structural ECS updates
	type siegeAddData struct {
		entity     ecs.Entity
		besiegerID uint32
	}
	var toAddSiege []siegeAddData
	var toRemoveSiege []ecs.Entity

	// 4. Evaluate Siege Conditions
	for _, v := range villages {
		besiegerID, isTarget := activeWars[v.countryID]

		hostiles := 0
		defenders := 0

		if isTarget {
			// Count NPCs within siege radius (e.g. 5 tiles -> 25 distSq)
			for _, npc := range npcs {
				dx := v.x - npc.x
				dy := v.y - npc.y
				distSq := dx*dx + dy*dy

				if distSq <= 25.0 {
					if npc.countryID == besiegerID {
						hostiles++
					} else if npc.countryID == v.countryID {
						defenders++
					}
				}
			}
		}

		// A village is besieged if there is an active war against its country,
		// and hostile troops outnumber defenders by at least 1 in the area.
		shouldBeSieged := isTarget && hostiles > defenders

		if shouldBeSieged {
			if !v.isSieged {
				toAddSiege = append(toAddSiege, siegeAddData{
					entity:     v.entity,
					besiegerID: besiegerID,
				})
			}

			// Apply Siege Effects: Starvation and Unrest
			v.market.FoodPrice += 5.0 // Drastically spike food prices
			if v.market.FoodPrice > 100.0 {
				v.market.FoodPrice = 100.0
			}

			if v.loyalty.Value >= 2 {
				v.loyalty.Value -= 2 // Drain loyalty
			} else {
				v.loyalty.Value = 0
			}
		} else {
			// If it shouldn't be sieged anymore but it currently is, remove it
			if v.isSieged {
				toRemoveSiege = append(toRemoveSiege, v.entity)
			}
		}
	}

	// 5. Apply Structural ECS Changes
	for _, data := range toAddSiege {
		if world.Alive(data.entity) && !world.Has(data.entity, s.siegeID) {
			world.Add(data.entity, s.siegeID)
			siege := (*components.SiegeMarker)(world.Get(data.entity, s.siegeID))
			siege.BesiegerCountryID = data.besiegerID
		}
	}

	for _, ent := range toRemoveSiege {
		if world.Alive(ent) && world.Has(ent, s.siegeID) {
			world.Remove(ent, s.siegeID)
		}
	}
}
