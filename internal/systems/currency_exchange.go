package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 15.3: Currency & Debt

// CurrencyExchangeSystem updates global currency values based on Capital prestige and resources.
type CurrencyExchangeSystem struct {
	world *ecs.World

	// Component IDs
	capitalID  ecs.ID
	affilID    ecs.ID
	storageID  ecs.ID
	legacyID   ecs.ID
	coinTagID  ecs.ID
	currencyID ecs.ID

	tickStamp uint64
	// marketRates maps an IssuerID (CityID) to its dynamically evaluated global exchange rate
	marketRates map[uint32]float32
}

// NewCurrencyExchangeSystem initializes the system for tracking currency exchange value.
func NewCurrencyExchangeSystem(world *ecs.World) *CurrencyExchangeSystem {
	return &CurrencyExchangeSystem{
		world:       world,
		capitalID:   ecs.ComponentID[components.CapitalComponent](world),
		affilID:     ecs.ComponentID[components.Affiliation](world),
		storageID:   ecs.ComponentID[components.StorageComponent](world),
		legacyID:    ecs.ComponentID[components.Legacy](world),
		coinTagID:   ecs.ComponentID[components.CoinEntity](world),
		currencyID:  ecs.ComponentID[components.CurrencyComponent](world),
		marketRates: make(map[uint32]float32),
	}
}

// Update evaluates all Capital cities to recalculate currency values and applies them to active coins.
func (s *CurrencyExchangeSystem) Update() {
	s.tickStamp++

	// Only process the global exchange rate evaluation every 100 ticks
	if s.tickStamp%100 != 0 {
		return
	}

	// Clear the global market rates map to ensure destroyed capitals do not persist stale rates
	for k := range s.marketRates {
		delete(s.marketRates, k)
	}

	// 1. Calculate market exchange rates for all active issuing Capitals
	query := s.world.Query(filter.All(s.capitalID, s.affilID, s.storageID, s.legacyID))
	for query.Next() {
		affil := (*components.Affiliation)(query.Get(s.affilID))
		storage := (*components.StorageComponent)(query.Get(s.storageID))
		legacy := (*components.Legacy)(query.Get(s.legacyID))

		// Exchange Rate = 1.0 + (Prestige / 100.0) + (Total Resources / 1000.0)
		totalResources := float32(storage.Wood + storage.Stone + storage.Iron + storage.Food)
		rate := 1.0 + (float32(legacy.Prestige) / 100.0) + (totalResources / 1000.0)

		// O(1) hashmap store for Phase 15.3 DOD caching
		s.marketRates[affil.CityID] = rate
	}

	// 2. Apply newly calculated global exchange rates to physical CoinEntities
	coinQuery := s.world.Query(filter.All(s.coinTagID, s.currencyID))
	for coinQuery.Next() {
		curr := (*components.CurrencyComponent)(coinQuery.Get(s.currencyID))

		// If a valid exchange rate exists for the issuer, update the coin's intrinsic value
		if rate, exists := s.marketRates[curr.IssuerID]; exists {
			curr.Value = rate
		} else {
			// If issuer Capital is destroyed or unmapped, value craters to near 0 but physically exists
			curr.Value = 0.01
		}
	}
}
