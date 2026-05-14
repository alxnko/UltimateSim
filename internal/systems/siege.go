package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem evaluates active geopolitical wars (Phase 29). It checks if an attacking
// Country's NPCs physically surround a defending Village in numbers greater than the
// Village's Population. If so, it places a SiegeMarker on the Village.
// The marker organically spikes local MarketComponent food prices and drains LoyaltyComponent.

type activeWarData struct {
	AttackerCountryID uint32
	DefenderCountryID uint32
}

type siegeNPCData struct {
	Entity    ecs.Entity
	CountryID uint32
	X         float32
	Y         float32
}

type structuralSiegeChange struct {
	villageEntity ecs.Entity
	besiegerID    uint32
}

type SiegeSystem struct {
	tickCounter uint64

	activeWars []activeWarData
	npcs       []siegeNPCData
	changes    []structuralSiegeChange

	// Component IDs
	capID       ecs.ID
	warCompID   ecs.ID
	affilID     ecs.ID
	posID       ecs.ID
	npcID       ecs.ID
	villageID   ecs.ID
	popID       ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	siegeID     ecs.ID
}

// NewSiegeSystem creates a new SiegeSystem.
func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		tickCounter: 0,
		activeWars:  make([]activeWarData, 0, 10),
		npcs:        make([]siegeNPCData, 0, 100),
		changes:     make([]structuralSiegeChange, 0, 10),

		capID:       ecs.ComponentID[components.CapitalComponent](world),
		warCompID:   ecs.ComponentID[components.WarTrackerComponent](world),
		affilID:     ecs.ComponentID[components.Affiliation](world),
		posID:       ecs.ComponentID[components.Position](world),
		npcID:       ecs.ComponentID[components.NPC](world),
		villageID:   ecs.ComponentID[components.Village](world),
		popID:       ecs.ComponentID[components.PopulationComponent](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
		loyaltyID:   ecs.ComponentID[components.LoyaltyComponent](world),
		siegeID:     ecs.ComponentID[components.SiegeMarker](world),
	}
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *SiegeSystem) IsExpensive() bool {
	return true
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run every 10 ticks to spread CPU load
	if s.tickCounter%10 != 0 {
		return
	}

	s.activeWars = s.activeWars[:0]
	s.npcs = s.npcs[:0]
	s.changes = s.changes[:0]

	// 1. Gather all active wars
	warFilter := filter.All(s.capID, s.warCompID, s.affilID)
	warQuery := world.Query(warFilter)
	for warQuery.Next() {
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warCompID))
		if !war.Active {
			continue
		}
		affil := (*components.Affiliation)(warQuery.Get(s.affilID))
		s.activeWars = append(s.activeWars, activeWarData{
			AttackerCountryID: affil.CountryID,
			DefenderCountryID: war.TargetCountryID,
		})
	}

	if len(s.activeWars) == 0 {
		// Clean up any lingering SiegeMarkers since no wars are active
		siegeFilter := filter.All(s.villageID, s.siegeID)
		sQuery := world.Query(siegeFilter)
		var toRemove []ecs.Entity
		for sQuery.Next() {
			toRemove = append(toRemove, sQuery.Entity())
		}
		for _, e := range toRemove {
			world.Remove(e, s.siegeID)
		}
		return
	}

	// 2. Pre-cache all NPCs from countries involved in active wars to prevent nested ecs.Query
	npcFilter := filter.All(s.npcID, s.posID, s.affilID)
	npcQuery := world.Query(npcFilter)
	for npcQuery.Next() {
		affil := (*components.Affiliation)(npcQuery.Get(s.affilID))
		if affil.CountryID == 0 {
			continue
		}

		// Check if NPC's country is involved in an active war (either attacker or defender)
		involved := false
		for i := 0; i < len(s.activeWars); i++ {
			if s.activeWars[i].AttackerCountryID == affil.CountryID || s.activeWars[i].DefenderCountryID == affil.CountryID {
				involved = true
				break
			}
		}

		if involved {
			pos := (*components.Position)(npcQuery.Get(s.posID))
			s.npcs = append(s.npcs, siegeNPCData{
				Entity:    npcQuery.Entity(),
				CountryID: affil.CountryID,
				X:         pos.X,
				Y:         pos.Y,
			})
		}
	}

	// 3. Evaluate Villages for Sieges
	villageFilter := filter.All(s.villageID, s.posID, s.popID, s.affilID, s.marketID, s.loyaltyID)
	villageQuery := world.Query(villageFilter)

	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))
		if affil.CountryID == 0 {
			continue
		}

		var relevantWar *activeWarData
		for i := 0; i < len(s.activeWars); i++ {
			if s.activeWars[i].DefenderCountryID == affil.CountryID {
				relevantWar = &s.activeWars[i]
				break
			}
		}

		if relevantWar == nil {
			// This village's country is not being attacked.
			// Remove SiegeMarker if it exists.
			if world.Has(villageQuery.Entity(), s.siegeID) {
				s.changes = append(s.changes, structuralSiegeChange{
					villageEntity: villageQuery.Entity(),
					besiegerID:    0, // 0 signifies removal
				})
			}
			continue
		}

		pos := (*components.Position)(villageQuery.Get(s.posID))
		pop := (*components.PopulationComponent)(villageQuery.Get(s.popID))

		// Count hostile NPCs near the village (radius squared <= 25.0, e.g., 5 tiles)
		hostileCount := 0
		for i := 0; i < len(s.npcs); i++ {
			npc := s.npcs[i]
			if npc.CountryID == relevantWar.AttackerCountryID {
				dx := pos.X - npc.X
				dy := pos.Y - npc.Y
				distSq := dx*dx + dy*dy
				if distSq <= 25.0 {
					hostileCount++
				}
			}
		}

		// If outnumbering the local population, apply/maintain siege
		if uint32(hostileCount) > pop.Count {
			if !world.Has(villageQuery.Entity(), s.siegeID) {
				// Queue adding the marker
				s.changes = append(s.changes, structuralSiegeChange{
					villageEntity: villageQuery.Entity(),
					besiegerID:    relevantWar.AttackerCountryID,
				})
			} else {
				// The marker is already here, meaning the siege is actively progressing.
				// Process the physical reality of the siege:
				market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
				loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))
				siegeMarker := (*components.SiegeMarker)(villageQuery.Get(s.siegeID))

				// Update the besieger in case it changed
				siegeMarker.BesiegerCountryID = relevantWar.AttackerCountryID

				// 1. Tangible Starvation (Spike Food Prices)
				market.FoodPrice += 5.0

				// 2. Political Collapse (Drain Loyalty)
				if loyalty.Value >= 1 {
					loyalty.Value -= 1
				}
			}
		} else {
			// Siege broken or not enough troops. Remove marker if exists.
			if world.Has(villageQuery.Entity(), s.siegeID) {
				s.changes = append(s.changes, structuralSiegeChange{
					villageEntity: villageQuery.Entity(),
					besiegerID:    0,
				})
			}
		}
	}

	// 4. Apply structural changes
	for i := 0; i < len(s.changes); i++ {
		change := s.changes[i]
		if change.besiegerID == 0 {
			world.Remove(change.villageEntity, s.siegeID)
		} else {
			world.Add(change.villageEntity, s.siegeID)
			marker := (*components.SiegeMarker)(world.Get(change.villageEntity, s.siegeID))
			marker.BesiegerCountryID = change.besiegerID
		}
	}
}
