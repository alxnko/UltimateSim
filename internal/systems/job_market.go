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
	affID      ecs.ID
	strikeID   ecs.ID

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
		affID:      ecs.ComponentID[components.Affiliation](world),
		strikeID:   ecs.ComponentID[components.StrikeMarker](world),
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
	// 1. Build a map of business treasuries and their affiliations for quick O(1) lookup
	businessTreasuries := make(map[uint64]*components.TreasuryComponent)
	businessCities := make(map[uint64]uint32)

	bq := s.world.Query(filter.All(s.businessID, s.idID, s.treasuryID, s.affID))
	for bq.Next() {
		id := (*components.Identity)(bq.Get(s.idID))
		treasury := (*components.TreasuryComponent)(bq.Get(s.treasuryID))
		aff := (*components.Affiliation)(bq.Get(s.affID))
		businessTreasuries[id.ID] = treasury
		businessCities[id.ID] = aff.CityID
	}

	// 2. Pre-cache Local WageRates mapped by CityID
	villageID := ecs.ComponentID[components.Village](s.world)
	marketID := ecs.ComponentID[components.MarketComponent](s.world)
	mq := s.world.Query(filter.All(villageID, marketID, s.affID))
	cityWages := make(map[uint32]float32)
	for mq.Next() {
		market := (*components.MarketComponent)(mq.Get(marketID))
		aff := (*components.Affiliation)(mq.Get(s.affID))
		cityWages[aff.CityID] = market.WageRate
	}

	// 3. Iterate over all employed NPCs and transfer wages
	type strikeTarget struct {
		Entity     ecs.Entity
		EmployerID uint64
	}
	var strikingEntities []strikeTarget

	eq := s.world.Query(filter.All(s.npcID, s.jobID, s.needsID, s.affID))
	for eq.Next() {
		job := (*components.JobComponent)(eq.Get(s.jobID))
		needs := (*components.Needs)(eq.Get(s.needsID))
		aff := (*components.Affiliation)(eq.Get(s.affID))

		// If the NPC has a job and an employer
		if job.JobID != components.JobNone && job.EmployerID != 0 {
			if treasury, ok := businessTreasuries[job.EmployerID]; ok {
				// Resolve dynamic wage from city
				wageAmount := float32(1.0)
				if cityWage, exists := cityWages[aff.CityID]; exists {
					wageAmount = cityWage
				}

				// Only pay if the business has enough wealth
				if treasury.Wealth >= wageAmount {
					treasury.Wealth -= wageAmount
					needs.Wealth += wageAmount
				} else {
					// Phase 24.1: Business cannot afford to pay, NPC leaves job and strikes
					oldEmployerID := job.EmployerID
					job.JobID = components.JobNone
					job.EmployerID = 0

					strikingEntities = append(strikingEntities, strikeTarget{
						Entity:     eq.Entity(),
						EmployerID: oldEmployerID,
					})
				}
			} else {
				// Employer no longer exists, clear job
				job.JobID = components.JobNone
				job.EmployerID = 0
			}
		}
	}

	// 4. Structurally modify the ECS assigning StrikeMarkers
	for _, st := range strikingEntities {
		if !s.world.Has(st.Entity, s.strikeID) {
			s.world.Add(st.Entity, s.strikeID)
			marker := (*components.StrikeMarker)(s.world.Get(st.Entity, s.strikeID))
			marker.TargetEmployerID = st.EmployerID
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
	// Phase 24.1: Strikers refuse to be re-hired by regular queries
	f := filter.All(s.npcID, s.jobID, s.needsID).Without(s.strikeID)
	eq := s.world.Query(&f)

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
