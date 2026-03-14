package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 15.4: Organic Inflation via Debasement
// InflationSystem measures the average Debasement of physical coins circulating
// at specific physical locations and directly inflates MarketComponent prices.
type InflationSystem struct {
	world *ecs.World

	// Component IDs
	coinTagID  ecs.ID
	currencyID ecs.ID
	posID      ecs.ID
	villageID  ecs.ID
	marketID   ecs.ID

	tickStamp uint64
}

// NewInflationSystem creates a new InflationSystem.
func NewInflationSystem(world *ecs.World) *InflationSystem {
	return &InflationSystem{
		world:      world,
		coinTagID:  ecs.ComponentID[components.CoinEntity](world),
		currencyID: ecs.ComponentID[components.CurrencyComponent](world),
		posID:      ecs.ComponentID[components.Position](world),
		villageID:  ecs.ComponentID[components.Village](world),
		marketID:   ecs.ComponentID[components.MarketComponent](world),
	}
}

// Update calculates physical coin debasement flowing through villages and spikes prices organically.
func (s *InflationSystem) Update() {
	s.tickStamp++

	// 1. Map all circulating physical coins by their integer Grid coordinates
	type loc struct {
		X, Y int
	}

	type debasementAgg struct {
		TotalDebasement float32
		Count           uint32
	}

	localDebasement := make(map[loc]*debasementAgg)

	coinQuery := s.world.Query(filter.All(s.coinTagID, s.currencyID, s.posID))
	for coinQuery.Next() {
		pos := (*components.Position)(coinQuery.Get(s.posID))
		curr := (*components.CurrencyComponent)(coinQuery.Get(s.currencyID))

		// Only factor in coins that actually have debasement
		if curr.Debasement > 0 {
			l := loc{X: int(pos.X), Y: int(pos.Y)}
			if agg, exists := localDebasement[l]; exists {
				agg.TotalDebasement += curr.Debasement
				agg.Count++
			} else {
				localDebasement[l] = &debasementAgg{
					TotalDebasement: curr.Debasement,
					Count:           1,
				}
			}
		}
	}

	if len(localDebasement) == 0 {
		return // No debased coins circulating
	}

	// 2. Apply inflation directly to any Village/Market that overlaps these coordinates
	villageQuery := s.world.Query(filter.All(s.villageID, s.marketID, s.posID))
	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(s.posID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))

		l := loc{X: int(pos.X), Y: int(pos.Y)}
		if agg, exists := localDebasement[l]; exists {
			averageDebasement := agg.TotalDebasement / float32(agg.Count)

			// Inflation multiplier formula: 1.0 + (Average Debasement)
			// e.g. 0.5 debasement = 1.5x price multiplier.
			// Because PriceDiscoverySystem sets baseline prices every tick based on Supply/Demand,
			// multiplying them here every tick scales the baseline correctly without permanent compounding.
			multiplier := 1.0 + averageDebasement

			market.FoodPrice *= multiplier
			market.WoodPrice *= multiplier
			market.StonePrice *= multiplier
			market.IronPrice *= multiplier
		}
	}
}
