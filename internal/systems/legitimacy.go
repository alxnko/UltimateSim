package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 35.1: Sovereign Legitimacy Engine
// Calculates the legitimacy score of rulers based on physical wealth, corruption friction,
// and social support (Hooks).

const (
	LegitimacyTickRate = 50 // Execute legitimacy calculations every 50 ticks
)

type LegitimacySystem struct {
	hooks       *engine.SparseHookGraph
	tickCounter uint64

	// Pre-cached component IDs
	identID  ecs.ID
	legitID  ecs.ID
	treasID  ecs.ID
	jurID    ecs.ID
	capID    ecs.ID
	equipID  ecs.ID
}

func NewLegitimacySystem(world *ecs.World, hooks *engine.SparseHookGraph) *LegitimacySystem {
	return &LegitimacySystem{
		hooks:       hooks,
		tickCounter: 0,
		identID:     ecs.ComponentID[components.Identity](world),
		legitID:     ecs.ComponentID[components.LegitimacyComponent](world),
		treasID:     ecs.ComponentID[components.TreasuryComponent](world),
		jurID:       ecs.ComponentID[components.JurisdictionComponent](world),
		capID:       ecs.ComponentID[components.CapitalComponent](world),
		equipID:     ecs.ComponentID[components.EquipmentComponent](world),
	}
}

func (s *LegitimacySystem) Update(world *ecs.World) {
	s.tickCounter++

	if s.tickCounter%LegitimacyTickRate != 0 {
		return
	}

	// Iterate over all governing entities that have a Legitimacy Score
	query := world.Query(ecs.All(s.identID, s.legitID, s.capID))

	for query.Next() {
		ident := (*components.Identity)(query.Get(s.identID))
		legit := (*components.LegitimacyComponent)(query.Get(s.legitID))

		// Base Legitimacy is 50.
		var newScore float32 = 50.0

		// 1. Economic Wealth Bonus (Treasury)
		if world.Has(query.Entity(), s.treasID) {
			treasury := (*components.TreasuryComponent)(query.Get(s.treasID))
			// Grant 1 point of legitimacy per 1000 wealth, capped at 25 bonus
			wealthBonus := treasury.Wealth / 1000.0
			if wealthBonus > 25.0 {
				wealthBonus = 25.0
			}
			newScore += wealthBonus
		}

		// 2. State Capacity Friction (Corruption)
		if world.Has(query.Entity(), s.jurID) {
			jur := (*components.JurisdictionComponent)(query.Get(s.jurID))
			// Subtract 2 points per corruption incident
			corruptionPenalty := float32(jur.Corruption) * 2.0
			newScore -= corruptionPenalty
		}

		// 4. Public Sentiment (Hooks pointing at the ruler)
		if s.hooks != nil {
			incomingHooks := s.hooks.GetAllIncomingHooks(ident.ID)
			var totalSentiment float32 = 0.0

			for _, points := range incomingHooks {
				// Convert to scale. High negative points heavily drain legitimacy.
				// E.g., a massive blood feud (-100) drain legitimacy heavily.
				totalSentiment += float32(points) / 10.0 // Every 10 hook points = 1 legitimacy
			}

			// Cap extreme sentiment swings to avoid immediate unbounded collapse from 1 person
			if totalSentiment > 50.0 {
				totalSentiment = 50.0
			} else if totalSentiment < -50.0 {
				totalSentiment = -50.0
			}

			newScore += totalSentiment
		}

		// Phase 52.1: Auras of Legitimacy (Artifact Equipment)
		// 3. Aura of Legitimacy
		if world.Has(query.Entity(), s.equipID) {
			equip := (*components.EquipmentComponent)(world.Get(query.Entity(), s.equipID))
			if equip.Equipped && equip.Weapon.Prestige > 0 {
				// The artifact's prestige mathematically scales to provide massive state legitimacy
				// e.g., 500 Prestige Sword grants +50 Legitimacy
				artifactBonus := float32(equip.Weapon.Prestige) / 10.0
				newScore += artifactBonus
			}
		}

		// Mathematical Bounds Check
		if newScore > 100.0 {
			newScore = 100.0
		} else if newScore < 0.0 {
			newScore = 0.0
		}

		// Assign parsed float back to DOD uint32 bounds
		legit.Score = uint32(newScore)
	}
}
