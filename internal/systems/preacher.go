package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 20.1: Ideological Warfare (PreacherSystem)
// Preachers actively override target BeliefComponent weights across vast regions.

type PreacherSystem struct {
	tickCounter uint64

	// Component IDs
	posID    ecs.ID
	jobID    ecs.ID
	beliefID ecs.ID
	ruinID   ecs.ID
}

func NewPreacherSystem(world *ecs.World) *PreacherSystem {
	return &PreacherSystem{
		posID:    ecs.ComponentID[components.Position](world),
		jobID:    ecs.ComponentID[components.JobComponent](world),
		beliefID: ecs.ComponentID[components.BeliefComponent](world),
		ruinID:   ecs.ComponentID[components.RuinComponent](world),
	}
}

func (s *PreacherSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Runs every 50 ticks
	if s.tickCounter%50 != 0 {
		return
	}

	// Filter all valid actors capable of holding beliefs
	filter := ecs.All(s.posID, s.beliefID).Without(s.ruinID)
	query := world.Query(&filter)

	type nodeData struct {
		entity ecs.Entity
		pos    *components.Position
		job    *components.JobComponent // Optional
		belief *components.BeliefComponent
	}

	var nodes []nodeData

	// O(N) extraction to flat array
	for query.Next() {
		var job *components.JobComponent
		if query.Has(s.jobID) {
			job = (*components.JobComponent)(query.Get(s.jobID))
		}

		nodes = append(nodes, nodeData{
			entity: query.Entity(),
			pos:    (*components.Position)(query.Get(s.posID)),
			job:    job,
			belief: (*components.BeliefComponent)(query.Get(s.beliefID)),
		})
	}

	// O(N^2) loop to let Preachers influence nearby nodes over vast distances
	for i := 0; i < len(nodes); i++ {
		preacher := nodes[i]

		if preacher.job == nil || preacher.job.JobID != components.JobPreacher {
			continue
		}

		if len(preacher.belief.Beliefs) == 0 {
			continue
		}

		// Find the preacher's strongest belief
		var strongestBeliefID uint32
		var maxWeight int32 = -1

		for _, b := range preacher.belief.Beliefs {
			if b.Weight > maxWeight {
				maxWeight = b.Weight
				strongestBeliefID = b.BeliefID
			}
		}

		if strongestBeliefID == 0 {
			continue
		}

		// Influence others
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			target := nodes[j]

			// Preachers target over a vast region: radius 20.0 = distSq 400.0
			dx := preacher.pos.X - target.pos.X
			dy := preacher.pos.Y - target.pos.Y
			distSq := dx*dx + dy*dy

			if distSq < 400.0 {
				found := false
				// Suppress competing beliefs and elevate the Preacher's strongest belief
				for k := range target.belief.Beliefs {
					if target.belief.Beliefs[k].BeliefID == strongestBeliefID {
						target.belief.Beliefs[k].Weight += 5
						found = true
					} else {
						// Suppress competing belief
						if target.belief.Beliefs[k].Weight > 0 {
							target.belief.Beliefs[k].Weight -= 1
						}
					}
				}

				if !found {
					target.belief.Beliefs = append(target.belief.Beliefs, components.Belief{
						BeliefID: strongestBeliefID,
						Weight:   5,
					})
				}
			}
		}
	}
}
