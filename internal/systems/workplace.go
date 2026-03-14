package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 15.4: Physical Locations & Workplaces
// WorkplaceSystem manages NPC travel to workplaces and calculates productivity.

type BusinessData struct {
	Workplace *components.WorkplaceComponent
	Treasury  *components.TreasuryComponent
}

type WorkplaceSystem struct {
	pathQueue        *engine.PathRequestQueue
	tickStamp        uint64
	activeBusinesses map[uint64]BusinessData
}

// NewWorkplaceSystem creates a new WorkplaceSystem.
func NewWorkplaceSystem(pathQueue *engine.PathRequestQueue) *WorkplaceSystem {
	return &WorkplaceSystem{
		pathQueue:        pathQueue,
		activeBusinesses: make(map[uint64]BusinessData),
	}
}

// Update runs the workplace logic, directing NPCs to work and calculating productivity.
func (s *WorkplaceSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Component IDs
	npcID := ecs.ComponentID[components.NPC](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)
	geneticsID := ecs.ComponentID[components.GenomeComponent](world)
	idID := ecs.ComponentID[components.Identity](world)
	businessID := ecs.ComponentID[components.BusinessComponent](world)
	workplaceID := ecs.ComponentID[components.WorkplaceComponent](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	// 1. Build a map of active business workplaces and treasuries for quick O(1) lookup
	// To strictly adhere to DOD rules and prevent nested ECS query loops without allocating
	// on every tick, we reuse and clear the activeBusinesses map.
	clear(s.activeBusinesses)

	// Query businesses with Workplace
	bq := world.Query(ecs.All(businessID, workplaceID, idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(idID))
		wp := (*components.WorkplaceComponent)(bq.Get(workplaceID))

		data := BusinessData{
			Workplace: wp,
		}

		if bq.Has(treasuryID) {
			data.Treasury = (*components.TreasuryComponent)(bq.Get(treasuryID))
		}

		s.activeBusinesses[id.ID] = data
	}

	// 2. Process all employed NPCs
	isWorkCycle := s.tickStamp%3600 == 0

	eq := world.Query(ecs.All(npcID, jobID, posID, pathID, geneticsID, idID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(jobID))

		// Skip unemployed NPCs
		if job.EmployerID == 0 || job.JobID == components.JobNone {
			continue
		}

		businessData, exists := s.activeBusinesses[job.EmployerID]
		if !exists {
			// Employer doesn't exist or doesn't have a workplace
			continue
		}

		pos := (*components.Position)(eq.Get(posID))
		path := (*components.Path)(eq.Get(pathID))
		genetics := (*components.GenomeComponent)(eq.Get(geneticsID))
		id := (*components.Identity)(eq.Get(idID))

		// Calculate distance once per NPC iteration
		dx := pos.X - businessData.Workplace.X
		dy := pos.Y - businessData.Workplace.Y
		distSq := dx*dx + dy*dy

		isAtWork := distSq <= 1.0

		// 3. Dispatch PathRequest once per cycle to travel to work
		if isWorkCycle {
			if !isAtWork { // 1.0 threshold for being "at work"
				req := engine.PathRequest{
					EntityID: id.ID,
					StartX:   pos.X,
					StartY:   pos.Y,
					TargetX:  businessData.Workplace.X,
					TargetY:  businessData.Workplace.Y,
				}
				s.pathQueue.Enqueue(req)
				path.HasPath = true
				path.TargetX = businessData.Workplace.X
				path.TargetY = businessData.Workplace.Y
			}
		}

		// 4. Calculate Productivity
		// If the NPC is physically at the workplace, increase output
		if isAtWork { // Within 1.0 units is considered at the location
			// Productivity formula based on Phase 15.4: Genetics.Strength/Intellect
			productivityBoost := float32(genetics.Strength)*0.01 + float32(genetics.Intellect)*0.01

			if businessData.Treasury != nil {
				businessData.Treasury.Wealth += productivityBoost
			}
		}
	}
}
