package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
	"sort"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem bridges Geography, Combat, and Economics by applying a SiegeMarker
// to Villages outnumbered by hostile NPCs during wars, organically spiking
// MarketComponent food prices and draining LoyaltyComponent to simulate starvation.

type siegeNpcData struct {
	CountryID uint32
	X         float32
	Y         float32
}

type structuralChange struct {
	entity            ecs.Entity
	cID               ecs.ID
	add               bool
	BesiegerCountryID uint32
}

type SiegeSystem struct {
	tickCounter uint64

	// Component IDs
	npcID     ecs.ID
	posID     ecs.ID
	affID     ecs.ID
	villageID ecs.ID
	storageID ecs.ID
	marketID  ecs.ID
	loyaltyID ecs.ID
	siegeID   ecs.ID

	// DOD Slices
	npcs    []siegeNpcData
	changes []structuralChange

	// Cache for sorting to avoid allocation
	enemyCountries []uint32
}

// NewSiegeSystem creates a new SiegeSystem.
func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	return &SiegeSystem{
		tickCounter: 0,
		npcID:       ecs.ComponentID[components.NPC](world),
		posID:       ecs.ComponentID[components.Position](world),
		affID:       ecs.ComponentID[components.Affiliation](world),
		villageID:   ecs.ComponentID[components.Village](world),
		storageID:   ecs.ComponentID[components.StorageComponent](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
		loyaltyID:   ecs.ComponentID[components.LoyaltyComponent](world),
		siegeID:     ecs.ComponentID[components.SiegeMarker](world),
		npcs:        make([]siegeNpcData, 0, 1000),
		changes:     make([]structuralChange, 0, 100),
		enemyCountries: make([]uint32, 0, 10),
	}
}

// Update evaluates active physical sieges.
func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process sieges periodically
	if s.tickCounter%30 != 0 {
		return
	}

	// Reset slices
	s.npcs = s.npcs[:0]
	s.changes = s.changes[:0]

	// 1. Extract all active NPCs into a flat DOD slice
	npcFilter := filter.All(s.npcID, s.posID, s.affID)
	npcQuery := world.Query(npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		s.npcs = append(s.npcs, siegeNpcData{
			CountryID: aff.CountryID,
			X:         pos.X,
			Y:         pos.Y,
		})
	}

	// 2. Iterate all Villages to calculate siege state
	villageFilter := filter.All(s.villageID, s.posID, s.affID, s.storageID, s.marketID, s.loyaltyID)
	villageQuery := world.Query(villageFilter)

	for villageQuery.Next() {
		entity := villageQuery.Entity()
		pos := (*components.Position)(villageQuery.Get(s.posID))
		aff := (*components.Affiliation)(villageQuery.Get(s.affID))
		storage := (*components.StorageComponent)(villageQuery.Get(s.storageID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))

		friendlyCount := 0
		s.enemyCountries = s.enemyCountries[:0]

		// O(N) Spatial check without nested ECS queries
		for i := 0; i < len(s.npcs); i++ {
			npc := &s.npcs[i]
			dx := npc.X - pos.X
			dy := npc.Y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq < 25.0 {
				if npc.CountryID == aff.CountryID {
					friendlyCount++
				} else if npc.CountryID != 0 {
					s.enemyCountries = append(s.enemyCountries, npc.CountryID)
				}
			}
		}

		// Sort to ensure determinism
		sort.Slice(s.enemyCountries, func(i, j int) bool {
			return s.enemyCountries[i] < s.enemyCountries[j]
		})

		// Count occurrences after sorting
		var maxEnemyCountry uint32
		var maxEnemyCount int

		if len(s.enemyCountries) > 0 {
			currentCountry := s.enemyCountries[0]
			currentCount := 1

			for i := 1; i < len(s.enemyCountries); i++ {
				if s.enemyCountries[i] == currentCountry {
					currentCount++
				} else {
					if currentCount > maxEnemyCount {
						maxEnemyCount = currentCount
						maxEnemyCountry = currentCountry
					}
					currentCountry = s.enemyCountries[i]
					currentCount = 1
				}
			}
			// Final check
			if currentCount > maxEnemyCount {
				maxEnemyCount = currentCount
				maxEnemyCountry = currentCountry
			}
		}

		isBesieged := maxEnemyCount > friendlyCount && maxEnemyCount > 0
		hasSiegeMarker := world.Has(entity, s.siegeID)

		if isBesieged {
			if !hasSiegeMarker {
				// Queue adding the marker
				s.changes = append(s.changes, structuralChange{
					entity:            entity,
					cID:               s.siegeID,
					add:               true,
					BesiegerCountryID: maxEnemyCountry,
				})
			} else {
				// Already besieged: apply effects
				siege := (*components.SiegeMarker)(villageQuery.Get(s.siegeID))

				// 1. Storage Depletion (Blockade)
				if storage.Food > 10 {
					storage.Food -= 10
				} else {
					storage.Food = 0
				}

				// 2. Price Spike (Starvation/Hoarding)
				market.FoodPrice += 20.0

				// 3. Loyalty Drain
				if loyalty.Value > 10 {
					loyalty.Value -= 10
				} else {
					loyalty.Value = 0

					// Capitulation! Village flips to the besieging country
					aff.CountryID = siege.BesiegerCountryID

					// Queue removing the marker (siege lifted)
					s.changes = append(s.changes, structuralChange{
						entity: entity,
						cID:    s.siegeID,
						add:    false,
					})
				}
			}
		} else {
			if hasSiegeMarker {
				// Relieved: Queue removing the marker
				s.changes = append(s.changes, structuralChange{
					entity: entity,
					cID:    s.siegeID,
					add:    false,
				})
			}
		}
	}

	// 3. Apply Structural ECS Changes
	for i := 0; i < len(s.changes); i++ {
		c := s.changes[i]
		if c.add {
			if !world.Has(c.entity, c.cID) {
				world.Add(c.entity, c.cID)
			}
			marker := (*components.SiegeMarker)(world.Get(c.entity, c.cID))
			marker.BesiegerCountryID = c.BesiegerCountryID
		} else {
			if world.Has(c.entity, c.cID) {
				world.Remove(c.entity, c.cID)
			}
		}
	}
}
