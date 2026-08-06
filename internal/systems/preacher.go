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

	// cache values, not component pointers — GC corruption class, see banditry.go
	// BeliefComponent (written) is re-fetched via the entity handle at use time.
	type nodeData struct {
		entity ecs.Entity
		x      float32
		y      float32
		hasJob bool // Optional
		jobID  uint8
	}

	var nodes []nodeData

	// O(N) extraction to flat array
	for query.Next() {
		hasJob := false
		var jobID uint8
		if query.Has(s.jobID) {
			job := (*components.JobComponent)(query.Get(s.jobID))
			hasJob = true
			jobID = job.JobID
		}

		pos := (*components.Position)(query.Get(s.posID))

		nodes = append(nodes, nodeData{
			entity: query.Entity(),
			x:      pos.X,
			y:      pos.Y,
			hasJob: hasJob,
			jobID:  jobID,
		})
	}

	// O(N^2) loop to let Preachers influence nearby nodes over vast distances
	for i := 0; i < len(nodes); i++ {
		preacher := nodes[i]

		if !preacher.hasJob || preacher.jobID != components.JobPreacher {
			continue
		}

		// Re-fetch via entity handle at use time (cached pointers do not survive archetype moves)
		if !world.Alive(preacher.entity) {
			continue
		}
		preacherBelief := (*components.BeliefComponent)(world.Get(preacher.entity, s.beliefID))

		if len(preacherBelief.Beliefs) == 0 {
			continue
		}

		// Find the preacher's strongest belief
		var strongestBeliefID uint32
		var maxWeight int32 = -1

		for _, b := range preacherBelief.Beliefs {
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
			dx := preacher.x - target.x
			dy := preacher.y - target.y
			distSq := dx*dx + dy*dy

			if distSq < 400.0 {
				// Re-fetch via entity handle at use time (cached pointers do not survive archetype moves)
				if !world.Alive(target.entity) {
					continue
				}
				targetBelief := (*components.BeliefComponent)(world.Get(target.entity, s.beliefID))

				found := false
				// Suppress competing beliefs and elevate the Preacher's strongest belief
				for k := range targetBelief.Beliefs {
					if targetBelief.Beliefs[k].BeliefID == strongestBeliefID {
						targetBelief.Beliefs[k].Weight += 5
						found = true
					} else {
						// Suppress competing belief
						if targetBelief.Beliefs[k].Weight > 0 {
							targetBelief.Beliefs[k].Weight -= 1
						}
					}
				}

				if !found {
					targetBelief.Beliefs = append(targetBelief.Beliefs, components.Belief{
						BeliefID: strongestBeliefID,
						Weight:   5,
					})
				}
			}
		}
	}
}
