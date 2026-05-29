package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 66: Physical Siege Engine
// SiegeSystem evaluates active war zones. If a Village is outnumbered by hostile
// NPCs within a certain radius, it places a SiegeMarker on the Village.
// When a Village has a SiegeMarker, it mathematically spikes FoodPrice and
// drains Loyalty, effectively starving the Village.

type SiegeSystem struct {
	tickCounter uint64

	villageFilter ecs.Filter
	npcFilter     ecs.Filter

	// Component IDs
	posID       ecs.ID
	affID       ecs.ID
	villageID   ecs.ID
	capID       ecs.ID
	warCompID   ecs.ID
	marketID    ecs.ID
	loyaltyID   ecs.ID
	npcID       ecs.ID
	siegeID     ecs.ID
}

func NewSiegeSystem(world *ecs.World) *SiegeSystem {
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	villageID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)

	vFilter := filter.All(posID, affID, villageID, marketID, loyaltyID)

	npcID := ecs.ComponentID[components.NPC](world)
	nFilter := filter.All(posID, affID, npcID)

	return &SiegeSystem{
		villageFilter: vFilter,
		npcFilter:     nFilter,
		posID:         posID,
		affID:         affID,
		villageID:     villageID,
		capID:         ecs.ComponentID[components.CapitalComponent](world),
		warCompID:     ecs.ComponentID[components.WarTrackerComponent](world),
		marketID:      marketID,
		loyaltyID:     loyaltyID,
		npcID:         npcID,
		siegeID:       ecs.ComponentID[components.SiegeMarker](world),
	}
}

func (s *SiegeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process every 10 ticks to spread CPU load but remain responsive
	if s.tickCounter%10 != 0 {
		return
	}

	// 1. Gather all active Wars mapping Attacker to Defender and vice-versa
	// A map of CountryID -> list of Enemy CountryIDs
	enemies := make(map[uint32][]uint32)
	hasAnyWar := false

	capQuery := world.Query(ecs.All(s.capID, s.warCompID, s.affID))
	for capQuery.Next() {
		war := (*components.WarTrackerComponent)(capQuery.Get(s.warCompID))
		aff := (*components.Affiliation)(capQuery.Get(s.affID))
		if war.Active {
			attacker := aff.CountryID
			defender := war.TargetCountryID
			enemies[attacker] = append(enemies[attacker], defender)
			enemies[defender] = append(enemies[defender], attacker)
			hasAnyWar = true
		}
	}

	if !hasAnyWar {
		return // No active wars, no sieges
	}

	// 2. Pre-cache all NPCs for spatial checks
	type siegeNpcData struct {
		x         float32
		y         float32
		countryID uint32
	}
	var npcs []siegeNpcData
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))
		npcs = append(npcs, siegeNpcData{
			x:         pos.X,
			y:         pos.Y,
			countryID: aff.CountryID,
		})
	}

	// 3. Process Villages
	query := world.Query(s.villageFilter)
	var addSiege []struct {
		entity     ecs.Entity
		besiegerID uint32
	}
	var removeSiege []ecs.Entity

	for query.Next() {
		entity := query.Entity()
		pos := (*components.Position)(query.Get(s.posID))
		aff := (*components.Affiliation)(query.Get(s.affID))
		market := (*components.MarketComponent)(query.Get(s.marketID))
		loyalty := (*components.LoyaltyComponent)(query.Get(s.loyaltyID))

		enemyCountries := enemies[aff.CountryID]
		if len(enemyCountries) == 0 {
			if world.Has(entity, s.siegeID) {
				removeSiege = append(removeSiege, entity)
			}
			continue
		}

		// Count defending vs hostile NPCs within a spatial radius (e.g., 25.0 squared distance)
		const siegeRadiusSq = 25.0
		defenders := 0
		var attackers map[uint32]int
		attackers = make(map[uint32]int)

		for _, npc := range npcs {
			dx := npc.x - pos.X
			dy := npc.y - pos.Y
			distSq := dx*dx + dy*dy

			if distSq <= siegeRadiusSq {
				if npc.countryID == aff.CountryID {
					defenders++
				} else {
					for _, enemyID := range enemyCountries {
						if npc.countryID == enemyID {
							attackers[enemyID]++
							break
						}
					}
				}
			}
		}

		// Find the strongest besieging force
		var maxAttackers int
		var besiegerID uint32

		// To ensure determinism, we need to extract and sort keys if there are ties.
		// However, a simpler way is to just iterate enemyCountries directly, which is a slice and therefore deterministic.
		for _, enemyID := range enemyCountries {
			count := attackers[enemyID]
			// Tie breaking: if counts are equal, higher enemyID wins (deterministic)
			if count > maxAttackers || (count == maxAttackers && enemyID > besiegerID && count > 0) {
				maxAttackers = count
				besiegerID = enemyID
			}
		}

		isBesieged := maxAttackers > defenders && maxAttackers > 0

		if isBesieged {
			if !world.Has(entity, s.siegeID) {
				addSiege = append(addSiege, struct {
					entity     ecs.Entity
					besiegerID uint32
				}{entity, besiegerID})
			} else {
				// Already under siege: Apply the effects of the siege!
				// Spiking food prices locally
				market.FoodPrice += 2.0

				// Drain loyalty
				if loyalty.Value > 0 {
					loyalty.Value--
				}
			}
		} else {
			if world.Has(entity, s.siegeID) {
				removeSiege = append(removeSiege, entity)
			}
		}
	}

	// 4. Apply structural changes (Add/Remove SiegeMarker)
	for _, req := range addSiege {
		world.Add(req.entity, s.siegeID)
		siege := (*components.SiegeMarker)(world.Get(req.entity, s.siegeID))
		siege.BesiegerCountryID = req.besiegerID
	}

	for _, entity := range removeSiege {
		world.Remove(entity, s.siegeID)
	}
}
