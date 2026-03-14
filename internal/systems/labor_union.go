package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 24.1: The Labor Union Engine (Systemic Emergence)
// LaborUnionSystem coordinates Strikers and assigns brutal generational grudges against "Scabs" via SparseHookGraph.

type LaborUnionSystem struct {
	hookGraph *engine.SparseHookGraph

	// Component IDs
	strikeID ecs.ID
	idID     ecs.ID
	jobID    ecs.ID
	npcID    ecs.ID
}

func NewLaborUnionSystem(world *ecs.World, hookGraph *engine.SparseHookGraph) *LaborUnionSystem {
	return &LaborUnionSystem{
		hookGraph: hookGraph,
		strikeID:  ecs.ComponentID[components.StrikeMarker](world),
		idID:      ecs.ComponentID[components.Identity](world),
		jobID:     ecs.ComponentID[components.JobComponent](world),
		npcID:     ecs.ComponentID[components.NPC](world),
	}
}

type strikeData struct {
	StrikerID        uint64
	TargetEmployerID uint64
}

func (s *LaborUnionSystem) Update(world *ecs.World) {
	if s.hookGraph == nil {
		return
	}

	// 1. Extract all active strikers into a flat slice to avoid O(N^2) nested arche-go queries
	var activeStrikes []strikeData

	strikeQuery := world.Query(filter.All(s.strikeID, s.idID))
	for strikeQuery.Next() {
		id := (*components.Identity)(strikeQuery.Get(s.idID))
		marker := (*components.StrikeMarker)(strikeQuery.Get(s.strikeID))

		activeStrikes = append(activeStrikes, strikeData{
			StrikerID:        id.ID,
			TargetEmployerID: marker.TargetEmployerID,
		})
	}

	if len(activeStrikes) == 0 {
		return // No active strikes
	}

	// 2. Iterate all currently employed NPCs ("Potential Scabs")
	scabQuery := world.Query(filter.All(s.npcID, s.jobID, s.idID))

	for scabQuery.Next() {
		job := (*components.JobComponent)(scabQuery.Get(s.jobID))
		scabID := (*components.Identity)(scabQuery.Get(s.idID))

		if job.EmployerID == 0 {
			continue // Not employed
		}

		// 3. O(N) evaluation against the flat active strikes cache
		for _, strike := range activeStrikes {
			if job.EmployerID == strike.TargetEmployerID {
				// Check if the hook already exists to prevent massive infinite stack overflow per tick.
				// We want a static -50 baseline.
				currentScabHook := s.hookGraph.GetHook(strike.StrikerID, scabID.ID)
				if currentScabHook > -50 {
					// The NPC is actively working for a business currently being struck.
					// Phase 24.1: The Butterfly Effect (Labor, Economy, Justice)
					// Striker generates massive negative hooks (-50) against the scab, triggering BloodFeudSystem (Phase 23.1)
					s.hookGraph.AddHook(strike.StrikerID, scabID.ID, -50)
				}

				currentEmpHook := s.hookGraph.GetHook(strike.StrikerID, job.EmployerID)
				if currentEmpHook > -10 {
					// Striker generates moderate negative hooks (-10) against the offending employer
					s.hookGraph.AddHook(strike.StrikerID, job.EmployerID, -10)
				}
			}
		}
	}
}
