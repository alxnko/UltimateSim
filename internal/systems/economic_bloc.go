package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.2: Strategic Unions & Pacts

// EconomicBlocSystem simulates shared access to StorageComponent stockpiles during famines.
// It detects localized famine and balances surplus food across the Economic Bloc members.
type EconomicBlocSystem struct {
	world *ecs.World

	// Pre-allocated maps to prevent inner-loop Arche queries.
	// Cache values, not component pointers — GC corruption class, see
	// banditry.go. Entity handles are stored and components re-fetched via
	// world.Get at every use, so writes stay visible across bloc iterations.
	cityMarkets  map[uint32]ecs.Entity
	cityStorages map[uint32]ecs.Entity

	// Component IDs
	unionEntityID ecs.ID
	unionCompID   ecs.ID
	villageID     ecs.ID
	affilID       ecs.ID
	marketID      ecs.ID
	storageID     ecs.ID
}

// NewEconomicBlocSystem initializes the bloc management logic
func NewEconomicBlocSystem(world *ecs.World) *EconomicBlocSystem {
	return &EconomicBlocSystem{
		world:         world,
		cityMarkets:   make(map[uint32]ecs.Entity),
		cityStorages:  make(map[uint32]ecs.Entity),
		unionEntityID: ecs.ComponentID[components.UnionEntity](world),
		unionCompID:   ecs.ComponentID[components.UnionComponent](world),
		villageID:     ecs.ComponentID[components.Village](world),
		affilID:       ecs.ComponentID[components.Affiliation](world),
		marketID:      ecs.ComponentID[components.MarketComponent](world),
		storageID:     ecs.ComponentID[components.StorageComponent](world),
	}
}

// Update balances the StorageComponent of members experiencing severe resource shortages
func (s *EconomicBlocSystem) Update() {
	// 1. Map current cities
	clear(s.cityMarkets)
	clear(s.cityStorages)

	cityQuery := s.world.Query(filter.All(s.villageID, s.affilID, s.marketID, s.storageID))
	for cityQuery.Next() {
		affil := (*components.Affiliation)(cityQuery.Get(s.affilID))
		ent := cityQuery.Entity()

		s.cityMarkets[affil.CityID] = ent
		s.cityStorages[affil.CityID] = ent
	}

	if len(s.cityMarkets) == 0 {
		return // Fast exit if no cities exist
	}

	// 2. Iterate over all Economic Blocs
	unionQuery := s.world.Query(filter.All(s.unionEntityID, s.unionCompID))
	for unionQuery.Next() {
		union := (*components.UnionComponent)(unionQuery.Get(s.unionCompID))

		// Only process Economic Blocs
		if union.UnionType != components.UnionEconomicBloc {
			continue
		}

		if len(union.MemberIDs) < 2 {
			continue
		}

		var highestFoodSurplus uint32
		var richestCityID uint32

		var highestFoodPrice float32 = 10.0 // Famine threshold
		var starvingCityID uint32
		var starvingFound bool

		// 3. Find the most starving city and the city with the largest surplus
		for _, memberID := range union.MemberIDs {
			marketEnt, hasMarket := s.cityMarkets[memberID]
			storageEnt, hasStorage := s.cityStorages[memberID]

			if hasMarket && hasStorage && s.world.Alive(marketEnt) && s.world.Alive(storageEnt) {
				market := (*components.MarketComponent)(s.world.Get(marketEnt, s.marketID))
				storage := (*components.StorageComponent)(s.world.Get(storageEnt, s.storageID))

				// Famine check derived from JobRebalancing System metrics
				// Now accurately finds the *most* starving city
				if market.FoodPrice > highestFoodPrice {
					highestFoodPrice = market.FoodPrice
					starvingCityID = memberID
					starvingFound = true
				}

				// Track highest surplus candidate
				if storage.Food > highestFoodSurplus {
					highestFoodSurplus = storage.Food
					richestCityID = memberID
				}
			}
		}

		// 4. Transfer Food from richest surplus city to the starving member
		if starvingFound && richestCityID != starvingCityID && highestFoodSurplus > 0 {
			// Transfer half of the available surplus dynamically
			transferAmount := highestFoodSurplus / 2

			// Prevent minor pointless transfers (e.g. 1 unit)
			if transferAmount > 10 {
				richestEnt := s.cityStorages[richestCityID]
				starvingEnt := s.cityStorages[starvingCityID]

				if s.world.Alive(richestEnt) && s.world.Alive(starvingEnt) {
					richestStorage := (*components.StorageComponent)(s.world.Get(richestEnt, s.storageID))
					starvingStorage := (*components.StorageComponent)(s.world.Get(starvingEnt, s.storageID))

					richestStorage.Food -= transferAmount
					starvingStorage.Food += transferAmount
				}
			}
		}
	}
}
