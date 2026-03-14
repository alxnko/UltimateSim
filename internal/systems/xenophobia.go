package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 20.3: Ideological Xenophobia (Traumatic Traditions)
// XenophobiaSystem scans for NPCs with `BeliefXenophobia`. If they interact or come into close proximity
// with an NPC that has a different `LanguageID` (Foreigner), they instantly generate a massive negative Hook
// against them, effectively acting as an automatic "Blood Feud" trigger.

type XenophobiaSystem struct {
	tickCounter uint64
	hooks       *engine.SparseHookGraph
	filter      ecs.Filter

	posID     ecs.ID
	identID   ecs.ID
	belID     ecs.ID
	cultureID ecs.ID
}

func NewXenophobiaSystem(world *ecs.World, hooks *engine.SparseHookGraph) *XenophobiaSystem {
	posID := ecs.ComponentID[components.Position](world)
	identID := ecs.ComponentID[components.Identity](world)
	belID := ecs.ComponentID[components.BeliefComponent](world)
	cultureID := ecs.ComponentID[components.CultureComponent](world)

	mask := ecs.All(posID, identID, belID, cultureID)

	return &XenophobiaSystem{
		hooks:     hooks,
		filter:    &mask,
		posID:     posID,
		identID:   identID,
		belID:     belID,
		cultureID: cultureID,
	}
}

func (s *XenophobiaSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process every 10 ticks to keep simulation performant while still reacting quickly
	if s.tickCounter%10 != 0 {
		return
	}

	type npcData struct {
		id         uint64
		x          float32
		y          float32
		languageID uint16
		xenophobe  bool
	}

	nodes := make([]npcData, 0, 500)

	query := world.Query(s.filter)
	for query.Next() {
		pos := (*components.Position)(query.Get(s.posID))
		ident := (*components.Identity)(query.Get(s.identID))
		bel := (*components.BeliefComponent)(query.Get(s.belID))
		culture := (*components.CultureComponent)(query.Get(s.cultureID))

		isXenophobe := false
		for _, b := range bel.Beliefs {
			// Ensure we are catching the belief
			if b.BeliefID == components.BeliefXenophobia && b.Weight >= 50 {
				isXenophobe = true
				break
			}
		}

		nodes = append(nodes, npcData{
			id:         ident.ID,
			x:          pos.X,
			y:          pos.Y,
			languageID: culture.LanguageID,
			xenophobe:  isXenophobe,
		})
	}

	for i := 0; i < len(nodes); i++ {
		actor := nodes[i]

		if !actor.xenophobe {
			continue
		}

		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			target := nodes[j]

			// If languages are different, they are considered foreigners
			if actor.languageID != target.languageID {
				// Distance check
				dx := actor.x - target.x
				dy := actor.y - target.y
				distSq := dx*dx + dy*dy

				if distSq < 10.0 { // Small proximity radius
					// Check if a hook already exists to avoid redundant assignment
					existingHook := s.hooks.GetHook(actor.id, target.id)
					if existingHook > -50 {
						// Instantly assign deep hatred, satisfying the requirement for Blood Feud trigger
						// AddHook uses += internally, so we use -100 to decrease it.
						s.hooks.AddHook(actor.id, target.id, -100)
					}
				}
			}
		}
	}
}