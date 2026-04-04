package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 53 - The Black Market Smuggling Engine
// Bridges Governance (Contraband Laws) directly to Macroeconomics (Market Logic).
// When an item is flagged as illegal inside a jurisdiction, the BlackMarketSystem dynamically
// intercepts and artificially spikes its local price by 5x (the Risk Premium).
// This naturally incentivizes Caravans and Risk-Takers to smuggle the contraband into the jurisdiction,
// creating systemic economic loops.

type blackMarketJurData struct {
	X             float32
	Y             float32
	RadiusSquared float32
	Contraband    uint32
}

type BlackMarketSystem struct {
	world         *ecs.World
	jurisdictions []blackMarketJurData
}

func NewBlackMarketSystem(world *ecs.World) *BlackMarketSystem {
	return &BlackMarketSystem{
		world:         world,
		jurisdictions: make([]blackMarketJurData, 0, 100),
	}
}

func (s *BlackMarketSystem) Update(world *ecs.World) {
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	contraID := ecs.ComponentID[components.ContrabandComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	// Step 1: Pre-cache all Jurisdictions that have Contraband laws to avoid nested queries
	s.jurisdictions = s.jurisdictions[:0]
	jurQuery := world.Query(ecs.All(jurID, contraID, posID))

	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))
		contra := (*components.ContrabandComponent)(jurQuery.Get(contraID))
		pos := (*components.Position)(jurQuery.Get(posID))

		// Only cache if there is actually contraband
		if contra.Contraband > 0 {
			s.jurisdictions = append(s.jurisdictions, blackMarketJurData{
				X:             pos.X,
				Y:             pos.Y,
				RadiusSquared: jur.RadiusSquared,
				Contraband:    contra.Contraband,
			})
		}
	}

	if len(s.jurisdictions) == 0 {
		return // No active contraband laws
	}

	// Step 2: Iterate over Villages and apply the Risk Premium modifier
	villageID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)

	villageQuery := world.Query(ecs.All(villageID, posID, marketID))

	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(posID))
		market := (*components.MarketComponent)(villageQuery.Get(marketID))

		// Check if Village falls within any active contraband jurisdiction
		for i := 0; i < len(s.jurisdictions); i++ {
			j := &s.jurisdictions[i]
			dx := pos.X - j.X
			dy := pos.Y - j.Y
			distSq := (dx * dx) + (dy * dy)

			if distSq <= j.RadiusSquared {
				// Apply 5x Risk Premium to any flagged item
				if (j.Contraband & (1 << components.ItemWood)) != 0 {
					market.WoodPrice *= 5.0
				}
				if (j.Contraband & (1 << components.ItemStone)) != 0 {
					market.StonePrice *= 5.0
				}
				if (j.Contraband & (1 << components.ItemIron)) != 0 {
					market.IronPrice *= 5.0
				}
				if (j.Contraband & (1 << components.ItemFood)) != 0 {
					market.FoodPrice *= 5.0
				}
				break // Assume primary jurisdiction dictates laws
			}
		}
	}
}
