package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by applying an 8-byte
// SiegeMarker to Villages outnumbered by hostile NPCs during wars, organically
// spiking local MarketComponent food prices and draining LoyaltyComponent to simulate starvation.

type siegeNpcData struct {
	Entity    ecs.Entity
	X         float32
	Y         float32
	CountryID uint32
}

type siegeVillageData struct {
	Entity    ecs.Entity
	X         float32
	Y         float32
	CountryID uint32
}

type SiegeSystem struct {
	// Component IDs
	posID         ecs.ID
	affID         ecs.ID
	marketID      ecs.ID
	loyaltyID     ecs.ID
	villageID     ecs.ID
	npcID         ecs.ID
	warTrackerID  ecs.ID
	siegeMarkerID ecs.ID

	// Filters
	warFilter     ecs.Filter
	npcFilter     ecs.Filter
	villageFilter ecs.Filter

	// Pre-allocated slices
	wars     []struct{ Attacker, Target uint32 }
	npcs     []siegeNpcData
	villages []siegeVillageData
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	npcID := ecs.ComponentID[components.NPC](world)
	warTrackerID := ecs.ComponentID[components.WarTrackerComponent](world)
	siegeMarkerID := ecs.ComponentID[components.SiegeMarker](world)

	wFilter := filter.All(warTrackerID, affID)
	nFilter := filter.All(npcID, posID, affID)
	vFilter := filter.All(villageID, posID, affID, marketID, loyaltyID)

	return &SiegeSystem{
		posID:         posID,
		affID:         affID,
		marketID:      marketID,
		loyaltyID:     loyaltyID,
		villageID:     villageID,
		npcID:         npcID,
		warTrackerID:  warTrackerID,
		siegeMarkerID: siegeMarkerID,
		warFilter:     &wFilter,
		npcFilter:     &nFilter,
		villageFilter: &vFilter,
		wars:          make([]struct{ Attacker, Target uint32 }, 0, 10),
		npcs:          make([]siegeNpcData, 0, 100),
		villages:      make([]siegeVillageData, 0, 20),
	}
}

func (s *SiegeSystem) Update(world *ecs.World) {
	// 1. Gather Active Wars
	s.wars = s.wars[:0]
	warQuery := world.Query(s.warFilter)
	for warQuery.Next() {
		aff := (*components.Affiliation)(warQuery.Get(s.affID))
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warTrackerID))
		if war.Active {
			s.wars = append(s.wars, struct{ Attacker, Target uint32 }{Attacker: aff.CountryID, Target: war.TargetCountryID})
		}
	}

	// Fast exit if no wars
	if len(s.wars) == 0 {
		return
	}

	// 2. Gather all NPCs
	s.npcs = s.npcs[:0]
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		if aff.CountryID != 0 {
			s.npcs = append(s.npcs, siegeNpcData{
				Entity:    npcQuery.Entity(),
				X:         pos.X,
				Y:         pos.Y,
				CountryID: aff.CountryID,
			})
		}
	}

	// 3. Gather all Villages
	s.villages = s.villages[:0]
	villageQuery := world.Query(s.villageFilter)
	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(s.posID))
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		if aff.CountryID != 0 {
			s.villages = append(s.villages, siegeVillageData{
				Entity:    villageQuery.Entity(),
				X:         pos.X,
				Y:         pos.Y,
				CountryID: aff.CountryID,
			})
		}
	}

	// 4. Evaluate Sieges
	type siegeChange struct {
		Entity            ecs.Entity
		BesiegerCountryID uint32
		ApplySiege        bool
	}
	var changes []siegeChange

	for i := 0; i < len(s.villages); i++ {
		village := s.villages[i]

		// Map hostile countries based on active wars
		// Note: The logic handles both Attackers vs Target and Defenders vs Attacker (in self defense)
		hostileCountries := make(map[uint32]bool)
		for _, w := range s.wars {
			if w.Target == village.CountryID {
				hostileCountries[w.Attacker] = true
			}
			if w.Attacker == village.CountryID {
				hostileCountries[w.Target] = true
			}
		}

		if len(hostileCountries) == 0 {
			continue // No active wars against this village's country
		}

		var defenders int
		var hostiles int
		var primaryBesieger uint32

		for j := 0; j < len(s.npcs); j++ {
			npc := s.npcs[j]

			dx := village.X - npc.X
			dy := village.Y - npc.Y
			distSq := dx*dx + dy*dy

			if distSq <= 100.0 {
				if npc.CountryID == village.CountryID {
					defenders++
				} else if hostileCountries[npc.CountryID] {
					hostiles++
					primaryBesieger = npc.CountryID // Just pick one of the hostile countries
				}
			}
		}

		// Siege logic: hostiles outnumber defenders and hostiles > 0
		isBesieged := hostiles > 0 && hostiles > defenders

		if isBesieged {
			changes = append(changes, siegeChange{Entity: village.Entity, BesiegerCountryID: primaryBesieger, ApplySiege: true})
		} else if world.Has(village.Entity, s.siegeMarkerID) {
			// Siege broken
			changes = append(changes, siegeChange{Entity: village.Entity, ApplySiege: false})
		}
	}

	// 5. Apply Structural Changes and Emergent Effects (outside of query)
	for _, change := range changes {
		if change.ApplySiege {
			// Apply SiegeMarker structurally if not already present
			if !world.Has(change.Entity, s.siegeMarkerID) {
				world.Add(change.Entity, s.siegeMarkerID)
			}
			// Safe to get pointer after structural change
			marker := (*components.SiegeMarker)(world.Get(change.Entity, s.siegeMarkerID))
			marker.BesiegerCountryID = change.BesiegerCountryID

			// Apply emergent effects: Spike Food Price, Drain Loyalty
			market := (*components.MarketComponent)(world.Get(change.Entity, s.marketID))
			loyalty := (*components.LoyaltyComponent)(world.Get(change.Entity, s.loyaltyID))

			market.FoodPrice += 5.0
			if loyalty.Value >= 5 {
				loyalty.Value -= 5
			} else {
				loyalty.Value = 0
			}
		} else {
			// Remove SiegeMarker structurally
			if world.Has(change.Entity, s.siegeMarkerID) {
				world.Remove(change.Entity, s.siegeMarkerID)
			}
		}
	}
}
