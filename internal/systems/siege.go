package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics.
// It maps active wars, evaluates spatial placement of hostile NPCs around Villages,
// and if attackers physically outnumber defenders, applies SiegeMarker.
// When active, it spikes MarketComponent food prices and drains LoyaltyComponent.

type SiegeSystem struct {
	tickCounter uint64

	// Component IDs
	capID       ecs.ID
	warCompID   ecs.ID
	posID       ecs.ID
	npcID       ecs.ID
	affID       ecs.ID
	villageID   ecs.ID
	siegeID     ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	jobID       ecs.ID
	vitalsID    ecs.ID
}

// NewSiegeSystem creates a new SiegeSystem.
func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		tickCounter: 0,
		capID:       ecs.ComponentID[components.CapitalComponent](world),
		warCompID:   ecs.ComponentID[components.WarTrackerComponent](world),
		posID:       ecs.ComponentID[components.Position](world),
		npcID:       ecs.ComponentID[components.NPC](world),
		affID:       ecs.ComponentID[components.Affiliation](world),
		villageID:   ecs.ComponentID[components.Village](world),
		siegeID:     ecs.ComponentID[components.SiegeMarker](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
		loyaltyID:   ecs.ComponentID[components.LoyaltyComponent](world),
		jobID:       ecs.ComponentID[components.JobComponent](world),
		vitalsID:    ecs.ComponentID[components.VitalsComponent](world),
	}
}

// Update evaluates military positioning to apply or execute sieges.
func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%100 != 0 {
		return
	}

	// 1. Gather all active wars
	type warData struct {
		attackerCountryID uint32
		defenderCountryID uint32
	}
	var activeWars []warData

	capFilter := ecs.All(s.capID, s.affID, s.warCompID)
	capQuery := world.Query(capFilter)
	for capQuery.Next() {
		aff := (*components.Affiliation)(capQuery.Get(s.affID))
		war := (*components.WarTrackerComponent)(capQuery.Get(s.warCompID))
		if war.Active {
			activeWars = append(activeWars, warData{
				attackerCountryID: aff.CountryID,
				defenderCountryID: war.TargetCountryID,
			})
		}
	}

	if len(activeWars) == 0 {
		return // No wars, no sieges
	}

	// 2. Cache all NPCs for rapid spatial evaluation
	type npcNode struct {
		x         float32
		y         float32
		countryID uint32
		isGuard   bool
	}
	var npcs []npcNode
	npcFilter := ecs.All(s.npcID, s.posID, s.affID, s.vitalsID)
	npcQuery := world.Query(npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		vitals := (*components.VitalsComponent)(npcQuery.Get(s.vitalsID))

		if vitals.Blood <= 0 || vitals.Consciousness <= 0 {
			continue // Dead or unconscious NPCs don't participate in sieges
		}

		isGuard := false
		if world.Has(npcQuery.Entity(), s.jobID) {
			job := (*components.JobComponent)(world.Get(npcQuery.Entity(), s.jobID))
			if job.JobID == components.JobGuard || job.JobID == components.JobMercenary {
				isGuard = true
			}
		}

		npcs = append(npcs, npcNode{
			x:         pos.X,
			y:         pos.Y,
			countryID: aff.CountryID,
			isGuard:   isGuard,
		})
	}

	// 3. Evaluate Villages for Sieges
	type siegeApply struct {
		entity            ecs.Entity
		attackerCountryID uint32
	}
	var toApply []siegeApply
	var toRemove []ecs.Entity

	villageFilter := ecs.All(s.villageID, s.posID, s.affID)
	villageQuery := world.Query(villageFilter)
	for villageQuery.Next() {
		vEntity := villageQuery.Entity()
		pos := (*components.Position)(villageQuery.Get(s.posID))
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))

		// Check if this village's country is defending in any war
		var attackingCountry uint32 = 0
		for _, w := range activeWars {
			if w.defenderCountryID == aff.CountryID {
				attackingCountry = w.attackerCountryID
				break
			}
		}

		isCurrentlySieged := world.Has(vEntity, s.siegeID)

		if attackingCountry == 0 {
			if isCurrentlySieged {
				toRemove = append(toRemove, vEntity)
			}
			continue
		}

		// Count forces within siege radius (distSq <= 100.0)
		attackers := 0
		defenders := 0
		for _, n := range npcs {
			dx := n.x - pos.X
			dy := n.y - pos.Y
			if dx*dx+dy*dy <= 100.0 {
				if n.countryID == attackingCountry {
					if n.isGuard {
						attackers += 3 // Guards/Mercs count as 3
					} else {
						attackers += 1
					}
				} else if n.countryID == aff.CountryID {
					if n.isGuard {
						defenders += 3
					} else {
						defenders += 1
					}
				}
			}
		}

		// Apply or maintain siege
		if attackers > 0 && attackers > defenders*2 {
			if !isCurrentlySieged {
				toApply = append(toApply, siegeApply{
					entity:            vEntity,
					attackerCountryID: attackingCountry,
				})
			}

			// Execute active siege effects
			if world.Has(vEntity, s.marketID) {
				market := (*components.MarketComponent)(world.Get(vEntity, s.marketID))
				// Food prices spike massively due to isolation
				market.FoodPrice += 15.0
			}
			if world.Has(vEntity, s.loyaltyID) {
				loyalty := (*components.LoyaltyComponent)(world.Get(vEntity, s.loyaltyID))
				// Loyalty to the ruling capital drains as they fail to break the siege
				if loyalty.Value >= 5 {
					loyalty.Value -= 5
				} else {
					loyalty.Value = 0
				}
			}
		} else {
			if isCurrentlySieged {
				toRemove = append(toRemove, vEntity)
			}
		}
	}

	// 4. Apply structural changes
	for _, app := range toApply {
		if world.Alive(app.entity) && !world.Has(app.entity, s.siegeID) {
			world.Add(app.entity, s.siegeID)
			marker := (*components.SiegeMarker)(world.Get(app.entity, s.siegeID))
			marker.AttackerCountryID = app.attackerCountryID
		}
	}

	for _, e := range toRemove {
		if world.Alive(e) && world.Has(e, s.siegeID) {
			world.Remove(e, s.siegeID)
		}
	}
}
