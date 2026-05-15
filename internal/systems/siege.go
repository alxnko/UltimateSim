package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by applying a SiegeMarker
// to Villages outnumbered by hostile NPCs during wars, organically spiking local
// MarketComponent food prices and draining LoyaltyComponent to simulate starvation.

type SiegeSystem struct {
	warCompID ecs.ID
	affID     ecs.ID
	posID     ecs.ID
	villageID ecs.ID
	marketID  ecs.ID
	loyaltyID ecs.ID
	siegeID   ecs.ID
	npcID     ecs.ID
	vitalsID  ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		warCompID: ecs.ComponentID[components.WarTrackerComponent](world),
		affID:     ecs.ComponentID[components.Affiliation](world),
		posID:     ecs.ComponentID[components.Position](world),
		villageID: ecs.ComponentID[components.Village](world),
		marketID:  ecs.ComponentID[components.MarketComponent](world),
		loyaltyID: ecs.ComponentID[components.LoyaltyComponent](world),
		siegeID:   ecs.ComponentID[components.SiegeMarker](world),
		npcID:     ecs.ComponentID[components.NPC](world),
		vitalsID:  ecs.ComponentID[components.VitalsComponent](world),
	}
}

type warData struct {
	AttackerCountryID uint32
	DefenderCountryID uint32
}

type siegeNpcData struct {
	CountryID uint32
	X         float32
	Y         float32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	// 1. Gather all active wars
	var activeWars []warData
	warQuery := world.Query(filter.All(s.warCompID, s.affID))
	for warQuery.Next() {
		warTracker := (*components.WarTrackerComponent)(warQuery.Get(s.warCompID))
		if warTracker.Active {
			aff := (*components.Affiliation)(warQuery.Get(s.affID))
			activeWars = append(activeWars, warData{
				AttackerCountryID: aff.CountryID,
				DefenderCountryID: warTracker.TargetCountryID,
			})
		}
	}

	if len(activeWars) == 0 {
		// Even if there are no wars, we might need to process existing sieges
		// Wait, if a war ends, sieges from that war should probably be lifted.
		// For simplicity, we just evaluate all villages with sieges below.
	}

	// 2. Pre-cache all living NPCs with a country affiliation
	var npcs []siegeNpcData
	npcQuery := world.Query(filter.All(s.npcID, s.affID, s.posID, s.vitalsID))
	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		if aff.CountryID != 0 {
			pos := (*components.Position)(npcQuery.Get(s.posID))
			vitals := (*components.VitalsComponent)(npcQuery.Get(s.vitalsID))
			if vitals.Blood > 0 { // Ensure NPC is alive
				npcs = append(npcs, siegeNpcData{
					CountryID: aff.CountryID,
					X:         pos.X,
					Y:         pos.Y,
				})
			}
		}
	}

	// 3. Evaluate Villages
	// We gather villages that have an Affiliation, Position, Market, and Loyalty.
	villageQuery := world.Query(filter.All(s.villageID, s.affID, s.posID, s.marketID, s.loyaltyID))

	type structuralChange struct {
		entity   ecs.Entity
		action   int // 0 = remove siege, 1 = add siege
		besieger uint32
	}
	var changes []structuralChange

	for villageQuery.Next() {
		entity := villageQuery.Entity()
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		pos := (*components.Position)(villageQuery.Get(s.posID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))

		// Check if this village is involved in any active war as a defender
		isAtWar := false
		var hostileCountryID uint32
		for _, war := range activeWars {
			if war.DefenderCountryID == aff.CountryID {
				isAtWar = true
				hostileCountryID = war.AttackerCountryID
				// Note: if attacked by multiple countries, we just take the first one for simplicity,
				// or we could accumulate all hostiles. We'll check all hostiles in the NPC loop.
				break
			}
			if war.AttackerCountryID == aff.CountryID {
				isAtWar = true
				hostileCountryID = war.DefenderCountryID
				break
			}
		}

		hasSiege := villageQuery.Has(s.siegeID)

		if !isAtWar {
			if hasSiege {
				changes = append(changes, structuralChange{entity: entity, action: 0})
			}
			continue
		}

		// Count hostile vs friendly NPCs within a radius (e.g., 25.0 distSq = 5 tiles)
		friendlyCount := 0
		hostileCount := 0

		for _, npc := range npcs {
			dx := npc.X - pos.X
			dy := npc.Y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 {
				if npc.CountryID == aff.CountryID {
					friendlyCount++
				} else {
					// Check if this NPC's country is at war with the village's country
					for _, war := range activeWars {
						if (war.AttackerCountryID == npc.CountryID && war.DefenderCountryID == aff.CountryID) ||
							(war.DefenderCountryID == npc.CountryID && war.AttackerCountryID == aff.CountryID) {
							hostileCount++
							hostileCountryID = npc.CountryID
							break
						}
					}
				}
			}
		}

		// Siege logic: if hostile NPCs outnumber defenders
		if hostileCount > friendlyCount && hostileCount > 0 {
			if !hasSiege {
				changes = append(changes, structuralChange{entity: entity, action: 1, besieger: hostileCountryID})
			} else {
				// Already has siege, apply continuous effects
				market.FoodPrice += 10.0 // Starvation spike
				if loyalty.Value > 0 {
					loyalty.Value -= 1 // Loyalty drain
				}
			}
		} else {
			if hasSiege {
				changes = append(changes, structuralChange{entity: entity, action: 0})
			}
		}
	}

	// 4. Apply structural changes
	for _, change := range changes {
		if world.Alive(change.entity) {
			if change.action == 1 {
				world.Add(change.entity, s.siegeID)
				siege := (*components.SiegeMarker)(world.Get(change.entity, s.siegeID))
				siege.BesiegerCountryID = change.besieger

				// Apply initial effects
				if world.Has(change.entity, s.marketID) {
					market := (*components.MarketComponent)(world.Get(change.entity, s.marketID))
					market.FoodPrice += 10.0
				}
				if world.Has(change.entity, s.loyaltyID) {
					loyalty := (*components.LoyaltyComponent)(world.Get(change.entity, s.loyaltyID))
					if loyalty.Value > 0 {
						loyalty.Value -= 1
					}
				}

			} else if change.action == 0 {
				world.Remove(change.entity, s.siegeID)
			}
		}
	}
}
