package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.2: Strategic Unions & Pacts

// PriceNormalizationSystem ensures goods flow freely without tariff penalties
// by calculating and applying the average market prices across all members of a Currency Union.
type PriceNormalizationSystem struct {
	world *ecs.World

	// Pre-allocated maps for O(1) matching during iteration to avoid nested queries
	cityMarkets map[uint32]*components.MarketComponent

	// Component IDs
	unionEntityID ecs.ID
	unionCompID   ecs.ID
	villageID     ecs.ID
	affilID       ecs.ID
	marketID      ecs.ID
}

// NewPriceNormalizationSystem initializes the system
func NewPriceNormalizationSystem(world *ecs.World) *PriceNormalizationSystem {
	return &PriceNormalizationSystem{
		world:         world,
		cityMarkets:   make(map[uint32]*components.MarketComponent),
		unionEntityID: ecs.ComponentID[components.UnionEntity](world),
		unionCompID:   ecs.ComponentID[components.UnionComponent](world),
		villageID:     ecs.ComponentID[components.Village](world),
		affilID:       ecs.ComponentID[components.Affiliation](world),
		marketID:      ecs.ComponentID[components.MarketComponent](world),
	}
}

// Update calculates average prices for each currency union and sets them
func (s *PriceNormalizationSystem) Update() {
	// 1. Build a flat map of all active City MarketComponents for O(1) lookups
	clear(s.cityMarkets)

	// In this engine, cities are identified by either their own Identity.ID or Affiliation.CityID
	// We will use the Affiliation.CityID for standardized mapping since they are Village entities.
	cityQuery := s.world.Query(filter.All(s.villageID, s.affilID, s.marketID))
	for cityQuery.Next() {
		affil := (*components.Affiliation)(cityQuery.Get(s.affilID))
		market := (*components.MarketComponent)(cityQuery.Get(s.marketID))
		s.cityMarkets[affil.CityID] = market
	}

	if len(s.cityMarkets) == 0 {
		return // No active markets to normalize
	}

	// 2. Iterate over all Currency Unions
	unionQuery := s.world.Query(filter.All(s.unionEntityID, s.unionCompID))
	for unionQuery.Next() {
		union := (*components.UnionComponent)(unionQuery.Get(s.unionCompID))

		// Only process Currency Unions
		if union.UnionType != components.UnionCurrency {
			continue
		}

		if len(union.MemberIDs) == 0 {
			continue
		}

		var totalWoodPrice float32
		var totalStonePrice float32
		var totalIronPrice float32
		var totalFoodPrice float32
		var activeMembers float32

		// 3. Gather total prices across valid member cities
		for _, memberID := range union.MemberIDs {
			if market, ok := s.cityMarkets[memberID]; ok {
				totalWoodPrice += market.WoodPrice
				totalStonePrice += market.StonePrice
				totalIronPrice += market.IronPrice
				totalFoodPrice += market.FoodPrice
				activeMembers += 1.0
			}
		}

		// 4. Calculate average and apply back to members
		if activeMembers > 0 {
			avgWoodPrice := totalWoodPrice / activeMembers
			avgStonePrice := totalStonePrice / activeMembers
			avgIronPrice := totalIronPrice / activeMembers
			avgFoodPrice := totalFoodPrice / activeMembers

			for _, memberID := range union.MemberIDs {
				if market, ok := s.cityMarkets[memberID]; ok {
					market.WoodPrice = avgWoodPrice
					market.StonePrice = avgStonePrice
					market.IronPrice = avgIronPrice
					market.FoodPrice = avgFoodPrice
				}
			}
		}
	}
}
