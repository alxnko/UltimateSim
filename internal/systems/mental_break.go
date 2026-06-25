package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 62: The Psychological Stress Engine
// Bridges Biology (Pain) into Psychological trauma (Stress) and Justice (CrimeMarker).

type MentalBreakSystem struct {
	filter      ecs.Filter
	tickCounter uint64

	vitalsID ecs.ID
	sanityID ecs.ID
	crimeID  ecs.ID
	velID    ecs.ID
	pathID   ecs.ID
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *MentalBreakSystem) IsExpensive() bool {
	return true
}

// NewMentalBreakSystem creates a new MentalBreakSystem.
func NewMentalBreakSystem(world *ecs.World) *MentalBreakSystem {
	vID := ecs.ComponentID[components.VitalsComponent](world)
	sID := ecs.ComponentID[components.SanityComponent](world)
	cID := ecs.ComponentID[components.CrimeMarker](world)
	velID := ecs.ComponentID[components.Velocity](world)
	pathID := ecs.ComponentID[components.Path](world)

	return &MentalBreakSystem{
		filter:   filter.All(vID, sID),
		vitalsID: vID,
		sanityID: sID,
		crimeID:  cID,
		velID:    velID,
		pathID:   pathID,
	}
}

// Update executes the system logic.
func (s *MentalBreakSystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%10 != 0 { // Evaluate stress organically every 10 ticks.
		return
	}

	query := world.Query(s.filter)

	// Collect structural modifications to do after query iteration
	type breakData struct {
		Entity ecs.Entity
		State  uint32
	}
	var breaks []breakData

	for query.Next() {
		vitals := (*components.VitalsComponent)(query.Get(s.vitalsID))
		sanity := (*components.SanityComponent)(query.Get(s.sanityID))

		// Pain directly feeds into Stress
		if vitals.Pain > 0 {
			sanity.Stress += vitals.Pain * 0.1 // 10 ticks elapsed
		} else {
			if sanity.Stress > 0 {
				sanity.Stress -= 0.5 // Natural recovery if no pain
			}
			if sanity.Stress < 0 {
				sanity.Stress = 0
			}
		}

		if sanity.BreakCooldown > 0 {
			sanity.BreakCooldown--
			continue
		}

		if sanity.Stress >= sanity.MaxStress {
			// Trigger a break.
			// To keep deterministic, we use RNG
			roll := engine.GetRandomInt() % 100

			var breakState uint32
			if roll < 50 {
				breakState = components.BreakBerserk
			} else {
				breakState = components.BreakCatatonic
			}

			sanity.BreakState = breakState
			sanity.BreakCooldown = 100 // Break lasts 100 * 10 ticks = 1000 ticks
			sanity.Stress = 0          // Reset stress after break

			breaks = append(breaks, breakData{
				Entity: query.Entity(),
				State:  breakState,
			})
		}
	}

	// Apply structural modifications and side effects
	for _, b := range breaks {
		if b.State == components.BreakBerserk {
			// Attach a severe CrimeMarker so Justice system responds
			if !world.Has(b.Entity, s.crimeID) {
				world.Add(b.Entity, s.crimeID)
			}
			// Important: Re-fetch after structural modification
			crime := (*components.CrimeMarker)(world.Get(b.Entity, s.crimeID))
			crime.CrimeLevel = 3
			crime.Bounty = 200
		} else if b.State == components.BreakCatatonic {
			// Zero out velocity to halt the NPC
			if world.Has(b.Entity, s.velID) {
				vel := (*components.Velocity)(world.Get(b.Entity, s.velID))
				vel.X = 0
				vel.Y = 0
			}
			// Zero out path to halt movement intent
			if world.Has(b.Entity, s.pathID) {
				path := (*components.Path)(world.Get(b.Entity, s.pathID))
				path.HasPath = false
				path.Nodes = path.Nodes[:0]
			}
		}
	}
}
