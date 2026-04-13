package systems

import (
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 56: The Conscription Engine (Demographic War Attrition)
// Ties Macro-Geopolitics (War) to Micro-Biology (Population).
// When a State is actively at war, it structurally drafts its abstract population over time.

type ConscriptionSystem struct {
	tickStamp uint64
}

// NewConscriptionSystem initializes the conscription logic
func NewConscriptionSystem() *ConscriptionSystem {
	return &ConscriptionSystem{
		tickStamp: 0,
	}
}

// Update evaluates active wars and exhausts population
func (s *ConscriptionSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Throttle execution to every 300 ticks
	if s.tickStamp%300 != 0 {
		return
	}

	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	warTrackerID := ecs.ComponentID[components.WarTrackerComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)

	// Pre-cache component IDs safely
	query := world.Query(filter.All(capitalID, warTrackerID, popID))

	// Seed local PRNG to guarantee determinism without polluting global state
	var seed [32]byte
	seed[0] = byte(s.tickStamp)
	seed[1] = byte(s.tickStamp >> 8)
	prng := rand.New(rand.NewChaCha8(seed))

	for query.Next() {
		warTracker := (*components.WarTrackerComponent)(query.Get(warTrackerID))
		pop := (*components.PopulationComponent)(query.Get(popID))

		// If actively at war, draft citizens
		if warTracker.Active && pop.Count > 0 {
			// Random casualties between 1 and 5 per cycle
			casualties := uint32(prng.IntN(5) + 1)
			if casualties > pop.Count {
				casualties = pop.Count
			}

			pop.Count -= casualties

			// Truncate the abstract citizens array
			currentLen := uint32(len(pop.Citizens))
			if casualties > currentLen {
				pop.Citizens = pop.Citizens[:0]
			} else {
				pop.Citizens = pop.Citizens[:currentLen-casualties]
			}
		}
	}
}
