package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem evaluates Village entities against nearby NPC positions. If a Village is outnumbered
// by hostile NPCs (based on active WarTrackerComponents), it applies a SiegeMarker.
// This marker naturally drives up local FoodPrice and drains Loyalty.

type siegeNPCData struct {
	entity    ecs.Entity
	x         float32
	y         float32
	countryID uint32
}

type warRelation struct {
	AttackerCountryID uint32
	DefenderCountryID uint32
}

type structuralChange struct {
	entity            ecs.Entity
	addSiege          bool
	removeSiege       bool
	besiegerCountryID uint32
}

type SiegeSystem struct {
	// Cached Filters
	warFilter     ecs.Filter
	npcFilter     ecs.Filter
	villageFilter ecs.Filter

	// Component IDs
	warCompID   ecs.ID
	affID       ecs.ID
	npcID       ecs.ID
	posID       ecs.ID
	villageID   ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	siegeCompID ecs.ID

	// Pre-allocated buffers for DOD and GC elimination
	activeWars  map[warRelation]bool
	npcs        []siegeNPCData
	forceCounts map[uint32]int
	changes     []structuralChange
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	warCompID := ecs.ComponentID[components.WarTrackerComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	npcID := ecs.ComponentID[components.NPC](world)
	posID := ecs.ComponentID[components.Position](world)
	villageID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	siegeCompID := ecs.ComponentID[components.SiegeMarker](world)

	warMask := ecs.All(warCompID, affID)
	npcMask := ecs.All(npcID, posID, affID)
	villageMask := ecs.All(villageID, posID, affID, marketID, loyaltyID)

	return &SiegeSystem{
		warFilter:     &warMask,
		npcFilter:     &npcMask,
		villageFilter: &villageMask,
		warCompID:     warCompID,
		affID:         affID,
		npcID:         npcID,
		posID:         posID,
		villageID:     villageID,
		marketID:      marketID,
		loyaltyID:     loyaltyID,
		siegeCompID:   siegeCompID,
		activeWars:    make(map[warRelation]bool, 100),
		npcs:          make([]siegeNPCData, 0, 500),
		forceCounts:   make(map[uint32]int, 10),
		changes:       make([]structuralChange, 0, 50),
	}
}

func (s *SiegeSystem) Update(world *ecs.World) {
	// Clear pre-allocated structures
	for k := range s.activeWars {
		delete(s.activeWars, k)
	}
	s.npcs = s.npcs[:0]
	s.changes = s.changes[:0]

	// 1. Pre-cache active wars to build a map of hostile relations
	warQuery := world.Query(s.warFilter)
	for warQuery.Next() {
		warComp := (*components.WarTrackerComponent)(warQuery.Get(s.warCompID))
		aff := (*components.Affiliation)(warQuery.Get(s.affID))
		if warComp.Active && aff.CountryID != 0 {
			// Attacker -> Defender
			s.activeWars[warRelation{AttackerCountryID: aff.CountryID, DefenderCountryID: warComp.TargetCountryID}] = true
			// The defender also considers the attacker hostile (symmetric for siege purposes)
			s.activeWars[warRelation{AttackerCountryID: warComp.TargetCountryID, DefenderCountryID: aff.CountryID}] = true
		}
	}

	// 2. Pre-cache all NPCs for spatial evaluation
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))

		// Only NPCs belonging to a country can participate in geopolitical sieges
		if aff.CountryID != 0 {
			s.npcs = append(s.npcs, siegeNPCData{
				entity:    npcQuery.Entity(),
				x:         pos.X,
				y:         pos.Y,
				countryID: aff.CountryID,
			})
		}
	}

	// 3. Evaluate Villages against NPC positions
	villageQuery := world.Query(s.villageFilter)

	for villageQuery.Next() {
		vEntity := villageQuery.Entity()
		vPos := (*components.Position)(villageQuery.Get(s.posID))
		vAff := (*components.Affiliation)(villageQuery.Get(s.affID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))
		hasSiege := villageQuery.Has(s.siegeCompID)

		if vAff.CountryID == 0 {
			continue // Unaligned villages cannot be geopolitically besieged
		}

		// Count forces within siege radius (e.g., distSq <= 100.0)
		for k := range s.forceCounts {
			delete(s.forceCounts, k)
		}

		// Only loop over the slice to maintain determinism
		for _, npc := range s.npcs {
			dx := vPos.X - npc.x
			dy := vPos.Y - npc.y
			distSq := dx*dx + dy*dy

			if distSq <= 100.0 {
				s.forceCounts[npc.countryID]++
			}
		}

		// Determine if besieged - purely deterministic evaluation
		defendingForces := s.forceCounts[vAff.CountryID]
		isBesieged := false
		var dominantBesieger uint32 = 0
		var maxHostileForces int = defendingForces // Only forces greater than defenders win

		// Iterate sequentially over npcs slice to prevent randomized Go map iteration issues
		for _, npc := range s.npcs {
			countryID := npc.countryID
			if countryID == vAff.CountryID {
				continue
			}

			if s.activeWars[warRelation{AttackerCountryID: countryID, DefenderCountryID: vAff.CountryID}] {
				count := s.forceCounts[countryID]

				// Deterministic tie-breaker: largest force wins.
				// If tied, the lower CountryID wins.
				if count > maxHostileForces {
					isBesieged = true
					dominantBesieger = countryID
					maxHostileForces = count
				} else if count == maxHostileForces && countryID < dominantBesieger && isBesieged {
					dominantBesieger = countryID
				}
			}
		}

		// Apply continuous effects or schedule structural changes
		if isBesieged {
			if !hasSiege {
				s.changes = append(s.changes, structuralChange{
					entity:            vEntity,
					addSiege:          true,
					besiegerCountryID: dominantBesieger,
				})
			} else {
				// Organic Macroeconomic & Psychological Impact
				// A siege restricts trade and starves the city, heavily spiking food prices.
				market.FoodPrice *= 1.2

				// Drain loyalty to model breaking morale.
				if loyalty.Value > 0 {
					if loyalty.Value >= 5 {
						loyalty.Value -= 5
					} else {
						loyalty.Value = 0
					}
				}
			}
		} else {
			if hasSiege {
				// Siege broken
				s.changes = append(s.changes, structuralChange{
					entity:      vEntity,
					removeSiege: true,
				})
			}
		}
	}

	// 4. Apply structural changes outside the main query to avoid ECS lock panics
	for _, change := range s.changes {
		if !world.Alive(change.entity) {
			continue
		}
		if change.addSiege {
			if !world.Has(change.entity, s.siegeCompID) {
				world.Add(change.entity, s.siegeCompID)
			}
			marker := (*components.SiegeMarker)(world.Get(change.entity, s.siegeCompID))
			marker.BesiegerCountryID = change.besiegerCountryID
		} else if change.removeSiege {
			if world.Has(change.entity, s.siegeCompID) {
				world.Remove(change.entity, s.siegeCompID)
			}
		}
	}
}
