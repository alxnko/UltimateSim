package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 41: The Ostracization Engine
// OstracizationSystem evaluates Memory buffers for unpunished negative interactions.
// It translates recorded thefts or assaults into deep negative hooks in the SparseHookGraph.

// cache values, not component pointers — GC corruption class, see banditry.go
// Memory (written) is re-fetched via the entity handle at use time.
type ostracizationNodeData struct {
	entity ecs.Entity
	id     uint64
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

		nodes = append(nodes, ostracizationNodeData{
			entity: query.Entity(),
			id:     ident.ID,
		})
	}

	for i := 0; i < len(nodes); i++ {
		node := nodes[i]

		if !world.Alive(node.entity) {
			continue
		}
		mem := (*components.Memory)(world.Get(node.entity, s.memID))

		// Evaluate memory buffer
		for j := 0; j < len(mem.Events); j++ {
			ev := &mem.Events[j]

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
