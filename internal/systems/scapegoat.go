package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 36.1: The Scapegoat & Witch Hunt Engine
// When a Jurisdiction faces high Trauma (e.g. from Disasters or Plagues), the state
// algorithmically selects a minority BeliefID to blame for the crisis.
// This temporarily relieves the Jurisdiction's Trauma but flags the minority believers
// as criminals in the JusticeSystem, creating a pipeline to Banishment and Blood Feuds.

type adminScapegoatData struct {
	Entity        ecs.Entity
	X             float32
	Y             float32
	RadiusSquared float32
	Comp          *components.ScapegoatComponent
	Jur           *components.JurisdictionComponent
}

type ScapegoatSystem struct {
	tickCounter   uint64
	jurisdictions []adminScapegoatData
}

func NewScapegoatSystem() *ScapegoatSystem {
	return &ScapegoatSystem{
		tickCounter:   0,
		jurisdictions: make([]adminScapegoatData, 0, 10),
	}
}

func (s *ScapegoatSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run every 50 ticks to spread CPU load, offset from TraumaticTraditions
	if s.tickCounter%50 != 10 {
		return
	}

	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	scapeID := ecs.ComponentID[components.ScapegoatComponent](world)

	s.jurisdictions = s.jurisdictions[:0]

	jurQuery := world.Query(ecs.All(jurID, posID, scapeID))
	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))

		// Only trigger scapegoating if Trauma is high (>= 15) and no active scapegoat exists
		scape := (*components.ScapegoatComponent)(jurQuery.Get(scapeID))
		if jur.Trauma >= 15 && !scape.Active {
			pos := (*components.Position)(jurQuery.Get(posID))
			s.jurisdictions = append(s.jurisdictions, adminScapegoatData{
				Entity:        jurQuery.Entity(),
				X:             pos.X,
				Y:             pos.Y,
				RadiusSquared: jur.RadiusSquared,
				Comp:          scape,
				Jur:           jur,
			})
		}
	}

	if len(s.jurisdictions) == 0 {
		return
	}

	// 2. Extract NPCs to find minority beliefs
	npcID := ecs.ComponentID[components.NPC](world)
	belID := ecs.ComponentID[components.BeliefComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	esoID := ecs.ComponentID[components.EsotericMarker](world)

	npcQuery := world.Query(ecs.All(npcID, posID, belID))

	type beliefNode struct {
		x          float32
		y          float32
		beliefs    []components.Belief
		isEsoteric bool
	}

	nodes := make([]beliefNode, 0, 500)
	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(posID))
		bel := (*components.BeliefComponent)(npcQuery.Get(belID))
		entity := npcQuery.Entity()

		isEso := false
		if world.Has(entity, identID) {
			ident := (*components.Identity)(world.Get(entity, identID))
			if (ident.BaseTraits & components.TraitEsoteric) != 0 {
				isEso = true
			}
		}
		if world.Has(entity, esoID) {
			isEso = true
		}

		nodes = append(nodes, beliefNode{
			x:          pos.X,
			y:          pos.Y,
			beliefs:    bel.Beliefs,
			isEsoteric: isEso,
		})
	}

	// 3. Evaluate each traumatized jurisdiction
	for i := 0; i < len(s.jurisdictions); i++ {
		j := &s.jurisdictions[i]

		beliefCounts := make(map[uint32]int)
		totalBelievers := 0
		totalEsoteric := 0

		// Count beliefs in radius
		for k := 0; k < len(nodes); k++ {
			n := &nodes[k]
			dx := n.x - j.X
			dy := n.y - j.Y
			if dx*dx+dy*dy <= j.RadiusSquared {
				if n.isEsoteric {
					totalEsoteric++
				}
				for _, b := range n.beliefs {
					beliefCounts[b.BeliefID]++
					totalBelievers++
				}
			}
		}

		if totalBelievers == 0 && totalEsoteric == 0 {
			continue // Ghost town
		}

		// Phase 49: The Witch Hunt Engine - Esoteric Scapegoating
		if totalEsoteric > 0 {
			j.Comp.TargetEsoteric = true
			j.Comp.Active = true

			// Catharsis: The state feels temporary relief by blaming the esoteric minorities
			if j.Jur.Trauma >= 10 {
				j.Jur.Trauma -= 10
			} else {
				j.Jur.Trauma = 0
			}
			continue // Skip minority belief logic
		}

		// Find a minority belief (e.g. less than 30% of total believers, but > 0)
		var minorityID uint32 = 0
		minCount := 999999

		for bID, count := range beliefCounts {
			if float32(count)/float32(totalBelievers) < 0.30 {
				if count < minCount || (count == minCount && bID < minorityID) {
					minCount = count
					minorityID = bID
				}
			}
		}

		// If a minority belief is found, set the Scapegoat and reduce trauma
		if minorityID != 0 {
			j.Comp.TargetBeliefID = minorityID
			j.Comp.Active = true

			// Catharsis: The state feels temporary relief by blaming the minority
			if j.Jur.Trauma >= 10 {
				j.Jur.Trauma -= 10
			} else {
				j.Jur.Trauma = 0
			}
		}
	}
}
