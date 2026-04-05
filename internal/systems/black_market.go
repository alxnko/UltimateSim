package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 53 - The Black Market Smuggling Engine
// BlackMarketSystem evaluates Villages inside Jurisdictions with Contraband laws.
// It applies a massive 5.0x Risk Premium to the local MarketComponent prices of any illegal goods,
// directly incentivizing NPCs to smuggle and pursue illegal careers to capture the artificial margin.

type BlackMarketSystem struct {
	world *ecs.World

	// Component IDs
	jurID      ecs.ID
	contraID   ecs.ID
	posID      ecs.ID
	villageID  ecs.ID
	marketID   ecs.ID
}

// NewBlackMarketSystem creates a new BlackMarketSystem.
func NewBlackMarketSystem(world *ecs.World) *BlackMarketSystem {
	return &BlackMarketSystem{
		world:     world,
		jurID:     ecs.ComponentID[components.JurisdictionComponent](world),
		contraID:  ecs.ComponentID[components.ContrabandComponent](world),
		posID:     ecs.ComponentID[components.Position](world),
		villageID: ecs.ComponentID[components.Village](world),
		marketID:  ecs.ComponentID[components.MarketComponent](world),
	}
}

// Update evaluates jurisdictions and applies risk premiums to local markets.
func (s *BlackMarketSystem) Update() {
	// 1. Extract all active Jurisdictions with Contraband into a flat array to prevent nested queries.
	type jurNodeData struct {
		X, Y          float32
		RadiusSquared float32
		Contraband    uint32
	}

	var activeJurisdictions []jurNodeData

	jurQuery := s.world.Query(filter.All(s.jurID, s.contraID, s.posID))
	for jurQuery.Next() {
		pos := (*components.Position)(jurQuery.Get(s.posID))
		jur := (*components.JurisdictionComponent)(jurQuery.Get(s.jurID))
		contra := (*components.ContrabandComponent)(jurQuery.Get(s.contraID))

		if contra.Contraband > 0 {
			activeJurisdictions = append(activeJurisdictions, jurNodeData{
				X:             pos.X,
				Y:             pos.Y,
				RadiusSquared: jur.RadiusSquared,
				Contraband:    contra.Contraband,
			})
		}
	}

	if len(activeJurisdictions) == 0 {
		return // No active contraband zones
	}

	// 2. Iterate all Villages with Markets and evaluate if they fall within any Contraband zones.
	villageQuery := s.world.Query(filter.All(s.villageID, s.marketID, s.posID))
	for villageQuery.Next() {
		pos := (*components.Position)(villageQuery.Get(s.posID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))

		// Check overlap with active jurisdictions
		for _, jurNode := range activeJurisdictions {
			dx := pos.X - jurNode.X
			dy := pos.Y - jurNode.Y
			distSq := dx*dx + dy*dy

			if distSq <= jurNode.RadiusSquared {
				// Apply Risk Premium for each contraband item type
				const riskPremium float32 = 5.0

				if (jurNode.Contraband & (1 << components.ItemWood)) != 0 {
					market.WoodPrice *= riskPremium
				}
				if (jurNode.Contraband & (1 << components.ItemStone)) != 0 {
					market.StonePrice *= riskPremium
				}
				if (jurNode.Contraband & (1 << components.ItemIron)) != 0 {
					market.IronPrice *= riskPremium
				}
				if (jurNode.Contraband & (1 << components.ItemFood)) != 0 {
					market.FoodPrice *= riskPremium
				}
			}
		}
	}
}
