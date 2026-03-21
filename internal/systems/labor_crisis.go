package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 47 - The Plague-Labor Economics Bridge
// Bridges Biological Entropy (Plagues) directly into Economic Friction and Justice (Blood Feuds).
// If a massive population drop is detected, the ensuing LaborCrisis spikes wages exponentially.
// Ambitious NPCs form Trade Unions (StrikeMarkers) and generate BloodFeuds against their
// employers or the state if their new exorbitant wage demands are not met.

type LaborCrisisSystem struct {
	world       *ecs.World
	hooks       *engine.SparseHookGraph
	tickCounter uint64

	// Pre-allocated cache for O(1) checks
	employerTreasuries map[uint64]float32

	// Component IDs
	villageID  ecs.ID
	popID      ecs.ID
	demoID     ecs.ID
	marketID   ecs.ID
	treasuryID ecs.ID
	npcID      ecs.ID
	idID       ecs.ID
	jobID      ecs.ID
	affilID    ecs.ID
	strikeID   ecs.ID
	bizID      ecs.ID
	adminID    ecs.ID
}

func NewLaborCrisisSystem(world *ecs.World, hooks *engine.SparseHookGraph) *LaborCrisisSystem {
	return &LaborCrisisSystem{
		world:              world,
		hooks:              hooks,
		tickCounter:        0,
		employerTreasuries: make(map[uint64]float32),
		villageID:          ecs.ComponentID[components.Village](world),
		popID:              ecs.ComponentID[components.PopulationComponent](world),
		demoID:             ecs.ComponentID[components.DemographicsComponent](world),
		marketID:           ecs.ComponentID[components.MarketComponent](world),
		treasuryID:         ecs.ComponentID[components.TreasuryComponent](world),
		npcID:              ecs.ComponentID[components.NPC](world),
		idID:               ecs.ComponentID[components.Identity](world),
		jobID:              ecs.ComponentID[components.JobComponent](world),
		affilID:            ecs.ComponentID[components.Affiliation](world),
		strikeID:           ecs.ComponentID[components.StrikeMarker](world),
		bizID:              ecs.ComponentID[components.BusinessEntity](world),
		adminID:            ecs.ComponentID[components.AdministrationMarker](world),
	}
}

func (s *LaborCrisisSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle to evaluate macro-economics periodically
	if s.tickCounter%100 != 0 {
		return
	}

	// Active crisis mapping to avoid nested queries
	activeCrises := make(map[uint32]*components.MarketComponent)

	// 1. Evaluate Demographic shifts in Villages
	villageQuery := s.world.Query(filter.All(s.villageID, s.popID, s.marketID, s.demoID, s.affilID))
	for villageQuery.Next() {
		pop := (*components.PopulationComponent)(villageQuery.Get(s.popID))
		demo := (*components.DemographicsComponent)(villageQuery.Get(s.demoID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))

		// Check for massive demographic drop (Plague/Disaster)
		// E.g., population falls below 80% of the peak
		if demo.PeakPopulation > 0 {
			survivalRatio := float32(pop.Count) / float32(demo.PeakPopulation)

			// Trip the crisis flag ONLY ONCE per 80% drop to avoid infinite exponential math
			if survivalRatio < 0.8 && !demo.LaborCrisisActive {
				demo.LaborCrisisActive = true
				market.WageRate *= 3.0
			}
		}

		// Track peak population for next evaluation, but only climb
		if pop.Count > demo.PeakPopulation {
			demo.PeakPopulation = pop.Count

			// If population recovers to peak, end the crisis
			if demo.LaborCrisisActive {
				demo.LaborCrisisActive = false
			}
		}

		// Cache active crisis cities for worker extortion phase
		if demo.LaborCrisisActive {
			activeCrises[affil.CityID] = market
		}
	}

	if len(activeCrises) == 0 {
		return
	}

	// 2. Pre-cache Employer/Business Treasuries for O(1) matching
	clear(s.employerTreasuries)

	// Fetch business entities
	bizQuery := s.world.Query(filter.All(s.bizID, s.idID, s.treasuryID))
	for bizQuery.Next() {
		ident := (*components.Identity)(bizQuery.Get(s.idID))
		treas := (*components.TreasuryComponent)(bizQuery.Get(s.treasuryID))
		s.employerTreasuries[ident.ID] = treas.Wealth
	}

	// Fetch City/State treasuries (Administration Marker often acts as employer for Guards/State jobs)
	adminQuery := s.world.Query(filter.All(s.adminID, s.idID, s.treasuryID))
	for adminQuery.Next() {
		ident := (*components.Identity)(adminQuery.Get(s.idID))
		treas := (*components.TreasuryComponent)(adminQuery.Get(s.treasuryID))
		s.employerTreasuries[ident.ID] = treas.Wealth
	}

	// 3. Evaluate Worker Demands (The Butterfly Effect)
	npcQuery := s.world.Query(filter.All(s.npcID, s.jobID, s.idID, s.affilID))

	// Collect structural modifications to prevent ECS panics
	type strikerData struct {
		Entity     ecs.Entity
		EmployerID uint64
	}
	var newStrikers []strikerData

	for npcQuery.Next() {
		job := (*components.JobComponent)(npcQuery.Get(s.jobID))
		ident := (*components.Identity)(npcQuery.Get(s.idID))
		affil := (*components.Affiliation)(npcQuery.Get(s.affilID))

		// Must be employed
		if job.EmployerID == 0 {
			continue
		}

		// Must be Ambitious to exploit the crisis
		if (ident.BaseTraits & components.TraitAmbitious) == 0 {
			continue
		}

		// Must be in a city undergoing a labor crisis
		market, exists := activeCrises[affil.CityID]
		if !exists {
			continue
		}

		employerWealth, empExists := s.employerTreasuries[job.EmployerID]
		if !empExists {
			employerWealth = 0.0 // Insolvent or missing employer
		}

		// If employer cannot afford the new extortionate wage (assume 10 ticks buffer required)
		if employerWealth < market.WageRate*10.0 {
			// Worker quits
			oldEmployer := job.EmployerID
			job.JobID = components.JobNone
			job.EmployerID = 0

			// Register for StrikeMarker
			newStrikers = append(newStrikers, strikerData{
				Entity:     npcQuery.Entity(),
				EmployerID: oldEmployer,
			})

			// Generate deep systemic hatred towards the employer (Blood Feud trigger)
			if s.hooks != nil {
				s.hooks.AddHook(ident.ID, oldEmployer, -50)
			}
		}
	}

	// Apply StrikeMarkers structurally
	for _, sData := range newStrikers {
		if s.world.Alive(sData.Entity) {
			if !s.world.Has(sData.Entity, s.strikeID) {
				s.world.Add(sData.Entity, s.strikeID)
			}
			strike := (*components.StrikeMarker)(s.world.Get(sData.Entity, s.strikeID))
			strike.TargetEmployerID = sData.EmployerID
		}
	}
}
