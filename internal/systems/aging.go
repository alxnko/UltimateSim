package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 19.3: Biological Entropy (Aging)
// Increments age for all entities and citizens, reducing health linearly after age 50
// and dramatically increasing sudden death probability (Needs.Food = 0) after age 80.

type AgingSystem struct {
	npcFilter   ecs.Filter
	popFilter   ecs.Filter
	tickCounter uint64
	tm          *engine.TickManager
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *AgingSystem) IsExpensive() bool {
	return true
}

// NewAgingSystem creates a new AgingSystem.
func NewAgingSystem(world *ecs.World, tm *engine.TickManager) *AgingSystem {
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	npcMask := ecs.All(idID, genID, needsID).Without(ruinID)

	popID := ecs.ComponentID[components.PopulationComponent](world)
	villageID := ecs.ComponentID[components.Village](world)

	popMask := ecs.All(popID, villageID).Without(ruinID)

	return &AgingSystem{
		npcFilter:   &npcMask,
		popFilter:   &popMask,
		tickCounter: 0,
		tm:          tm,
	}
}

// Update executes the aging logic. A "year" is 360 ticks.
func (s *AgingSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process once every 360 ticks
	if s.tickCounter%360 != 0 {
		return
	}

	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)

	// 1. Process active NPCs
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		id := (*components.Identity)(npcQuery.Get(idID))
		gen := (*components.GenomeComponent)(npcQuery.Get(genID))
		needs := (*components.Needs)(npcQuery.Get(needsID))

		id.Age++

		// Apply linear entropy penalty
		if id.Age > 50 {
			if gen.Health > 0 {
				gen.Health--
			}
		}

		// Apply severe sudden death chance
		if id.Age > 80 {
			// Increase chance of death the older they get. Base 5% chance at 80, +1% per year
			deathChance := float32(id.Age-80)*0.01 + 0.05
			if engine.GetRandomFloat32() < deathChance {
				needs.Food = 0 // Will be reaped by DeathSystem
			}
		}
	}

	// 2. Process abstracted PopulationComponent (Citizens in Villages)
	popQuery := world.Query(s.popFilter)
	for popQuery.Next() {
		pop := (*components.PopulationComponent)(popQuery.Get(popID))

		count := len(pop.Citizens)
		if count == 0 {
			continue
		}

		// Phase 31.7: Statistical Aging Sampling during FastForward
		// If population is huge, we only process 10% and extrapolate.
		useSampling := s.tm != nil && s.tm.IsFastForward && count > 1000
		step := 1
		if useSampling {
			step = 10
		}

		survivingCitizens := pop.Citizens[:0] // Retain capacity

		for i := 0; i < count; i += step {
			cit := pop.Citizens[i]
			cit.Age++

			if cit.Age > 50 {
				if cit.Genetics.Health > 0 {
					cit.Genetics.Health--
				}
			}

			survives := true
			if cit.Age > 80 {
				deathChance := float32(cit.Age-80)*0.01 + 0.05
				if engine.GetRandomFloat32() < deathChance {
					survives = false
				}
			}

			if survives {
				survivingCitizens = append(survivingCitizens, cit)
				// If sampling, we "keep" the others by assuming they lived if this one lived
				if useSampling {
					for j := 1; j < step && (i+j) < count; j++ {
						// Simple clone of neighboring citizen to maintain count
						// (Statistical approximation of genetic diversity)
						survivingCitizens = append(survivingCitizens, pop.Citizens[i+j])
					}
				}
			} else {
				// Citizen died. If sampling, we assume 'step' people died.
				if useSampling {
					if pop.Count >= uint32(step) {
						pop.Count -= uint32(step)
					} else {
						pop.Count = 0
					}
				} else {
					if pop.Count > 0 {
						pop.Count--
					}
				}
			}
		}

		pop.Citizens = survivingCitizens
	}
}
