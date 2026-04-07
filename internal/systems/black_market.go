package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 53 - Black Market Smuggling Engine
// Bridges Governance and Macroeconomics by artificially spiking the MarketComponent price
// of items flagged in a Jurisdiction's ContrabandComponent bitmask by 5x (Risk Premium)
// to natively incentivize smuggling.

const (
	BlackMarketRiskPremium float32 = 5.0
)

type contrabandJurisdictionData struct {
	Entity        ecs.Entity
	CityID        uint32
	X             float32
	Y             float32
	RadiusSquared float32
	Contraband    uint32
}

type BlackMarketSystem struct {
	jurisdictions []contrabandJurisdictionData
	tickCounter   uint64

	// Pre-cached component IDs
	jurID    ecs.ID
	posID    ecs.ID
	affID    ecs.ID
	contraID ecs.ID
	villID   ecs.ID
	markID   ecs.ID
}

func NewBlackMarketSystem(world *ecs.World) *BlackMarketSystem {
	return &BlackMarketSystem{
		jurisdictions: make([]contrabandJurisdictionData, 0, 100),
		tickCounter:   0,
		jurID:         ecs.ComponentID[components.JurisdictionComponent](world),
		posID:         ecs.ComponentID[components.Position](world),
		affID:         ecs.ComponentID[components.Affiliation](world),
		contraID:      ecs.ComponentID[components.ContrabandComponent](world),
		villID:        ecs.ComponentID[components.Village](world),
		markID:        ecs.ComponentID[components.MarketComponent](world),
	}
}

func (s *BlackMarketSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle to evaluate periodically (e.g. every 50 ticks)
	if s.tickCounter%50 != 0 {
		return
	}

	// 1. Cache Jurisdictions with Contraband limits
	jurQuery := world.Query(ecs.All(s.jurID, s.posID, s.affID, s.contraID))
	s.jurisdictions = s.jurisdictions[:0]

	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(s.jurID))
		pos := (*components.Position)(jurQuery.Get(s.posID))
		aff := (*components.Affiliation)(jurQuery.Get(s.affID))
		contra := (*components.ContrabandComponent)(jurQuery.Get(s.contraID))

		if contra.Contraband > 0 {
			s.jurisdictions = append(s.jurisdictions, contrabandJurisdictionData{
				Entity:        jurQuery.Entity(),
				CityID:        aff.CityID,
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

	// 2. Iterate Villages with Markets
	villQuery := world.Query(ecs.All(s.villID, s.posID, s.markID))

	for villQuery.Next() {
		pos := (*components.Position)(villQuery.Get(s.posID))
		market := (*components.MarketComponent)(villQuery.Get(s.markID))

		// Find if this village is inside a contraband jurisdiction
		var activeJur *contrabandJurisdictionData
		for i := 0; i < len(s.jurisdictions); i++ {
			j := &s.jurisdictions[i]
			dx := pos.X - j.X
			dy := pos.Y - j.Y
			distSq := (dx * dx) + (dy * dy)
			if distSq <= j.RadiusSquared {
				activeJur = j
				break // First hit applies
			}
		}

		if activeJur != nil {
			// Phase 53 Fix: Rather than compounding multipliers directly onto the base price,
			// we recalculate the price discovery baseline immediately by ensuring the
			// market price floors are permanently boosted based on scarcity rules.
			// This avoids infinite multiplier loops while naturally signaling career changes.
			if (activeJur.Contraband & (1 << components.ItemWood)) != 0 {
				if market.WoodPrice < 10.0*BlackMarketRiskPremium {
					market.WoodPrice = 10.0 * BlackMarketRiskPremium
				}
			}
			if (activeJur.Contraband & (1 << components.ItemStone)) != 0 {
				if market.StonePrice < 10.0*BlackMarketRiskPremium {
					market.StonePrice = 10.0 * BlackMarketRiskPremium
				}
			}
			if (activeJur.Contraband & (1 << components.ItemIron)) != 0 {
				if market.IronPrice < 10.0*BlackMarketRiskPremium {
					market.IronPrice = 10.0 * BlackMarketRiskPremium
				}
			}
			if (activeJur.Contraband & (1 << components.ItemFood)) != 0 {
				if market.FoodPrice < 5.0*BlackMarketRiskPremium {
					market.FoodPrice = 5.0 * BlackMarketRiskPremium
				}
			}
		}
	}
}
