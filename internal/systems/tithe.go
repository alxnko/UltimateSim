package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 30.1 - Ideological Economy (The Tithe Engine)
// TitheSystem executes wealth transfer from devout NPCs to local Preachers.
// It bridges the Memetic Engine (Phase 07/20) with the Economic Engine (Phase 13/15).

type preacherData struct {
	entity            ecs.Entity
	x                 float32
	y                 float32
	dominantBeliefID  uint32
}

type TitheSystem struct {
	tickCounter uint64

	// Component IDs
	posID    ecs.ID
	jobID    ecs.ID
	beliefID ecs.ID
	needsID  ecs.ID
	npcID    ecs.ID

	preachers []preacherData // Pre-allocated slice for DOD iteration
}

// NewTitheSystem creates a new TitheSystem.
func NewTitheSystem(world *ecs.World) *TitheSystem {
	return &TitheSystem{
		tickCounter: 0,
		posID:       ecs.ComponentID[components.Position](world),
		jobID:       ecs.ComponentID[components.JobComponent](world),
		beliefID:    ecs.ComponentID[components.BeliefComponent](world),
		needsID:     ecs.ComponentID[components.Needs](world),
		npcID:       ecs.ComponentID[components.NPC](world),
		preachers:   make([]preacherData, 0, 50),
	}
}

// Update evaluates tithe transfers every 50 ticks to simulate monthly/weekly cycles.
func (s *TitheSystem) Update(world *ecs.World) {
	s.tickCounter++

	if s.tickCounter%50 != 0 {
		return
	}

	s.preachers = s.preachers[:0]

	// 1. Extract all Preachers and find their dominant belief
	preacherFilter := ecs.All(s.posID, s.jobID, s.beliefID, s.needsID)
	preacherQuery := world.Query(preacherFilter)

	for preacherQuery.Next() {
		job := (*components.JobComponent)(preacherQuery.Get(s.jobID))
		if job.JobID != components.JobPreacher {
			continue
		}

		belief := (*components.BeliefComponent)(preacherQuery.Get(s.beliefID))
		if len(belief.Beliefs) == 0 {
			continue // Unbelieving preacher
		}

		var dominantBeliefID uint32
		var maxWeight int32 = -1

		for _, b := range belief.Beliefs {
			if b.Weight > maxWeight {
				maxWeight = b.Weight
				dominantBeliefID = b.BeliefID
			}
		}

		if dominantBeliefID == 0 {
			continue
		}

		pos := (*components.Position)(preacherQuery.Get(s.posID))

		s.preachers = append(s.preachers, preacherData{
			entity:           preacherQuery.Entity(),
			x:                pos.X,
			y:                pos.Y,
			dominantBeliefID: dominantBeliefID,
		})
	}

	if len(s.preachers) == 0 {
		return // No preachers to collect tithes
	}

	// 2. Iterate over all NPCs and collect tithes if they are near a preacher of the same belief
	npcFilter := ecs.All(s.npcID, s.posID, s.beliefID, s.needsID)
	npcQuery := world.Query(npcFilter)

	for npcQuery.Next() {
		needs := (*components.Needs)(npcQuery.Get(s.needsID))

		// Only collect from those who have wealth
		if needs.Wealth <= 0 {
			continue
		}

		// Also make sure they have a job/employer, not just a random wanderer, unless they are very devout.
		// For Total Simulation, anyone with wealth pays the church.

		belief := (*components.BeliefComponent)(npcQuery.Get(s.beliefID))
		if len(belief.Beliefs) == 0 {
			continue
		}

		// Find NPC's dominant belief
		var dominantBeliefID uint32
		var maxWeight int32 = -1

		for _, b := range belief.Beliefs {
			if b.Weight > maxWeight {
				maxWeight = b.Weight
				dominantBeliefID = b.BeliefID
			}
		}

		if dominantBeliefID == 0 {
			continue
		}

		pos := (*components.Position)(npcQuery.Get(s.posID))

		// Find nearby preacher of the same belief
		for i := 0; i < len(s.preachers); i++ {
			p := s.preachers[i]

			if p.dominantBeliefID == dominantBeliefID {
				// Distance check
				dx := pos.X - p.x
				dy := pos.Y - p.y
				distSq := dx*dx + dy*dy

				// "Parish" radius of ~5 tiles (distSq < 25.0)
				if distSq < 25.0 {
					// Preacher is close enough. Extract 10% of NPC's wealth.
					titheAmount := needs.Wealth * 0.10
					needs.Wealth -= titheAmount

					// Add to preacher's wealth natively without modifying world state
					if world.Alive(p.entity) && world.Has(p.entity, s.needsID) {
						pNeeds := (*components.Needs)(world.Get(p.entity, s.needsID))
						pNeeds.Wealth += titheAmount
					}

					break // Only pay tithe to one preacher
				}
			}
		}
	}
}
