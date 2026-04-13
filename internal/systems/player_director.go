package systems

import (
	"fmt"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// PlayerDirectorSystem evaluates local simulation state to suggest emergent actions.
type PlayerDirectorSystem struct {
	hookGraph   *engine.SparseHookGraph
	filter      ecs.Filter
	tickCounter uint64
}

// NewPlayerDirectorSystem creates a new PlayerDirectorSystem.
func NewPlayerDirectorSystem(hookGraph *engine.SparseHookGraph) *PlayerDirectorSystem {
	return &PlayerDirectorSystem{
		hookGraph: hookGraph,
	}
}

// Initialize sets up the Arche-Go filter.
func (s *PlayerDirectorSystem) Initialize(world *ecs.World) {
	s.filter = ecs.All(
		ecs.ComponentID[components.Possessed](world),
		ecs.ComponentID[components.Position](world),
		ecs.ComponentID[components.Identity](world),
	)
}

// Update evaluates surroundings and prints suggestions to the console (as a placeholder for UI).
func (s *PlayerDirectorSystem) Update(world *ecs.World) {
	s.tickCounter++
	query := world.Query(s.filter)
	for query.Next() {
		pos := (*components.Position)(query.Get(ecs.ComponentID[components.Position](world)))
		id := (*components.Identity)(query.Get(ecs.ComponentID[components.Identity](world)))

		// 1. Scan for nearby Crimes (Opportunities for Agent of Chaos)
		crimeFilter := ecs.All(ecs.ComponentID[components.CrimeMarker](world), ecs.ComponentID[components.Position](world))
		crimeQuery := world.Query(crimeFilter)
		for crimeQuery.Next() {
			targetPos := (*components.Position)(crimeQuery.Get(ecs.ComponentID[components.Position](world)))
			dx, dy := pos.X-targetPos.X, pos.Y-targetPos.Y
			distSq := dx*dx + dy*dy
			if distSq < 100.0 {
				fmt.Printf("[DIRECTOR] ALERT: Local criminal detected. Bounty available.\n")
			}
		}

		// 2. Scan for nearby Grudges (Opportunities for Mercenary Work)
		// This logic would ideally use the hook graph to find nearby NPCs with high negative hooks.
		// For now, we'll just print a generic message based on the tick count.
		if s.tickCounter % 1000 == 0 {
			fmt.Printf("[DIRECTOR] TIP: The local Clan ID %d is harboring grudges. Check the tavern for contracts.\n", id.ID % 10)
		}
	}
}
