package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 66 - The Physical Siege Engine
// SiegeSystem evaluates spatial military superiority around settlements.
// If hostile forces outnumber defending forces near a city during an active war,
// it applies a SiegeMarker, physically draining the city's food and loyalty,
// and spiking prices to organically trigger starvation and refugee events.

type structuralSiegeChange struct {
	entity            ecs.Entity
	addSiege          bool
	removeSiege       bool
	besiegerCountryID uint32
}

type siegeNpcData struct {
	CountryID uint32
	X         float32
	Y         float32
}

type siegeWarData struct {
	AttackerID uint32
	TargetID   uint32
}

type SiegeSystem struct {
	tickCounter uint64

	// Caches to avoid lock issues
	changes  []structuralSiegeChange
	wars     []siegeWarData
	npcs     []siegeNpcData

	// Filter and Component IDs
	villFilter ecs.Filter
	npcFilter  ecs.Filter

	villID    ecs.ID
	posID     ecs.ID
	affID     ecs.ID
	loyaltyID ecs.ID
	marketID  ecs.ID
	storageID ecs.ID
	warID     ecs.ID
	siegeID   ecs.ID
	npcCompID ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	villID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)

	vMask := ecs.All(villID, posID, affID, loyaltyID, marketID, storageID)

	npcCompID := ecs.ComponentID[components.NPC](world)
	npcMask := ecs.All(npcCompID, posID, affID)

	return &SiegeSystem{
		changes:   make([]structuralSiegeChange, 0, 10),
		wars:      make([]siegeWarData, 0, 10),
		npcs:      make([]siegeNpcData, 0, 100),
		villFilter: &vMask,
		npcFilter:  &npcMask,
		villID:    villID,
		posID:     posID,
		affID:     affID,
		loyaltyID: loyaltyID,
		marketID:  marketID,
		storageID: storageID,
		warID:     ecs.ComponentID[components.WarTrackerComponent](world),
		siegeID:   ecs.ComponentID[components.SiegeMarker](world),
		npcCompID: npcCompID,
	}
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Evaluate periodically (e.g. every 10 ticks)
	if s.tickCounter%10 != 0 {
		return
	}

	s.changes = s.changes[:0]

	// 1. Cache active wars (determinisic flat array)
	s.wars = s.wars[:0]
	// We query Capitals to find active wars
	capID := ecs.ComponentID[components.CapitalComponent](world)
	warQuery := world.Query(ecs.All(capID, s.affID, s.warID))
	for warQuery.Next() {
		aff := (*components.Affiliation)(warQuery.Get(s.affID))
		war := (*components.WarTrackerComponent)(warQuery.Get(s.warID))
		if war.Active {
			s.wars = append(s.wars, siegeWarData{
				AttackerID: aff.CountryID,
				TargetID:   war.TargetCountryID,
			})
		}
	}

	if len(s.wars) == 0 {
		// Clean up existing sieges if wars ended
		siegeQuery := world.Query(ecs.All(s.siegeID))
		for siegeQuery.Next() {
			s.changes = append(s.changes, structuralSiegeChange{
				entity:      siegeQuery.Entity(),
				removeSiege: true,
			})
		}
		s.applyStructuralChanges(world)
		return
	}

	// 2. Cache all NPC coordinates and affiliations
	s.npcs = s.npcs[:0]
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		if aff.CountryID == 0 {
			continue // Neutral NPCs don't siege
		}
		pos := (*components.Position)(npcQuery.Get(s.posID))
		s.npcs = append(s.npcs, siegeNpcData{
			CountryID: aff.CountryID,
			X:         pos.X,
			Y:         pos.Y,
		})
	}

	// 3. Iterate over Villages and apply/remove sieges, and apply siege effects
	villQuery := world.Query(s.villFilter)

	for villQuery.Next() {
		ent := villQuery.Entity()
		aff := (*components.Affiliation)(villQuery.Get(s.affID))
		pos := (*components.Position)(villQuery.Get(s.posID))
		market := (*components.MarketComponent)(villQuery.Get(s.marketID))
		storage := (*components.StorageComponent)(villQuery.Get(s.storageID))
		loyalty := (*components.LoyaltyComponent)(villQuery.Get(s.loyaltyID))

		// Check if this village is a target in any active war
		var activeAttackerCountryID uint32 = 0
		for _, war := range s.wars {
			if aff.CountryID == war.TargetID {
				activeAttackerCountryID = war.AttackerID
				break
			}
		}

		hasSiege := villQuery.Has(s.siegeID)

		if activeAttackerCountryID == 0 {
			if hasSiege {
				s.changes = append(s.changes, structuralSiegeChange{
					entity:      ent,
					removeSiege: true,
				})
			}
			continue
		}

		// Calculate numerical superiority
		hostileCount := 0
		friendlyCount := 0

		for _, n := range s.npcs {
			dx := n.X - pos.X
			dy := n.Y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq <= 25.0 {
				if n.CountryID == activeAttackerCountryID {
					hostileCount++
				} else if n.CountryID == aff.CountryID {
					friendlyCount++
				}
			}
		}

		isBesieged := hostileCount > friendlyCount && hostileCount > 0

		if isBesieged {
			if !hasSiege {
				s.changes = append(s.changes, structuralSiegeChange{
					entity:            ent,
					addSiege:          true,
					besiegerCountryID: activeAttackerCountryID,
				})
			}

			// Apply Siege Effects
			// 1. Spikes Food Price
			if market.FoodPrice < 100.0 {
				market.FoodPrice += 10.0
			}

			// 2. Drains Food (burn crops/blockade)
			if storage.Food >= 5 {
				storage.Food -= 5
			} else {
				storage.Food = 0
				// 3. Breaks Loyalty if starving
				if loyalty.Value > 0 {
					loyalty.Value -= 1
				}
			}

		} else {
			if hasSiege {
				s.changes = append(s.changes, structuralSiegeChange{
					entity:      ent,
					removeSiege: true,
				})
			}
		}
	}

	// 4. Apply structural changes safely
	s.applyStructuralChanges(world)
}

func (s *SiegeSystem) applyStructuralChanges(world *ecs.World) {
	for _, change := range s.changes {
		if world.Alive(change.entity) {
			if change.addSiege {
				if !world.Has(change.entity, s.siegeID) {
					world.Add(change.entity, s.siegeID)
				}
				// Fetch fresh pointer due to Archetype swap
				marker := (*components.SiegeMarker)(world.Get(change.entity, s.siegeID))
				marker.BesiegerCountryID = change.besiegerCountryID
			} else if change.removeSiege {
				if world.Has(change.entity, s.siegeID) {
					world.Remove(change.entity, s.siegeID)
				}
			}
		}
	}
}
