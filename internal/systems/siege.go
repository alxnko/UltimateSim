package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by applying a SiegeMarker
// to Villages outnumbered by hostile NPCs during wars. It organically spikes local
// MarketComponent.FoodPrice and drains LoyaltyComponent to simulate starvation and panic.

type SiegeSystem struct {
	tickCounter uint64

	warFilter ecs.Filter
	vilFilter ecs.Filter
	npcFilter ecs.Filter

	capID    ecs.ID
	warID    ecs.ID
	vilID    ecs.ID
	posID    ecs.ID
	affID    ecs.ID
	loyID    ecs.ID
	marID    ecs.ID
	npcID    ecs.ID
	siegeID  ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	capID := ecs.ComponentID[components.CapitalComponent](world)
	warID := ecs.ComponentID[components.WarTrackerComponent](world)
	vilID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	loyID := ecs.ComponentID[components.LoyaltyComponent](world)
	marID := ecs.ComponentID[components.MarketComponent](world)
	npcID := ecs.ComponentID[components.NPC](world)
	siegeID := ecs.ComponentID[components.SiegeMarker](world)

	warMask := ecs.All(capID, warID, affID)
	vilMask := ecs.All(vilID, posID, affID, loyID, marID)
	npcMask := ecs.All(npcID, posID, affID)

	return &SiegeSystem{
		tickCounter: 0,
		warFilter:   &warMask,
		vilFilter:   &vilMask,
		npcFilter:   &npcMask,
		capID:       capID,
		warID:       warID,
		vilID:       vilID,
		posID:       posID,
		affID:       affID,
		loyID:       loyID,
		marID:       marID,
		npcID:       npcID,
		siegeID:     siegeID,
	}
}

type siegeVilData struct {
	entity ecs.Entity
	x      float32
	y      float32
	cID    uint32
}

type siegeNPCData struct {
	x   float32
	y   float32
	cID uint32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process logic periodically
	if s.tickCounter%10 != 0 {
		return
	}

	// 1. Gather active wars: AttackerCountryID -> TargetCountryID
	// Use map for deterministic tie-breaker check or just active wars
	// Because of potential multiple wars, let's track Attacker -> Target relationships
	// For DOD performance, cache TargetCountryID -> []AttackerCountryIDs
	activeWars := make(map[uint32][]uint32)
	wQuery := world.Query(s.warFilter)
	for wQuery.Next() {
		war := (*components.WarTrackerComponent)(wQuery.Get(s.warID))
		if !war.Active {
			continue
		}
		aff := (*components.Affiliation)(wQuery.Get(s.affID))
		attackerCID := aff.CountryID
		targetCID := war.TargetCountryID
		activeWars[targetCID] = append(activeWars[targetCID], attackerCID)
	}

	if len(activeWars) == 0 {
		return // No active wars
	}

	// 2. Pre-cache all Villages and their positions
	var villages []siegeVilData
	vQuery := world.Query(s.vilFilter)
	for vQuery.Next() {
		aff := (*components.Affiliation)(vQuery.Get(s.affID))
		// Only consider villages belonging to a country involved as a target
		if _, exists := activeWars[aff.CountryID]; exists {
			pos := (*components.Position)(vQuery.Get(s.posID))
			villages = append(villages, siegeVilData{
				entity: vQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				cID:    aff.CountryID,
			})
		}
	}

	// 3. Pre-cache all NPCs and their positions
	var npcs []siegeNPCData
	nQuery := world.Query(s.npcFilter)
	for nQuery.Next() {
		aff := (*components.Affiliation)(nQuery.Get(s.affID))
		pos := (*components.Position)(nQuery.Get(s.posID))
		npcs = append(npcs, siegeNPCData{
			x:   pos.X,
			y:   pos.Y,
			cID: aff.CountryID,
		})
	}

	type entityAction struct {
		entity ecs.Entity
		cID    uint32
	}

	markersToAdd := make([]entityAction, 0)
	markersToRemove := make([]ecs.Entity, 0)
	surrenders := make([]entityAction, 0)

	// 4. Evaluate each village for siege conditions
	for _, vData := range villages {
		targetCID := vData.cID
		attackers := activeWars[targetCID]

		// Tally attackers and defenders within distance
		defenders := 0
		attackerCounts := make(map[uint32]int) // AttackerCountryID -> count

		for _, nData := range npcs {
			dx := vData.x - nData.x
			dy := vData.y - nData.y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 {
				if nData.cID == targetCID {
					defenders++
				} else {
					// Check if this NPC belongs to an attacking country
					for _, aCID := range attackers {
						if nData.cID == aCID {
							attackerCounts[aCID]++
							break
						}
					}
				}
			}
		}

		// Find the dominant attacker
		dominantAttacker := uint32(0)
		maxCount := 0
		for aCID, count := range attackerCounts {
			if count > maxCount {
				maxCount = count
				dominantAttacker = aCID
			} else if count == maxCount && aCID > dominantAttacker {
				// Deterministic tie-breaker
				dominantAttacker = aCID
			}
		}

		hasSiege := world.Has(vData.entity, s.siegeID)

		if maxCount > defenders {
			// Outnumbered by attackers
			if !hasSiege {
				markersToAdd = append(markersToAdd, entityAction{entity: vData.entity, cID: dominantAttacker})
			} else {
				// Already under siege, apply effects
				loyalty := (*components.LoyaltyComponent)(world.Get(vData.entity, s.loyID))
				market := (*components.MarketComponent)(world.Get(vData.entity, s.marID))
				siege := (*components.SiegeMarker)(world.Get(vData.entity, s.siegeID))

				// Spike food prices
				market.FoodPrice *= 1.5
				if market.FoodPrice > 500.0 {
					market.FoodPrice = 500.0
				}

				// Drain loyalty
				drain := uint32(10)
				if siege.Intensity > 0 {
					drain += siege.Intensity
				}
				if loyalty.Value > drain {
					loyalty.Value -= drain
				} else {
					loyalty.Value = 0
					// Village surrenders to the besieger
					surrenders = append(surrenders, entityAction{entity: vData.entity, cID: siege.BesiegerCountryID})
					markersToRemove = append(markersToRemove, vData.entity)
				}

				siege.Intensity += 1
			}
		} else {
			// Siege lifted or failed
			if hasSiege {
				markersToRemove = append(markersToRemove, vData.entity)
			}
		}
	}

	// 5. Apply structural changes
	for _, action := range markersToAdd {
		if world.Alive(action.entity) && !world.Has(action.entity, s.siegeID) {
			world.Add(action.entity, s.siegeID)
			sm := (*components.SiegeMarker)(world.Get(action.entity, s.siegeID))
			sm.BesiegerCountryID = action.cID
			sm.Intensity = 1
		}
	}

	for _, e := range markersToRemove {
		if world.Alive(e) && world.Has(e, s.siegeID) {
			world.Remove(e, s.siegeID)
		}
	}

	for _, action := range surrenders {
		if world.Alive(action.entity) {
			aff := (*components.Affiliation)(world.Get(action.entity, s.affID))
			aff.CountryID = action.cID
		}
	}
}
