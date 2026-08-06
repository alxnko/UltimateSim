package systems

// Grand Strategy Phase (P2.6): Economy layer — tax policy, trade routes and
// the market snapshot API backing the market panel UI.
// Spec: docs/superpowers/specs/2026-08-06-grand-strategy-playable.md
//
// All helpers run complete (never broken-out) queries or break+Close, and
// apply structural mutations strictly after every query loop has finished.

import (
	"errors"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// DefaultTaxRatePercent is the implicit rate of the baseline TaxationSystem
// formula: a country without a TaxPolicyComponent (or with Rate == this
// value) collects exactly what the base system always collected (1.0x).
const DefaultTaxRatePercent uint8 = 10

// taxationInterval mirrors the TaxationSystem collection cadence (every 100
// ticks); TaxPolicySystem uses it to skip snapshot work on idle ticks.
const taxationInterval uint64 = 100

// TradeRouteInterval is the trade-route transfer window in ticks.
const TradeRouteInterval uint64 = 500

// DefaultTradeRouteVolume is the per-good volume of a fresh trade route.
const DefaultTradeRouteVolume uint16 = 10

// Trade profit distribution. Both are exact binary fractions so route income
// stays bit-identical across runs (determinism law).
const (
	tradeCityIncomeShare float32 = 0.25  // Each endpoint city's cut of profit
	tradeCountryCutShare float32 = 0.125 // Each endpoint's sovereign cut
)

// Sentinel errors returned by the economy actions.
var (
	ErrNoSuchCountry = errors.New("no capital found for country")
	ErrNoSuchCity    = errors.New("no such city")
	ErrSameCity      = errors.New("trade route endpoints must differ")
	ErrRouteExists   = errors.New("trade route already exists")
)

// SetTaxRate sets the national tax rate (percent, clamped to
// 0..components.MaxTaxRatePercent) on the capital of the given country,
// adding the TaxPolicyComponent if missing. Returns ErrNoSuchCountry when the
// country has no living capital.
func SetTaxRate(world *ecs.World, countryID uint32, rate uint8) error {
	capital, ok := FindCapitalOf(world, countryID)
	if !ok {
		return ErrNoSuchCountry
	}
	if rate > components.MaxTaxRatePercent {
		rate = components.MaxTaxRatePercent
	}
	taxID := ecs.ComponentID[components.TaxPolicyComponent](world)
	if !world.Has(capital, taxID) {
		world.Add(capital, taxID) // arche zero-initializes added components
	}
	policy := (*components.TaxPolicyComponent)(world.Get(capital, taxID))
	policy.Rate = rate
	return nil
}

// GetTaxRate returns the country's tax rate percent, falling back to
// DefaultTaxRatePercent when the country has no capital or no policy set.
func GetTaxRate(world *ecs.World, countryID uint32) uint8 {
	capital, ok := FindCapitalOf(world, countryID)
	if !ok {
		return DefaultTaxRatePercent
	}
	taxID := ecs.ComponentID[components.TaxPolicyComponent](world)
	if !world.Has(capital, taxID) {
		return DefaultTaxRatePercent
	}
	policy := (*components.TaxPolicyComponent)(world.Get(capital, taxID))
	return policy.Rate
}

// CountryTaxMultiplier converts a country's tax-rate percent into the
// multiplier applied to the baseline TaxationSystem collection.
// Rate == DefaultTaxRatePercent -> 1.0 (unchanged base behavior).
func CountryTaxMultiplier(world *ecs.World, countryID uint32) float32 {
	return float32(GetTaxRate(world, countryID)) / float32(DefaultTaxRatePercent)
}

// villageTaxSnap captures a village treasury right before a base collection
// so the wrapper can measure exactly what the base system took.
type villageTaxSnap struct {
	treasury  *components.TreasuryComponent
	pre       float32
	countryID uint32
}

// TaxPolicySystem wraps the base TaxationSystem and makes its collections
// respect each country's TaxPolicyComponent. It REPLACES the bare
// TaxationSystem registration: register NewTaxPolicySystem(NewTaxationSystem(
// world, hooks)) instead. On collection ticks it snapshots eligible village
// treasuries, runs the base collection, then rescales the measured per-village
// take by the country multiplier: extra tax is drawn from the village (capped
// by its remaining wealth); rebates flow back out of the country treasury
// (capped by the amount just collected, so no treasury goes negative).
// Evasion (loyalty < corruption) collects 0 and is therefore never rescaled.
type TaxPolicySystem struct {
	base  *TaxationSystem
	snaps []villageTaxSnap // Reused scratch buffer
}

// NewTaxPolicySystem wraps an existing TaxationSystem with tax-rate policy.
func NewTaxPolicySystem(base *TaxationSystem) *TaxPolicySystem {
	return &TaxPolicySystem{
		base:  base,
		snaps: make([]villageTaxSnap, 0, 64),
	}
}

func (s *TaxPolicySystem) Update(world *ecs.World) {
	// Idle ticks: the base system only collects every taxationInterval ticks
	// (it increments tickStamp first, then tests %interval), so skip the
	// snapshot bookkeeping when this Update cannot be a collection tick.
	if (s.base.tickStamp+1)%taxationInterval != 0 {
		s.base.Update(world)
		return
	}

	villageID := ecs.ComponentID[components.Village](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	taxID := ecs.ComponentID[components.TaxPolicyComponent](world)

	// 1. Snapshot every village the base system may tax (same query shape).
	// Held pointers stay valid across base.Update: TaxationSystem performs no
	// structural mutations (no Add/Remove/RemoveEntity).
	s.snaps = s.snaps[:0]
	villageQuery := world.Query(filter.All(villageID, affilID, marketID, treasuryID, loyaltyID))
	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(affilID))
		treasury := (*components.TreasuryComponent)(villageQuery.Get(treasuryID))
		s.snaps = append(s.snaps, villageTaxSnap{
			treasury:  treasury,
			pre:       treasury.Wealth,
			countryID: affil.CountryID,
		})
	}

	// 2. Run the baseline collection.
	s.base.Update(world)

	// 3. Build per-country multiplier + treasury lookup (read-only map use;
	// iteration below runs over the deterministic snapshot slice).
	type countryTax struct {
		treasury *components.TreasuryComponent
		mult     float32
	}
	countries := make(map[uint32]countryTax)
	capitalQuery := world.Query(filter.All(countryCompID, capitalID, affilID, treasuryID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(affilID))
		treasury := (*components.TreasuryComponent)(capitalQuery.Get(treasuryID))
		rate := DefaultTaxRatePercent
		if capitalQuery.Has(taxID) {
			policy := (*components.TaxPolicyComponent)(world.Get(capitalQuery.Entity(), taxID))
			rate = policy.Rate
		}
		countries[affil.CountryID] = countryTax{
			treasury: treasury,
			mult:     float32(rate) / float32(DefaultTaxRatePercent),
		}
	}

	// 4. Rescale each measured collection by the country multiplier.
	for i := range s.snaps {
		snap := &s.snaps[i]
		country, ok := countries[snap.countryID]
		if !ok || country.mult == 1.0 {
			continue
		}
		collected := snap.pre - snap.treasury.Wealth
		if collected <= 0 {
			continue // Evasion or nothing collectable: rate does not apply.
		}
		delta := collected*country.mult - collected
		if delta > 0 {
			// Higher rate: take the surplus, capped by remaining village wealth.
			if delta > snap.treasury.Wealth {
				delta = snap.treasury.Wealth
			}
			snap.treasury.Wealth -= delta
			country.treasury.Wealth += delta
		} else {
			// Lower rate: rebate out of the country treasury. The rebate is
			// <= collected, which the country received this very tick, but cap
			// defensively so the treasury can never go negative.
			rebate := -delta
			if rebate > country.treasury.Wealth {
				rebate = country.treasury.Wealth
			}
			country.treasury.Wealth -= rebate
			snap.treasury.Wealth += rebate
		}
	}
}

// cityExists reports whether a living (non-ruined) village with the given
// CityID (uint32(Identity.ID), the CityBinder convention) exists.
func cityExists(world *ecs.World, cityID uint32) bool {
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	found := false
	query := world.Query(filter.All(villageID, identID))
	for query.Next() {
		if found || query.Has(ruinID) {
			continue
		}
		ident := (*components.Identity)(query.Get(identID))
		if uint32(ident.ID) == cityID {
			found = true
		}
	}
	return found
}

// EstablishTradeRoute creates a trade-route entity linking two living cities.
// The route is undirected and deduplicated in both orders. Returns the new
// route entity, or an error (ErrSameCity / ErrNoSuchCity / ErrRouteExists).
func EstablishTradeRoute(world *ecs.World, cityA, cityB uint32) (ecs.Entity, error) {
	if cityA == cityB {
		return ecs.Entity{}, ErrSameCity
	}
	if cityA == 0 || cityB == 0 || !cityExists(world, cityA) || !cityExists(world, cityB) {
		return ecs.Entity{}, ErrNoSuchCity
	}

	routeID := ecs.ComponentID[components.TradeRouteComponent](world)

	exists := false
	query := world.Query(filter.All(routeID))
	for query.Next() {
		route := (*components.TradeRouteComponent)(query.Get(routeID))
		if (route.FromCity == cityA && route.ToCity == cityB) ||
			(route.FromCity == cityB && route.ToCity == cityA) {
			exists = true
		}
	}
	if exists {
		return ecs.Entity{}, ErrRouteExists
	}

	// Structural creation strictly after all queries have been exhausted.
	ent := world.NewEntity(routeID)
	route := (*components.TradeRouteComponent)(world.Get(ent, routeID))
	route.FromCity = cityA
	route.ToCity = cityB
	route.Volume = DefaultTradeRouteVolume
	return ent, nil
}

// TradeRouteInfo is a flat copy of one route for UI listings.
type TradeRouteInfo struct {
	FromCity uint32
	ToCity   uint32
	Volume   uint16
}

// ListTradeRoutes returns all trade routes sorted by (FromCity, ToCity) for
// deterministic UI ordering.
func ListTradeRoutes(world *ecs.World) []TradeRouteInfo {
	routeID := ecs.ComponentID[components.TradeRouteComponent](world)

	routes := make([]TradeRouteInfo, 0, 16)
	query := world.Query(filter.All(routeID))
	for query.Next() {
		route := (*components.TradeRouteComponent)(query.Get(routeID))
		routes = append(routes, TradeRouteInfo{
			FromCity: route.FromCity,
			ToCity:   route.ToCity,
			Volume:   route.Volume,
		})
	}

	slices.SortFunc(routes, func(a, b TradeRouteInfo) int {
		if a.FromCity != b.FromCity {
			if a.FromCity < b.FromCity {
				return -1
			}
			return 1
		}
		if a.ToCity < b.ToCity {
			return -1
		}
		if a.ToCity > b.ToCity {
			return 1
		}
		return 0
	})
	return routes
}

// tradeCityData caches one living city's economy pointers for a transfer
// window. Pointers stay valid until the deferred route removals at the end of
// the update (no structural mutation happens before then).
type tradeCityData struct {
	storage   *components.StorageComponent
	market    *components.MarketComponent
	treasury  *components.TreasuryComponent
	countryID uint32
}

// tradeGoodsOrder fixes the per-window good iteration order (determinism law).
var tradeGoodsOrder = [4]uint8{
	components.ItemWood,
	components.ItemStone,
	components.ItemIron,
	components.ItemFood,
}

// TradeRouteSystem executes trade-route transfers every TradeRouteInterval
// ticks: for each route and each good, up to Volume units move from the
// cheaper endpoint toward the pricier one (buy low, sell high per PriceOf).
// The price gradient generates profit: each endpoint city treasury earns
// tradeCityIncomeShare of it and each endpoint's country treasury earns
// tradeCountryCutShare. Routes whose endpoint city died (despawned or ruined)
// are removed.
type TradeRouteSystem struct {
	tickStamp uint64
	toRemove  []ecs.Entity // Reused scratch buffer for deferred removals
}

// NewTradeRouteSystem initializes the trade-route transfer loop.
func NewTradeRouteSystem() *TradeRouteSystem {
	return &TradeRouteSystem{
		toRemove: make([]ecs.Entity, 0, 16),
	}
}

func (s *TradeRouteSystem) Update(world *ecs.World) {
	s.tickStamp++
	if s.tickStamp%TradeRouteInterval != 0 {
		return
	}

	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	routeID := ecs.ComponentID[components.TradeRouteComponent](world)

	// 1. Index living cities by CityID (lookup-only map; route iteration
	// below follows the deterministic query order).
	cities := make(map[uint32]tradeCityData)
	cityQuery := world.Query(filter.All(villageID, identID, affilID, storageID, marketID, treasuryID))
	for cityQuery.Next() {
		if cityQuery.Has(ruinID) {
			continue // A ruined city no longer trades.
		}
		ident := (*components.Identity)(cityQuery.Get(identID))
		affil := (*components.Affiliation)(cityQuery.Get(affilID))
		cities[uint32(ident.ID)] = tradeCityData{
			storage:   (*components.StorageComponent)(cityQuery.Get(storageID)),
			market:    (*components.MarketComponent)(cityQuery.Get(marketID)),
			treasury:  (*components.TreasuryComponent)(cityQuery.Get(treasuryID)),
			countryID: affil.CountryID,
		}
	}

	// 2. Index country treasuries for the sovereign cut.
	countryTreasuries := make(map[uint32]*components.TreasuryComponent)
	capitalQuery := world.Query(filter.All(countryCompID, capitalID, affilID, treasuryID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(affilID))
		countryTreasuries[affil.CountryID] = (*components.TreasuryComponent)(capitalQuery.Get(treasuryID))
	}

	// 3. Execute transfers; collect dead routes for deferred removal.
	s.toRemove = s.toRemove[:0]
	routeQuery := world.Query(filter.All(routeID))
	for routeQuery.Next() {
		route := (*components.TradeRouteComponent)(routeQuery.Get(routeID))
		from, okFrom := cities[route.FromCity]
		to, okTo := cities[route.ToCity]
		if !okFrom || !okTo {
			s.toRemove = append(s.toRemove, routeQuery.Entity())
			continue
		}

		for _, item := range tradeGoodsOrder {
			priceFrom := PriceOf(from.market, item)
			priceTo := PriceOf(to.market, item)
			if priceFrom == priceTo {
				continue // No gradient, no trade.
			}

			// Goods flow from the cheaper city toward the pricier one.
			src, dst := from, to
			low, high := priceFrom, priceTo
			if priceFrom > priceTo {
				src, dst = to, from
				low, high = priceTo, priceFrom
			}

			srcStock := StorageOf(src.storage, item)
			dstStock := StorageOf(dst.storage, item)
			qty := uint32(route.Volume)
			if *srcStock < qty {
				qty = *srcStock
			}
			if qty == 0 {
				continue
			}

			*srcStock -= qty
			*dstStock += qty

			// Route income from the price gradient.
			profit := (high - low) * float32(qty)
			from.treasury.Wealth += profit * tradeCityIncomeShare
			to.treasury.Wealth += profit * tradeCityIncomeShare
			if ct, ok := countryTreasuries[from.countryID]; ok && from.countryID != 0 {
				ct.Wealth += profit * tradeCountryCutShare
			}
			if ct, ok := countryTreasuries[to.countryID]; ok && to.countryID != 0 {
				ct.Wealth += profit * tradeCountryCutShare
			}
		}
	}

	// 4. Deferred structural mutation: drop routes whose endpoint died.
	for _, ent := range s.toRemove {
		world.RemoveEntity(ent)
	}
}

// CityPrices is one row of the market panel: current local prices of every
// good PriceOf supports for one city.
type CityPrices struct {
	CityID uint32
	Food   float32
	Wood   float32
	Stone  float32
	Iron   float32
}

// MarketSnapshot returns the current market prices of every living
// (non-ruined) city, sorted by CityID for deterministic UI ordering.
func MarketSnapshot(world *ecs.World) []CityPrices {
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	rows := make([]CityPrices, 0, 16)
	query := world.Query(filter.All(villageID, identID, marketID))
	for query.Next() {
		if query.Has(ruinID) {
			continue
		}
		ident := (*components.Identity)(query.Get(identID))
		market := (*components.MarketComponent)(query.Get(marketID))
		rows = append(rows, CityPrices{
			CityID: uint32(ident.ID),
			Food:   market.FoodPrice,
			Wood:   market.WoodPrice,
			Stone:  market.StonePrice,
			Iron:   market.IronPrice,
		})
	}

	slices.SortFunc(rows, func(a, b CityPrices) int {
		if a.CityID < b.CityID {
			return -1
		}
		if a.CityID > b.CityID {
			return 1
		}
		return 0
	})
	return rows
}
