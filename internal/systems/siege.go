package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: The Physical Siege Engine
// Bridges Geography, Combat, and Economics by identifying villages outnumbered by hostile NPCs.
// Spikes local MarketComponent food prices and drains LoyaltyComponent to simulate starvation.

type SiegeSystem struct {
	// Component IDs
	villageID     ecs.ID
	affilID       ecs.ID
	marketID      ecs.ID
	loyaltyID     ecs.ID
	posID         ecs.ID
	capitalID     ecs.ID
	warID         ecs.ID
	npcID         ecs.ID
	siegeMarkerID ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		villageID:     ecs.ComponentID[components.Village](world),
		affilID:       ecs.ComponentID[components.Affiliation](world),
		marketID:      ecs.ComponentID[components.MarketComponent](world),
		loyaltyID:     ecs.ComponentID[components.LoyaltyComponent](world),
		posID:         ecs.ComponentID[components.Position](world),
		capitalID:     ecs.ComponentID[components.CapitalComponent](world),
		warID:         ecs.ComponentID[components.WarTrackerComponent](world),
		npcID:         ecs.ComponentID[components.NPC](world),
		siegeMarkerID: ecs.ComponentID[components.SiegeMarker](world),
	}
}

type npcSiegeData struct {
	countryID uint32
	x         float32
	y         float32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	// 1. Pre-cache active wars from CapitalComponent entities.
	// Map from CountryID -> map[TargetCountryID]bool
	activeWars := make(map[uint32]map[uint32]bool)
	capQuery := world.Query(filter.All(s.capitalID, s.warID, s.affilID))
	for capQuery.Next() {
		affil := (*components.Affiliation)(capQuery.Get(s.affilID))
		war := (*components.WarTrackerComponent)(capQuery.Get(s.warID))
		if war.Active {
			if activeWars[affil.CountryID] == nil {
				activeWars[affil.CountryID] = make(map[uint32]bool)
			}
			activeWars[affil.CountryID][war.TargetCountryID] = true

			// For a true two-way war, we also map the defender to the attacker
			if activeWars[war.TargetCountryID] == nil {
				activeWars[war.TargetCountryID] = make(map[uint32]bool)
			}
			activeWars[war.TargetCountryID][affil.CountryID] = true
		}
	}

	// 2. Pre-cache all NPC positions and affiliations
	var npcs []npcSiegeData
	npcQuery := world.Query(filter.All(s.npcID, s.posID, s.affilID))
	for npcQuery.Next() {
		affil := (*components.Affiliation)(npcQuery.Get(s.affilID))
		pos := (*components.Position)(npcQuery.Get(s.posID))

		if affil.CountryID != 0 {
			npcs = append(npcs, npcSiegeData{
				countryID: affil.CountryID,
				x:         pos.X,
				y:         pos.Y,
			})
		}
	}

	// Slices to queue structural changes
	var addSiege []ecs.Entity
	var removeSiege []ecs.Entity

	// 3. Iterate over Villages
	villQuery := world.Query(filter.All(s.villageID, s.affilID, s.marketID, s.loyaltyID, s.posID))
	for villQuery.Next() {
		affil := (*components.Affiliation)(villQuery.Get(s.affilID))
		pos := (*components.Position)(villQuery.Get(s.posID))
		market := (*components.MarketComponent)(villQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villQuery.Get(s.loyaltyID))

		// Check if village's country is involved in any war
		hostileCount := 0
		friendlyCount := 0

		if enemies, atWar := activeWars[affil.CountryID]; atWar {
			// Count nearby NPCs
			for i := 0; i < len(npcs); i++ {
				dx := npcs[i].x - pos.X
				dy := npcs[i].y - pos.Y
				distSq := dx*dx + dy*dy

				if distSq <= 25.0 {
					if npcs[i].countryID == affil.CountryID {
						friendlyCount++
					} else if enemies[npcs[i].countryID] {
						hostileCount++
					}
				}
			}
		}

		ent := villQuery.Entity()
		hasSiege := villQuery.Has(s.siegeMarkerID)

		if hostileCount > friendlyCount {
			if !hasSiege {
				addSiege = append(addSiege, ent)
			}
			// Apply Siege Effects: Starvation and Disloyalty
			market.FoodPrice += 2.0
			if loyalty.Value > 0 {
				loyalty.Value -= 1
			}
		} else {
			if hasSiege {
				removeSiege = append(removeSiege, ent)
			}
		}
	}

	// Close query manually if we broke early (we didn't, but Arche auto-closes on finish)

	// 4. Apply Structural Changes Outside Query Loop
	for _, e := range addSiege {
		if world.Alive(e) && !world.Has(e, s.siegeMarkerID) {
			world.Add(e, s.siegeMarkerID)
		}
	}

	for _, e := range removeSiege {
		if world.Alive(e) && world.Has(e, s.siegeMarkerID) {
			world.Remove(e, s.siegeMarkerID)
		}
	}
}
