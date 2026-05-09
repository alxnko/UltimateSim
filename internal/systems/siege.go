package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine
// The SiegeSystem bridges Geography, Combat, and Economics.
// It applies a SiegeMarker to Villages outnumbered by hostile NPCs during wars,
// organically spiking local MarketComponent food prices and draining LoyaltyComponent to simulate starvation.

type siegeAddOp struct {
	entity            ecs.Entity
	besiegerCountryID uint32
}

type npcSiegeData struct {
	countryID uint32
	x         float32
	y         float32
}

type SiegeSystem struct {
	tickCounter uint64

	// Component IDs
	capID     ecs.ID
	warID     ecs.ID
	affID     ecs.ID
	posID     ecs.ID
	villID    ecs.ID
	marketID  ecs.ID
	loyaltyID ecs.ID
	npcID     ecs.ID
	siegeID   ecs.ID
	vitalsID  ecs.ID

	// Filters
	warFilter     ecs.Filter
	villageFilter ecs.Filter
	npcFilter     ecs.Filter
	siegeFilter   ecs.Filter
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	capID := ecs.ComponentID[components.CapitalComponent](world)
	warID := ecs.ComponentID[components.WarTrackerComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	posID := ecs.ComponentID[components.Position](world)
	villID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)
	siegeID := ecs.ComponentID[components.SiegeMarker](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)

	wMask := ecs.All(capID, warID, affID)
	vMask := ecs.All(villID, posID, affID, marketID, loyaltyID)
	nMask := ecs.All(npcID, posID, affID, vitalsID)
	sMask := ecs.All(villID, posID, affID, marketID, loyaltyID, siegeID)

	return &SiegeSystem{
		capID:         capID,
		warID:         warID,
		affID:         affID,
		posID:         posID,
		villID:        villID,
		marketID:      marketID,
		loyaltyID:     loyaltyID,
		npcID:         npcID,
		siegeID:       siegeID,
		vitalsID:      vitalsID,
		warFilter:     &wMask,
		villageFilter: &vMask,
		npcFilter:     &nMask,
		siegeFilter:   &sMask,
	}
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// 1. Map Target Country -> Attacker Country (Only active wars)
	activeWars := make(map[uint32][]uint32)
	wQuery := world.Query(s.warFilter)
	for wQuery.Next() {
		war := (*components.WarTrackerComponent)(wQuery.Get(s.warID))
		if !war.Active {
			continue
		}
		aff := (*components.Affiliation)(wQuery.Get(s.affID))
		activeWars[war.TargetCountryID] = append(activeWars[war.TargetCountryID], aff.CountryID)
	}

	// 2. Cache all NPCs positions and affiliations to avoid O(N^2) Arche lookups inside inner loop
	var npcs []npcSiegeData
	nQuery := world.Query(s.npcFilter)
	for nQuery.Next() {
		pos := (*components.Position)(nQuery.Get(s.posID))
		aff := (*components.Affiliation)(nQuery.Get(s.affID))
		npcs = append(npcs, npcSiegeData{
			countryID: aff.CountryID,
			x:         pos.X,
			y:         pos.Y,
		})
	}

	var toAdd []siegeAddOp
	var toRemove []ecs.Entity

	// 3. Evaluate Un-besieged Villages for possible siege
	vQuery := world.Query(s.villageFilter)
	for vQuery.Next() {
		entity := vQuery.Entity()
		if world.Has(entity, s.siegeID) {
			continue // Handled below
		}

		aff := (*components.Affiliation)(vQuery.Get(s.affID))
		attackers, hasWar := activeWars[aff.CountryID]
		if !hasWar {
			continue
		}

		pos := (*components.Position)(vQuery.Get(s.posID))

		defendersCount := 0
		maxAttackersCount := 0
		primaryAttackerID := uint32(0)

		attackerCounts := make(map[uint32]int)

		for _, n := range npcs {
			dx := n.x - pos.X
			dy := n.y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq <= 100.0 {
				if n.countryID == aff.CountryID {
					defendersCount++
				} else {
					for _, attackerID := range attackers {
						if n.countryID == attackerID {
							attackerCounts[attackerID]++
							if attackerCounts[attackerID] > maxAttackersCount {
								maxAttackersCount = attackerCounts[attackerID]
								primaryAttackerID = attackerID
							}
							break
						}
					}
				}
			}
		}

		if maxAttackersCount > 0 && maxAttackersCount > defendersCount {
			toAdd = append(toAdd, siegeAddOp{
				entity:            entity,
				besiegerCountryID: primaryAttackerID,
			})
		}
	}

	// 4. Apply emergent effects to currently Besieged Villages
	sQuery := world.Query(s.siegeFilter)
	for sQuery.Next() {
		entity := sQuery.Entity()
		aff := (*components.Affiliation)(sQuery.Get(s.affID))
		pos := (*components.Position)(sQuery.Get(s.posID))
		market := (*components.MarketComponent)(sQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(sQuery.Get(s.loyaltyID))
		siege := (*components.SiegeMarker)(sQuery.Get(s.siegeID))

		// Check if siege still active (attackers > defenders)
		defendersCount := 0
		attackersCount := 0

		for _, n := range npcs {
			dx := n.x - pos.X
			dy := n.y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq <= 100.0 {
				if n.countryID == aff.CountryID {
					defendersCount++
				} else if n.countryID == siege.BesiegerCountryID {
					attackersCount++
				}
			}
		}

		if attackersCount <= defendersCount {
			toRemove = append(toRemove, entity)
		} else {
			// Emergent starvation and loyalty drain
			market.FoodPrice += 1.5
			if loyalty.Value > 0 {
				loyalty.Value--
			}
		}
	}

	// 5. Apply Structural Changes deterministically (swap-and-pop safe)
	for _, op := range toAdd {
		if !world.Has(op.entity, s.siegeID) { // double check
			world.Add(op.entity, s.siegeID)
			marker := (*components.SiegeMarker)(world.Get(op.entity, s.siegeID))
			marker.BesiegerCountryID = op.besiegerCountryID
		}
	}

	for _, e := range toRemove {
		if world.Has(e, s.siegeID) {
			world.Remove(e, s.siegeID)
		}
	}
}
