package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 41: The Ostracization Engine
// OstracizationSystem evaluates Memory buffers for unpunished negative interactions.
// It translates recorded thefts or assaults into deep negative hooks in the SparseHookGraph.

type ostracizationNodeData struct {
	entity ecs.Entity
	id     uint64
	mem    *components.Memory
}

type OstracizationSystem struct {
	tickCounter uint64
	hooks       *engine.SparseHookGraph
	filter      ecs.Filter

	// Component IDs mapped once during NewOstracizationSystem
	identID ecs.ID
	memID   ecs.ID
}

func NewOstracizationSystem(world *ecs.World, hooks *engine.SparseHookGraph) *OstracizationSystem {
	identID := ecs.ComponentID[components.Identity](world)
	memID := ecs.ComponentID[components.Memory](world)

	mask := ecs.All(identID, memID)

	return &OstracizationSystem{
		hooks:   hooks,
		filter:  &mask,
		identID: identID,
		memID:   memID,
	}
}

func (s *OstracizationSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run on an offset tick to avoid bottlenecking
	if s.tickCounter%20 != 0 {
		return
	}

	// Extract NPCs into a flat DOD slice
	query := world.Query(s.filter)
	nodes := make([]ostracizationNodeData, 0, 500)

	for query.Next() {
		ident := (*components.Identity)(query.Get(s.identID))
		mem := (*components.Memory)(query.Get(s.memID))

		nodes = append(nodes, ostracizationNodeData{
			entity: query.Entity(),
			id:     ident.ID,
			mem:    mem,
		})
	}

	for i := 0; i < len(nodes); i++ {
		node := nodes[i]

		// Evaluate memory buffer
		for j := 0; j < len(node.mem.Events); j++ {
			ev := &node.mem.Events[j]

			if ev.InteractionType == components.InteractionTheft || ev.InteractionType == components.InteractionAssault {
				// Translate the negative memory into a concrete grudge
				s.hooks.AddHook(node.id, ev.TargetID, -20)

				// Clear the event to prevent infinitely processing the same memory every 20 ticks
				ev.InteractionType = 0
				ev.TargetID = 0
			}
		}
	}
}
