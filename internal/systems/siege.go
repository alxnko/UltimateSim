package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: The Physical Siege Engine
// SiegeSystem connects Geography, Combat, and Economics.
// It applies a SiegeMarker to Villages that are outnumbered by hostile NPCs during wars.
// This organically spikes local MarketComponent food prices and drains LoyaltyComponent to simulate starvation.

const SiegeTickRate = 100
const SiegeRadiusSquared float32 = 100.0

type SiegeSystem struct {
	tickCounter uint64

	// Component IDs
	capID     ecs.ID
	warCompID ecs.ID
	villageID ecs.ID
	affilID   ecs.ID
	posID     ecs.ID
	npcID     ecs.ID
	identID   ecs.ID
	siegeID   ecs.ID
	marketID  ecs.ID
	loyaltyID ecs.ID
}

// NewSiegeSystem creates a new SiegeSystem.
func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		tickCounter: 0,

		capID:     ecs.ComponentID[components.CapitalComponent](world),
		warCompID: ecs.ComponentID[components.WarTrackerComponent](world),
		villageID: ecs.ComponentID[components.Village](world),
		affilID:   ecs.ComponentID[components.Affiliation](world),
		posID:     ecs.ComponentID[components.Position](world),
		npcID:     ecs.ComponentID[components.NPC](world),
		identID:   ecs.ComponentID[components.Identity](world),
		siegeID:   ecs.ComponentID[components.SiegeMarker](world),
		marketID:  ecs.ComponentID[components.MarketComponent](world),
		loyaltyID: ecs.ComponentID[components.LoyaltyComponent](world),
	}
}

type siegeNpcData struct {
	CountryID uint32
	X         float32
	Y         float32
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	if s.tickCounter%SiegeTickRate != 0 {
		return
	}

	// 1. Identify active wars to see who is fighting whom.
	// Map AttackerCountryID -> slice of TargetCountryIDs (to handle multiple wars)
	activeWars := make(map[uint32][]uint32)

	warFilter := filter.All(s.capID, s.warCompID, s.affilID)
	warQuery := world.Query(warFilter)
	for warQuery.Next() {
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warCompID))
		if !war.Active {
			continue
		}
		affil := (*components.Affiliation)(warQuery.Get(s.affilID))
		activeWars[affil.CountryID] = append(activeWars[affil.CountryID], war.TargetCountryID)
	}

	// If there are no wars, sieges shouldn't happen. Clean up existing sieges and exit early.
	if len(activeWars) == 0 {
		s.cleanupSieges(world)
		return
	}

	// 2. Pre-cache all relevant NPC positions and their CountryID to avoid O(N^2) Arche-Go queries.
	var npcDataList []siegeNpcData
	npcFilter := filter.All(s.npcID, s.affilID, s.posID)
	npcQuery := world.Query(npcFilter)
	for npcQuery.Next() {
		affil := (*components.Affiliation)(npcQuery.Get(s.affilID))
		pos := (*components.Position)(npcQuery.Get(s.posID))

		// Only consider NPCs that belong to a country
		if affil.CountryID != 0 {
			npcDataList = append(npcDataList, siegeNpcData{
				CountryID: affil.CountryID,
				X:         pos.X,
				Y:         pos.Y,
			})
		}
	}

	// 3. Evaluate each village for siege conditions.
	// We defer structural changes (adding/removing SiegeMarker) to after the query iteration.
	type structuralChange struct {
		entity      ecs.Entity
		addMarker   bool
		besiegerID  uint32
	}
	changes := make([]structuralChange, 0)

	villageFilter := filter.All(s.villageID, s.affilID, s.posID, s.marketID, s.loyaltyID)
	villageQuery := world.Query(villageFilter)
	for villageQuery.Next() {
		entity := villageQuery.Entity()
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))
		pos := (*components.Position)(villageQuery.Get(s.posID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))
		hasSiegeMarker := world.Has(entity, s.siegeID)

		villageCountryID := affil.CountryID
		if villageCountryID == 0 {
			if hasSiegeMarker {
				changes = append(changes, structuralChange{entity: entity, addMarker: false})
			}
			continue
		}

		// Count friendly and hostile NPCs in radius
		friendlyCount := 0

		// Map AttackerCountryID -> Count of hostile NPCs
		hostileCounts := make(map[uint32]int)

		for _, npc := range npcDataList {
			dx := npc.X - pos.X
			dy := npc.Y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq <= SiegeRadiusSquared {
				if npc.CountryID == villageCountryID {
					friendlyCount++
				} else {
					// Check if this NPC's country is at war with the village's country
					targetCountryIDs, isAtWar := activeWars[npc.CountryID]
					if isAtWar {
						for _, tID := range targetCountryIDs {
							if tID == villageCountryID {
								hostileCounts[npc.CountryID]++
								break
							}
						}
					}
				}
			}
		}

		// Determine if the village is under siege.
		// It is under siege if a single hostile country has more troops than the friendly troops.
		isBesieged := false
		var dominantBesiegerID uint32 = 0
		maxHostiles := 0

		for attackerID, count := range hostileCounts {
			if count > friendlyCount && count > maxHostiles {
				isBesieged = true
				dominantBesiegerID = attackerID
				maxHostiles = count
			}
		}

		if isBesieged {
			if !hasSiegeMarker {
				changes = append(changes, structuralChange{
					entity:     entity,
					addMarker:  true,
					besiegerID: dominantBesiegerID,
				})
			} else {
				// Already under siege, apply the continuous penalties
				market.FoodPrice += 10.0 // Extreme starvation

				if loyalty.Value >= 5 {
					loyalty.Value -= 5
				} else {
					loyalty.Value = 0
				}
			}
		} else {
			if hasSiegeMarker {
				changes = append(changes, structuralChange{entity: entity, addMarker: false})
			}
		}
	}

	// 4. Apply structural changes safely outside of the Arche-Go iteration lock.
	for _, change := range changes {
		if change.addMarker {
			world.Add(change.entity, s.siegeID)
			// Re-fetch pointer due to swap-and-pop hazard
			marker := (*components.SiegeMarker)(world.Get(change.entity, s.siegeID))
			marker.BesiegerCountryID = change.besiegerID
		} else {
			world.Remove(change.entity, s.siegeID)
		}
	}
}

// cleanupSieges removes all SiegeMarker components when there are no active wars.
func (s *SiegeSystem) cleanupSieges(world *ecs.World) {
	filter := filter.All(s.siegeID)
	query := world.Query(filter)

	var toRemove []ecs.Entity
	for query.Next() {
		toRemove = append(toRemove, query.Entity())
	}

	for _, entity := range toRemove {
		world.Remove(entity, s.siegeID)
	}
}
