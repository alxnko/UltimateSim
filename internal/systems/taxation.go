package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.1: The Country Entity (Macro-State)

// TaxationSystem manages the transfer of wealth from sub-cities to their governing Country Capital.
type TaxationSystem struct {
	world *ecs.World

	// Component IDs
	countryID  ecs.ID
	capitalID  ecs.ID
	affilID    ecs.ID
	treasuryID ecs.ID
	villageID  ecs.ID
	marketID   ecs.ID

	tickStamp uint64
}

// NewTaxationSystem initializes the taxation loop.
func NewTaxationSystem(world *ecs.World) *TaxationSystem {
	return &TaxationSystem{
		world:      world,
		countryID:  ecs.ComponentID[components.CountryComponent](world),
		capitalID:  ecs.ComponentID[components.CapitalComponent](world),
		affilID:    ecs.ComponentID[components.Affiliation](world),
		treasuryID: ecs.ComponentID[components.TreasuryComponent](world),
		villageID:  ecs.ComponentID[components.Village](world),
		marketID:   ecs.ComponentID[components.MarketComponent](world),
	}
}

// Update processes taxation strictly every 100 ticks to avoid simulation loops lag.
func (s *TaxationSystem) Update() {
	s.tickStamp++

	if s.tickStamp%100 != 0 {
		return
	}

	// 1. Build a flat map of active Country Treasuries for O(1) matching.
	countryTreasuries := make(map[uint32]*components.TreasuryComponent)

	// A Country Capital must have CountryComponent, CapitalComponent, Affiliation, and TreasuryComponent.
	capitalQuery := s.world.Query(filter.All(s.countryID, s.capitalID, s.affilID, s.treasuryID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(s.affilID))
		treasury := (*components.TreasuryComponent)(capitalQuery.Get(s.treasuryID))

		// Map the treasury using the CountryID
		countryTreasuries[affil.CountryID] = treasury
	}

	// If no countries exist, skip village taxation completely to save CPU cycles.
	if len(countryTreasuries) == 0 {
		return
	}

	// 2. Iterate over all Villages with an Affiliation, MarketComponent, and TreasuryComponent.
	villageQuery := s.world.Query(filter.All(s.villageID, s.affilID, s.marketID, s.treasuryID))
	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		treasury := (*components.TreasuryComponent)(villageQuery.Get(s.treasuryID))

		// Check if the village belongs to a valid country.
		if countryTreasury, ok := countryTreasuries[affil.CountryID]; ok {
			// Sub-cities transfer a portion of their MarketComponent revenue to the Country's TreasuryComponent.
			// Calculate tax base linearly off current local market prices (e.g. higher demand = higher tax).
			taxAmount := (market.FoodPrice + market.WoodPrice + market.StonePrice + market.IronPrice) * 1.0

			// Deduct the tax if the Village has sufficient wealth.
			if treasury.Wealth >= taxAmount {
				treasury.Wealth -= taxAmount
				countryTreasury.Wealth += taxAmount
			} else {
				// If insufficient wealth, drain whatever is left.
				countryTreasury.Wealth += treasury.Wealth
				treasury.Wealth = 0.0
			}
		}
	}
}
