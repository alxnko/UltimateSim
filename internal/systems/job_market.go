package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 15.2: Employment & Wages

// JobMarketSystem matches NPCs to available jobs and distributes wages.
type JobMarketSystem struct {
	world *ecs.World

	// Component IDs
	npcID      ecs.ID
	jobID      ecs.ID
	needsID    ecs.ID
	idID       ecs.ID
	businessID ecs.ID
	treasuryID ecs.ID

	// Internal state tracking
	tickStamp uint64
}

// NewJobMarketSystem creates a new JobMarketSystem.
func NewJobMarketSystem(world *ecs.World) *JobMarketSystem {
	return &JobMarketSystem{
		world:      world,
		npcID:      ecs.ComponentID[components.NPC](world),
		jobID:      ecs.ComponentID[components.JobComponent](world),
		needsID:    ecs.ComponentID[components.Needs](world),
		idID:       ecs.ComponentID[components.Identity](world),
		businessID: ecs.ComponentID[components.BusinessComponent](world),
		treasuryID: ecs.ComponentID[components.TreasuryComponent](world),
	}
}

// Update runs the job market logic, hiring unemployed NPCs and paying wages.
func (s *JobMarketSystem) Update() {
	s.tickStamp++

	// Pay wages every 10 ticks
	if s.tickStamp%10 == 0 {
		s.payWages()
	}

	// Hire unemployed NPCs every 50 ticks
	if s.tickStamp%50 == 0 {
		s.hireNPCs()
	}
}

// payWages transfers wealth from business treasuries to their employees.
func (s *JobMarketSystem) payWages() {
	// 1. Build a map of business treasuries for quick O(1) lookup
	businessTreasuries := make(map[uint64]*components.TreasuryComponent)

	bq := s.world.Query(filter.All(s.businessID, s.idID, s.treasuryID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		treasury := (*components.TreasuryComponent)(bq.Get(s.treasuryID))
		businessTreasuries[id.ID] = treasury
	}

	// 2. Iterate over all employed NPCs and transfer wages
	wageAmount := float32(1.0) // Flat wage rate for simulation simplicity

	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.needsID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))
		needs := (*components.Needs)(eq.Get(s.needsID))

		// If the NPC has a job and an employer
		if job.JobID != components.JobNone && job.EmployerID != 0 {
			if treasury, ok := businessTreasuries[job.EmployerID]; ok {
				// Only pay if the business has enough wealth
				if treasury.Wealth >= wageAmount {
					treasury.Wealth -= wageAmount
					needs.Wealth += wageAmount
				} else {
					// Business cannot afford to pay, NPC leaves job (Strike/Quit)
					job.JobID = components.JobNone
					job.EmployerID = 0
				}
			} else {
				// Employer no longer exists, clear job
				job.JobID = components.JobNone
				job.EmployerID = 0
			}
		}
	}
}

// hireNPCs assigns unemployed NPCs to businesses seeking labor.
func (s *JobMarketSystem) hireNPCs() {
	// 1. Collect all active businesses
	var activeBusinesses []uint64
	bq := s.world.Query(filter.All(s.businessID, s.idID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		activeBusinesses = append(activeBusinesses, id.ID)
	}

	if len(activeBusinesses) == 0 {
		return // No businesses to hire
	}

	// 2. Iterate over unemployed NPCs (or farmers/lumberjacks seeking better jobs)
	// We check those whose Wealth need is low
	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.needsID))

	businessIndex := 0

	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))
		needs := (*components.Needs)(eq.Get(s.needsID))

		// Target NPCs with low wealth and no active employer
		if needs.Wealth < 50.0 && job.EmployerID == 0 {
			// Hire them at the current business in our simple round-robin list
			employerID := activeBusinesses[businessIndex]

			// Set as an Artisan for this example
			job.JobID = components.JobArtisan
			job.EmployerID = employerID

			// Cycle through businesses to distribute labor evenly
			businessIndex = (businessIndex + 1) % len(activeBusinesses)
		}
	}
}
