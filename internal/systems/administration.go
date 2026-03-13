package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.3: Profit-Driven Unification
// AdministrationSystem evaluates neighboring cities for economic synergies (Profit Gain > Loss).
// If a shared currency via Union would increase MarketComponent trade volume (modeled by reducing a >15% price disparity),
// the city initiates a "Diplomatic Hook" via the SparseHookGraph to propose a Union.

const (
	AdministrationTickRate  = 1000 // Perform analysis every 1000 ticks
	MaxDiplomaticRange      = 100  // Maximum distance to consider a merger
	PriceDisparityThreshold = 0.15 // >15% disparity in prices triggers a hook
)

type AdministrationSystem struct {
	world     *ecs.World
	hooks     *engine.SparseHookGraph
	tickStamp uint64

	// Pre-allocated DOD slices
	cities []adminCityData
}

type adminCityData struct {
	id     uint64
	pos    *components.Position
	market *components.MarketComponent
	affil  *components.Affiliation
}

// NewAdministrationSystem initializes the administrative cost-benefit analysis.
func NewAdministrationSystem(world *ecs.World, hooks *engine.SparseHookGraph) *AdministrationSystem {
	return &AdministrationSystem{
		world:     world,
		hooks:     hooks,
		tickStamp: 0,
		cities:    make([]adminCityData, 0, 500),
	}
}

// Update executes the core loop.
func (s *AdministrationSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Only run deep analysis periodically to save CPU
	if s.tickStamp%AdministrationTickRate != 0 {
		return
	}

	identID := ecs.ComponentID[components.Identity](world)
	posID := ecs.ComponentID[components.Position](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	villageID := ecs.ComponentID[components.Village](world)

	query := world.Query(filter.All(villageID, identID, posID, marketID, affilID))

	s.cities = s.cities[:0] // Retain capacity, clear bounds for DOD reuse

	// Extract all cities strictly sequentially
	for query.Next() {
		ident := (*components.Identity)(query.Get(identID))
		pos := (*components.Position)(query.Get(posID))
		market := (*components.MarketComponent)(query.Get(marketID))
		affil := (*components.Affiliation)(query.Get(affilID))

		s.cities = append(s.cities, adminCityData{
			id:     ident.ID,
			pos:    pos,
			market: market,
			affil:  affil,
		})
	}

	// O(N^2) evaluation over flat arrays (fastest for CPU cache)
	for i := 0; i < len(s.cities); i++ {
		cityA := s.cities[i]

		for j := i + 1; j < len(s.cities); j++ {
			cityB := s.cities[j]

			// If already in same Country/Union, no need for Diplomatic Hook
			if cityA.affil.CountryID != 0 && cityA.affil.CountryID == cityB.affil.CountryID {
				continue
			}

			// Distance Check (Integer math to preserve determinism & DOD speed)
			dx := cityA.pos.X - cityB.pos.X
			dy := cityA.pos.Y - cityB.pos.Y
			distSq := dx*dx + dy*dy

			if distSq > MaxDiplomaticRange*MaxDiplomaticRange {
				continue
			}

			// Calculate Total Price Volume/Disparity
			sumA := cityA.market.FoodPrice + cityA.market.WoodPrice + cityA.market.StonePrice + cityA.market.IronPrice
			sumB := cityB.market.FoodPrice + cityB.market.WoodPrice + cityB.market.StonePrice + cityB.market.IronPrice

			// Prevent divide by zero (baseline should never technically be 0, but DOD safety)
			if sumA < 1.0 {
				sumA = 1.0
			}
			if sumB < 1.0 {
				sumB = 1.0
			}

			// Calculate variance ratio
			var ratio float32
			if sumA > sumB {
				ratio = (sumA - sumB) / sumB
			} else {
				ratio = (sumB - sumA) / sumA
			}

			// Profit Gain Check: > 15% discrepancy indicates high friction and large potential gain
			if ratio > PriceDisparityThreshold {
				// Emergent Merger Hook! Add Diplomatic points.
				s.hooks.AddHook(cityA.id, cityB.id, 1)
				s.hooks.AddHook(cityB.id, cityA.id, 1) // Reciprocal for mutual agreement
			}
		}
	}
}
