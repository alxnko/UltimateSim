package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 20.3: Traumatic Traditions
// TraumaticTraditionsSystem monitors Jurisdictions for high Trauma values (mass death via plague/starvation).
// If a threshold is met, survivors in that region develop `BeliefXenophobia`.
// The Jurisdiction's Trauma decays slowly over time.

type TraumaticTraditionsSystem struct {
	tickCounter uint64

	// Component IDs
	jurID   ecs.ID
	posID   ecs.ID
	affID   ecs.ID
	belID   ecs.ID
}

func NewTraumaticTraditionsSystem(world *ecs.World) *TraumaticTraditionsSystem {
	return &TraumaticTraditionsSystem{
		jurID:   ecs.ComponentID[components.JurisdictionComponent](world),
		posID:   ecs.ComponentID[components.Position](world),
		affID:   ecs.ComponentID[components.Affiliation](world),
		belID:   ecs.ComponentID[components.BeliefComponent](world),
	}
}

func (s *TraumaticTraditionsSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process every 50 ticks to spread CPU load and act as a decay cycle
	if s.tickCounter%50 != 0 {
		return
	}

	jurQuery := world.Query(ecs.All(s.jurID, s.posID))

	type jurData struct {
		entity ecs.Entity
		comp   *components.JurisdictionComponent
		x      float32
		y      float32
	}

	// Collect traumatic jurisdictions
	activeTraumas := make([]jurData, 0, 10)

	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(s.jurID))
		pos := (*components.Position)(jurQuery.Get(s.posID))

		// Trauma threshold for "Massive Societal Trauma"
		if jur.Trauma > 10 {
			activeTraumas = append(activeTraumas, jurData{
				entity: jurQuery.Entity(),
				comp:   jur,
				x:      pos.X,
				y:      pos.Y,
			})
		}

		// Decay trauma
		if jur.Trauma > 0 {
			jur.Trauma--
		}
	}

	if len(activeTraumas) == 0 {
		return
	}

	// Infect survivors with Xenophobia belief
	npcQuery := world.Query(ecs.All(s.posID, s.belID, s.affID))

	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		bel := (*components.BeliefComponent)(npcQuery.Get(s.belID))

		for i := 0; i < len(activeTraumas); i++ {
			j := &activeTraumas[i]
			dx := pos.X - j.x
			dy := pos.Y - j.y

			// If inside traumatized jurisdiction
			if dx*dx+dy*dy <= j.comp.RadiusSquared {
				hasXenophobia := false
				for _, b := range bel.Beliefs {
					if b.BeliefID == components.BeliefXenophobia {
						hasXenophobia = true
						break
					}
				}

				if !hasXenophobia {
					bel.Beliefs = append(bel.Beliefs, components.Belief{
						BeliefID: components.BeliefXenophobia,
						Weight:   100, // Deeply ingrained trauma
					})
				}
				break // Only process infection once per NPC
			}
		}
	}
}