package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 60 - The Physical Crafting Engine
// CraftingSystem implements the Rimworld Layer for artisans.
// Artisans must physically be at a WorkbenchComponent to process raw materials (Iron)
// into massive Market Wealth for their Employer.

type EmployerData struct {
	Workbenches []*components.WorkbenchComponent
	Storage     *components.StorageComponent
	Treasury    *components.TreasuryComponent
}

type CraftingSystem struct {
	tickCounter   uint64
	employerCache map[uint64]EmployerData

	// Pre-cached component IDs
	npcID   ecs.ID
	posID   ecs.ID
	jobID   ecs.ID
	wbID    ecs.ID
	idID    ecs.ID
	storID  ecs.ID
	treasID ecs.ID
}

func NewCraftingSystem(world *ecs.World) *CraftingSystem {
	return &CraftingSystem{
		tickCounter:   0,
		employerCache: make(map[uint64]EmployerData),
		npcID:         ecs.ComponentID[components.NPC](world),
		posID:         ecs.ComponentID[components.Position](world),
		jobID:         ecs.ComponentID[components.JobComponent](world),
		wbID:          ecs.ComponentID[components.WorkbenchComponent](world),
		idID:          ecs.ComponentID[components.Identity](world),
		storID:        ecs.ComponentID[components.StorageComponent](world),
		treasID:       ecs.ComponentID[components.TreasuryComponent](world),
	}
}

func (s *CraftingSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Artisans craft in batches every 20 ticks
	if s.tickCounter%20 != 0 {
		return
	}

	// Clear maps & slices for DOD memory reuse
	for id, data := range s.employerCache {
		data.Workbenches = data.Workbenches[:0]
		s.employerCache[id] = data
	}
	clear(s.employerCache)

	// DOD Pre-caching strategy:
	// We map EmployerID -> []WorkbenchComponent, StorageComponent, and TreasuryComponent.
	// This avoids nested O(N^2) arche-go queries inside the main Artisan loop.

	// 1. Gather all active Workbenches
	wbQuery := world.Query(filter.All(s.wbID))
	for wbQuery.Next() {
		wb := (*components.WorkbenchComponent)(wbQuery.Get(s.wbID))

		data := s.employerCache[wb.EmployerID]
		data.Workbenches = append(data.Workbenches, wb)
		s.employerCache[wb.EmployerID] = data
	}

	// 2. Gather Storage & Treasury for these Employers
	busQuery := world.Query(filter.All(s.idID, s.storID, s.treasID))
	for busQuery.Next() {
		id := (*components.Identity)(busQuery.Get(s.idID))

		// Only cache if they have workbenches, optimizing memory
		if data, exists := s.employerCache[id.ID]; exists {
			data.Storage = (*components.StorageComponent)(busQuery.Get(s.storID))
			data.Treasury = (*components.TreasuryComponent)(busQuery.Get(s.treasID))
			s.employerCache[id.ID] = data
		}
	}

	// 3. Main Loop: Iterate all Artisans
	npcQuery := world.Query(filter.All(s.npcID, s.posID, s.jobID))

	for npcQuery.Next() {
		job := (*components.JobComponent)(npcQuery.Get(s.jobID))

		// Ensure NPC is an Artisan
		if job.JobID != components.JobArtisan || job.EmployerID == 0 {
			continue
		}

		// Look up employer data
		data, hasEmployer := s.employerCache[job.EmployerID]
		if !hasEmployer || len(data.Workbenches) == 0 || data.Storage == nil || data.Treasury == nil {
			continue
		}

		pos := (*components.Position)(npcQuery.Get(s.posID))

		// Check if physically at ANY of the employer's workbenches (allowing a small radius for pathing float precision)
		atWorkbench := false
		for _, wb := range data.Workbenches {
			dx := pos.X - wb.X
			dy := pos.Y - wb.Y
			distSq := dx*dx + dy*dy

			if distSq <= 1.0 { // Within 1 unit distance
				atWorkbench = true
				break
			}
		}

		if atWorkbench {
			// Check if they have raw Iron
			if data.Storage.Iron >= 5 {
				// Crafting success: Drain Iron, increase Treasury Wealth heavily
				data.Storage.Iron -= 5
				data.Treasury.Wealth += 50.0 // Representing physical sale of high-tier crafted goods
			}
		}
	}
}
