package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 50 - The Military-Industrial Complex
// WarEconomySystem connects Geopolitical Resource Wars (Phase 29) to
// Macroeconomics (Phase 13) and Sovereign Legitimacy (Phase 35).
// When a Country is at war, it continuously burns through its StorageComponent.Iron.
// If Iron runs out, the state artificially spikes local MarketComponent.IronPrice
// to aggressively buy Iron using its TreasuryComponent.
// If the Treasury bankrupts, the state defaults on the war and suffers massive Legitimacy loss,
// organically triggering the MilitaryRevoltSystem (Phase 27.1).

type WarEconomySystem struct {
	tickCounter uint64

	// Component IDs
	capID     ecs.ID
	warCompID ecs.ID
	storageID ecs.ID
	marketID  ecs.ID
	treasID   ecs.ID
	legitID   ecs.ID
}

// NewWarEconomySystem creates a new WarEconomySystem.
func NewWarEconomySystem(world *ecs.World) *WarEconomySystem {
	return &WarEconomySystem{
		tickCounter: 0,
		capID:       ecs.ComponentID[components.CapitalComponent](world),
		warCompID:   ecs.ComponentID[components.WarTrackerComponent](world),
		storageID:   ecs.ComponentID[components.StorageComponent](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
		treasID:     ecs.ComponentID[components.TreasuryComponent](world),
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

// Update evaluates Capital entities engaged in active wars.
func (s *WarEconomySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process war economics periodically
	if s.tickCounter%100 != 0 {
		return
	}

	filter := ecs.All(s.capID, s.warCompID, s.storageID, s.marketID, s.treasID)
	query := world.Query(filter)

	for query.Next() {
		war := (*components.WarTrackerComponent)(query.Get(s.warCompID))

		// Only process if the country is actively in a geopolitical war
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
		market := (*components.MarketComponent)(query.Get(s.marketID))
		treasury := (*components.TreasuryComponent)(query.Get(s.treasID))

		// The war effort consumes 5 units of Iron per cycle
		if storage.Iron >= 5 {
			storage.Iron -= 5
		} else {
			// State lacks Iron to arm its mob, attempt to acquire it aggressively
			storage.Iron = 0

			// State purchasing: It costs 100 Wealth to acquire 50 Iron
			if treasury.Wealth >= 100.0 {
				treasury.Wealth -= 100.0
				storage.Iron += 50

				// Systemic consequence: The state artificially spikes local Iron Price
				// This guarantees CareerChangeSystem will force local peasants to become Artisans
				market.IronPrice += 50.0
			} else {
				// State Default: The country has bankrupted itself via war
				war.Active = false

				// Evaluate Sovereign Legitimacy penalty
				if world.Has(query.Entity(), s.legitID) {
					legit := (*components.LegitimacyComponent)(query.Get(s.legitID))

					// Apply massive penalty for losing the war
					if legit.Score >= 50 {
						legit.Score -= 50
					} else {
						legit.Score = 0
					}
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
