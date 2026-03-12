package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 13.2: Labor Rebalancing
// CareerChangeSystem forces specific lower-tier processing NPCs analyzing the
// JobComponent parameters to strictly adopt base Farmer/Lumberjack flags
// if Price values mapping Grain/Wood severely cross bounds limits.

type CareerChangeSystem struct{}

func NewCareerChangeSystem() *CareerChangeSystem {
	return &CareerChangeSystem{}
}

func (s *CareerChangeSystem) Update(world *ecs.World) {
	villageID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	identityID := ecs.ComponentID[components.Identity](world)

	// Step 1: Pre-calculate flat map of village market prices to retain DOD O(1) matching
	// during the next iteration loop without nested queries.
	villageFilter := ecs.All(villageID, marketID, identityID)
	villageQuery := world.Query(villageFilter)

	// Use uint32 for CityID matching against Affiliation.CityID
	marketPrices := make(map[uint32]*components.MarketComponent)

	for villageQuery.Next() {
		market := (*components.MarketComponent)(villageQuery.Get(marketID))
		identity := (*components.Identity)(villageQuery.Get(identityID))
		// Identity.ID represents the CityID for Affiliation maps
		marketPrices[uint32(identity.ID)] = market
	}

	// Step 2: Iterate over NPCs holding jobs and affiliations
	jobID := ecs.ComponentID[components.JobComponent](world)
	affiliationID := ecs.ComponentID[components.Affiliation](world)

	npcFilter := ecs.All(jobID, affiliationID)
	npcQuery := world.Query(npcFilter)

	for npcQuery.Next() {
		job := (*components.JobComponent)(npcQuery.Get(jobID))
		affiliation := (*components.Affiliation)(npcQuery.Get(affiliationID))

		// Check if NPC is in a processing job that can be reverted
		if job.JobID != components.JobArtisan {
			continue
		}

		// Instant O(1) hashmap lookup using pre-calculated values
		market, exists := marketPrices[affiliation.CityID]
		if !exists {
			continue
		}

		// Phase 13.2: Labor Rebalancing Logic
		// Strict bounds enforcement based on famine/resource shortages.
		if market.FoodPrice > 10.0 {
			// Severe food shortage -> Revert to Farmer
			job.JobID = components.JobFarmer
		} else if market.WoodPrice > 10.0 {
			// Severe wood shortage -> Revert to Lumberjack
			job.JobID = components.JobLumberjack
		}
	}
}
