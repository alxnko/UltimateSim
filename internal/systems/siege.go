package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: The Physical Siege Engine
// This system simulates sieges by comparing hostile vs defending physical NPC presence
// around a Village entity during Geopolitical Resource Wars.
// It mathematically spikes food prices and drains loyalty when outnumbered, triggering economic collapse.

type siegeChange struct {
	entity ecs.Entity
	add    bool
	cID    ecs.ID
	marker components.SiegeMarker
}

type siegeVillageData struct {
	Entity    ecs.Entity
	Position  components.Position
	CountryID uint32
}

type SiegeSystem struct {
	npcFilter     ecs.Filter
	villageFilter ecs.Filter
	capitalFilter ecs.Filter

	posID       ecs.ID
	affilID     ecs.ID
	villageID   ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	siegeID     ecs.ID
	warID       ecs.ID
	countryID   ecs.ID
	capitalID   ecs.ID

	changes     []siegeChange
	villageData []siegeVillageData
}

func (s *SiegeSystem) Initialize(world *ecs.World) {
	s.posID = ecs.ComponentID[components.Position](world)
	s.affilID = ecs.ComponentID[components.Affiliation](world)
	s.villageID = ecs.ComponentID[components.Village](world)
	s.marketID = ecs.ComponentID[components.MarketComponent](world)
	s.loyaltyID = ecs.ComponentID[components.LoyaltyComponent](world)
	s.siegeID = ecs.ComponentID[components.SiegeMarker](world)
	s.warID = ecs.ComponentID[components.WarTrackerComponent](world)
	s.countryID = ecs.ComponentID[components.CountryComponent](world)
	s.capitalID = ecs.ComponentID[components.CapitalComponent](world)

	s.npcFilter = filter.All(ecs.ComponentID[components.NPC](world), s.posID, s.affilID)
	s.villageFilter = filter.All(s.villageID, s.posID, s.affilID, s.marketID, s.loyaltyID)
	s.capitalFilter = filter.All(s.capitalID, s.warID, s.countryID)

	s.changes = make([]siegeChange, 0, 64)
	s.villageData = make([]siegeVillageData, 0, 128)
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.changes = s.changes[:0]
	s.villageData = s.villageData[:0]

	// 1. Find all active wars deterministically
	type activeWar struct {
		attacker uint32
		target   uint32
	}
	var activeWars []activeWar
	capitalQuery := world.Query(s.capitalFilter)
	for capitalQuery.Next() {
		war := (*components.WarTrackerComponent)(capitalQuery.Get(s.warID))
		if war.Active && war.TargetCountryID != 0 {
			if capitalQuery.Has(s.affilID) {
				affil := (*components.Affiliation)(capitalQuery.Get(s.affilID))
				activeWars = append(activeWars, activeWar{attacker: affil.CountryID, target: war.TargetCountryID})
			}
		}
	}

	if len(activeWars) == 0 {
		// If no wars are active, we should still clear existing sieges
		s.clearAllSieges(world)
		return
	}

	// 2. Pre-cache all Villages
	villageQuery := world.Query(s.villageFilter)
	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(s.posID))
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))

		s.villageData = append(s.villageData, siegeVillageData{
			Entity:    villageQuery.Entity(),
			Position:  *pos,
			CountryID: affil.CountryID,
		})
	}

	// 3. For each village, count defenders and hostiles
	for _, vData := range s.villageData {
		var attackers []uint32
		for _, war := range activeWars {
			if vData.CountryID == war.target {
				attackers = append(attackers, war.attacker)
			}
		}

		hasSiegeMarker := world.Has(vData.Entity, s.siegeID)

		if len(attackers) == 0 {
			// Not a target of an active war. Clear siege if it has one.
			if hasSiegeMarker {
				s.changes = append(s.changes, siegeChange{entity: vData.Entity, add: false, cID: s.siegeID})
			}
			continue
		}

		hostiles := 0
		defenders := 0

		// To track which attacker has the most presence
		attackerCounts := make(map[uint32]int)

		npcQuery := world.Query(s.npcFilter)
		for npcQuery.Next() {
			pos := (*components.Position)(npcQuery.Get(s.posID))
			affil := (*components.Affiliation)(npcQuery.Get(s.affilID))

			dx := pos.X - vData.Position.X
			dy := pos.Y - vData.Position.Y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 { // 5 tile radius for siege presence
				isAttacker := false
				for _, attackerID := range attackers {
					if affil.CountryID == attackerID {
						isAttacker = true
						break
					}
				}

				if isAttacker {
					hostiles++
					attackerCounts[affil.CountryID]++
				} else if affil.CountryID == vData.CountryID {
					defenders++
				}
			}
		}

		if hostiles > defenders && hostiles > 0 {
			// Find primary besieger deterministically (largest presence, break ties by lowest CountryID)
			var primaryBesieger uint32 = 0
			var maxCount int = -1

			for _, attackerID := range attackers {
				count := attackerCounts[attackerID]
				if count > maxCount || (count == maxCount && attackerID < primaryBesieger) {
					primaryBesieger = attackerID
					maxCount = count
				}
			}

			// Siege active
			if !hasSiegeMarker {
				s.changes = append(s.changes, siegeChange{
					entity: vData.Entity,
					add:    true,
					cID:    s.siegeID,
					marker: components.SiegeMarker{BesiegerCountryID: primaryBesieger},
				})
			}

			// Apply siege effects: spike food prices, drain loyalty
			market := (*components.MarketComponent)(world.Get(vData.Entity, s.marketID))
			if market != nil {
				market.FoodPrice += 0.5 // Organically spike food price
			}

			loyalty := (*components.LoyaltyComponent)(world.Get(vData.Entity, s.loyaltyID))
			if loyalty != nil {
				if loyalty.Value > 0 {
					loyalty.Value--
				}
			}

		} else {
			// Siege lifted
			if hasSiegeMarker {
				s.changes = append(s.changes, siegeChange{entity: vData.Entity, add: false, cID: s.siegeID})
			}
		}
	}

	// 4. Apply structural changes
	for _, change := range s.changes {
		if change.add {
			world.Add(change.entity, change.cID)
			marker := (*components.SiegeMarker)(world.Get(change.entity, change.cID))
			*marker = change.marker
		} else {
			world.Remove(change.entity, change.cID)
		}
	}
}

func (s *SiegeSystem) clearAllSieges(world *ecs.World) {
	query := world.Query(filter.All(s.siegeID))
	for query.Next() {
		s.changes = append(s.changes, siegeChange{entity: query.Entity(), add: false, cID: s.siegeID})
	}

	for _, change := range s.changes {
		world.Remove(change.entity, change.cID)
	}
}
