package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 50 - The Military-Industrial Complex (War Economy Engine)
// Connects Phase 29 (Resource Wars) to Phase 13 (Macroeconomics) and Phase 35 (Sovereignty).
// When WarTrackerComponent.Active is true, it drains StorageComponent.Iron.
// If depleted, it uses TreasuryComponent.Wealth to buy Iron and artificially spikes MarketComponent.IronPrice.
// Bankruptcy triggers a drop in LegitimacyComponent.Score.

type warEconomyNodeData struct {
	Entity     ecs.Entity
	WarTracker *components.WarTrackerComponent
	Storage    *components.StorageComponent
	Treasury   *components.TreasuryComponent
	Market     *components.MarketComponent
	Legitimacy *components.LegitimacyComponent
}

type WarEconomySystem struct {
	tickCounter uint64
	buffer      []warEconomyNodeData

	warID     ecs.ID
	storageID ecs.ID
	treasID   ecs.ID
	marketID  ecs.ID
	legitID   ecs.ID
}

func NewWarEconomySystem(world *ecs.World) *WarEconomySystem {
	return &WarEconomySystem{
		tickCounter: 0,
		buffer:      make([]warEconomyNodeData, 0, 100),
		warID:       ecs.ComponentID[components.WarTrackerComponent](world),
		storageID:   ecs.ComponentID[components.StorageComponent](world),
		treasID:     ecs.ComponentID[components.TreasuryComponent](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
		legitID:     ecs.ComponentID[components.LegitimacyComponent](world),
	}
}

func (s *WarEconomySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process war economy every 50 ticks to balance performance and impact
	if s.tickCounter%50 != 0 {
		return
	}

	s.buffer = s.buffer[:0]
	query := world.Query(filter.All(s.warID, s.storageID, s.treasID, s.marketID, s.legitID))

	for query.Next() {
		war := (*components.WarTrackerComponent)(query.Get(s.warID))

		if !war.Active {
			continue
		}

		storage := (*components.StorageComponent)(query.Get(s.storageID))
		treasury := (*components.TreasuryComponent)(query.Get(s.treasID))
		market := (*components.MarketComponent)(query.Get(s.marketID))
		legitimacy := (*components.LegitimacyComponent)(query.Get(s.legitID))

		s.buffer = append(s.buffer, warEconomyNodeData{
			Entity:     query.Entity(),
			WarTracker: war,
			Storage:    storage,
			Treasury:   treasury,
			Market:     market,
			Legitimacy: legitimacy,
		})
	}

	for i := 0; i < len(s.buffer); i++ {
		node := s.buffer[i]

		// 1. Drain Iron for active war effort
		if node.Storage.Iron >= 10 {
			node.Storage.Iron -= 10
		} else {
			// Iron is depleted, attempt to buy it using Treasury Wealth
			node.Storage.Iron = 0

			if node.Treasury.Wealth >= 100 {
				// Buy emergency Iron
				node.Treasury.Wealth -= 100
				node.Storage.Iron += 10
				// Artificially spike local iron prices due to massive state demand
				node.Market.IronPrice += 5.0
			} else {
				// State is bankrupt and out of iron. The war effort is collapsing.
				if node.Legitimacy.Score >= 10 {
					node.Legitimacy.Score -= 10
				} else {
					node.Legitimacy.Score = 0
				}
			}
		}
	}
}
