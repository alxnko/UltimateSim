package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 13.1: Local Price Discovery (Market Logic)
// Each city establishes dynamic local prices mathematically determined by its
// current StorageComponent metrics versus PopulationComponent (Demand).

type PriceDiscoverySystem struct{}

func NewPriceDiscoverySystem() *PriceDiscoverySystem {
	return &PriceDiscoverySystem{}
}

func (s *PriceDiscoverySystem) Update(world *ecs.World) {
	villageID := ecs.ComponentID[components.Village](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)

	filter := ecs.All(villageID, storageID, popID, marketID)
	query := world.Query(filter)

	for query.Next() {
		storage := (*components.StorageComponent)(query.Get(storageID))
		pop := (*components.PopulationComponent)(query.Get(popID))
		market := (*components.MarketComponent)(query.Get(marketID))

		// Price logic bounds: BasePrice * (Demand / (Supply + 1.0))
		// We add 1.0 to Supply to prevent divide-by-zero panics.

		// Food demand scales heavily with population
		foodDemand := float32(pop.Count) * 10.0
		market.FoodPrice = 1.0 * (foodDemand / (float32(storage.Food) + 1.0))

		// Wood demand scales moderately
		woodDemand := float32(pop.Count) * 5.0
		market.WoodPrice = 1.0 * (woodDemand / (float32(storage.Wood) + 1.0))

		// Stone demand
		stoneDemand := float32(pop.Count) * 2.0
		market.StonePrice = 1.0 * (stoneDemand / (float32(storage.Stone) + 1.0))

		// Iron demand
		ironDemand := float32(pop.Count) * 1.0
		market.IronPrice = 1.0 * (ironDemand / (float32(storage.Iron) + 1.0))

		// Phase 15.4: Organic Inflation via Debasement
		// PriceDiscoverySystem does not know about physical coins locally,
		// but InflationSystem forcibly modifies MarketComponent afterwards in the pipeline.
	}
}
