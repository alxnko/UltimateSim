package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 60 - The Physical Crafting Engine
// CraftingSystem requires JobArtisan NPCs to physically walk to a WorkbenchComponent
// to consume Iron and convert it into market Wealth for their employer.

// cache values, not component pointers — GC corruption class, see banditry.go.
// Employers are cached as entity handles (Treasury/Storage are written through,
// so they are re-fetched via world.Get at use time); workbenches are read-only
// so their coordinates are copied by value.
type workbenchData struct {
	X float32
	Y float32
}

type CraftingSystem struct {
	world     *ecs.World
	pathQueue *engine.PathRequestQueue

	villageFilter   ecs.Filter
	businessFilter  ecs.Filter
	workbenchFilter ecs.Filter
	artisanFilter   ecs.Filter

	employerCache  map[uint64]ecs.Entity
	workbenchCache map[uint64][]workbenchData
}

func NewCraftingSystem(world *ecs.World, pathQueue *engine.PathRequestQueue) *CraftingSystem {
	vID := ecs.ComponentID[components.Village](world)
	tID := ecs.ComponentID[components.TreasuryComponent](world)
	sID := ecs.ComponentID[components.StorageComponent](world)
	idID := ecs.ComponentID[components.Identity](world)

	villageMask := filter.All(vID, tID, sID, idID)

	bID := ecs.ComponentID[components.BusinessComponent](world)
	businessMask := filter.All(bID, tID, sID, idID)

	wbID := ecs.ComponentID[components.WorkbenchComponent](world)
	workbenchMask := filter.All(wbID)

	npcID := ecs.ComponentID[components.NPC](world)
	jID := ecs.ComponentID[components.JobComponent](world)
	pID := ecs.ComponentID[components.Position](world)
	identID := ecs.ComponentID[components.Identity](world)

	artisanMask := filter.All(npcID, jID, pID, identID)

	return &CraftingSystem{
		world:           world,
		pathQueue:       pathQueue,
		villageFilter:   villageMask,
		businessFilter:  businessMask,
		workbenchFilter: workbenchMask,
		artisanFilter:   artisanMask,
		employerCache:   make(map[uint64]ecs.Entity),
		workbenchCache:  make(map[uint64][]workbenchData),
	}
}

func (s *CraftingSystem) Update(world *ecs.World) {
	s.updateCaches()
	s.processArtisans()
}

func (s *CraftingSystem) updateCaches() {
	clear(s.employerCache)
	for k := range s.workbenchCache {
		delete(s.workbenchCache, k)
	}

	idID := ecs.ComponentID[components.Identity](s.world)

	// Cache Villages (entity handles only; components re-fetched at use time)
	qVill := s.world.Query(s.villageFilter)
	for qVill.Next() {
		id := (*components.Identity)(qVill.Get(idID))
		s.employerCache[id.ID] = qVill.Entity()
	}

	// Cache Businesses
	qBus := s.world.Query(s.businessFilter)
	for qBus.Next() {
		id := (*components.Identity)(qBus.Get(idID))
		s.employerCache[id.ID] = qBus.Entity()
	}

	// Cache Workbenches (read-only coordinates copied by value)
	wbID := ecs.ComponentID[components.WorkbenchComponent](s.world)
	qWb := s.world.Query(s.workbenchFilter)
	for qWb.Next() {
		wb := (*components.WorkbenchComponent)(qWb.Get(wbID))
		s.workbenchCache[wb.EmployerID] = append(s.workbenchCache[wb.EmployerID], workbenchData{X: wb.X, Y: wb.Y})
	}
}

func (s *CraftingSystem) processArtisans() {
	jID := ecs.ComponentID[components.JobComponent](s.world)
	pID := ecs.ComponentID[components.Position](s.world)
	identID := ecs.ComponentID[components.Identity](s.world)
	pathID := ecs.ComponentID[components.Path](s.world)
	tID := ecs.ComponentID[components.TreasuryComponent](s.world)
	sID := ecs.ComponentID[components.StorageComponent](s.world)

	qArt := s.world.Query(s.artisanFilter)
	for qArt.Next() {
		job := (*components.JobComponent)(qArt.Get(jID))
		if job.JobID != components.JobArtisan {
			continue
		}

		pos := (*components.Position)(qArt.Get(pID))
		ident := (*components.Identity)(qArt.Get(identID))

		employerEnt, eExists := s.employerCache[job.EmployerID]
		if !eExists {
			continue
		}

		workbenches, wExists := s.workbenchCache[job.EmployerID]
		if !wExists || len(workbenches) == 0 {
			continue
		}

		// Find closest workbench
		var bestWb *workbenchData
		var bestDist float32 = 9999999.0

		for i := range workbenches {
			wb := &workbenches[i]
			dx := pos.X - wb.X
			dy := pos.Y - wb.Y
			distSq := dx*dx + dy*dy
			if distSq < bestDist {
				bestDist = distSq
				bestWb = wb
			}
		}

		if bestWb != nil {
			if bestDist > 1.0 {
				// Move towards workbench
				if s.world.Has(qArt.Entity(), pathID) && s.pathQueue != nil {
					path := (*components.Path)(s.world.Get(qArt.Entity(), pathID))
					if !path.HasPath || path.TargetX != bestWb.X || path.TargetY != bestWb.Y {
						req := engine.PathRequest{
							EntityID: ident.ID,
							StartX:   pos.X,
							StartY:   pos.Y,
							TargetX:  bestWb.X,
							TargetY:  bestWb.Y,
						}
						s.pathQueue.Enqueue(req)
						path.HasPath = true
						path.TargetX = bestWb.X
						path.TargetY = bestWb.Y
					}
				} else {
					// Fallback simple movement
					dx := pos.X - bestWb.X
					dy := pos.Y - bestWb.Y
					if dx > 0 {
						pos.X -= 0.5
					} else if dx < 0 {
						pos.X += 0.5
					}
					if dy > 0 {
						pos.Y -= 0.5
					} else if dy < 0 {
						pos.Y += 0.5
					}
				}
			} else {
				// At workbench, craft! Re-fetch employer components by entity
				// handle — cache values, not component pointers (GC corruption
				// class, see banditry.go).
				if s.world.Alive(employerEnt) {
					storage := (*components.StorageComponent)(s.world.Get(employerEnt, sID))
					if storage.Iron > 0 {
						storage.Iron--
						treasury := (*components.TreasuryComponent)(s.world.Get(employerEnt, tID))
						treasury.Wealth += 50.0 // Value added per craft
					}
				}
			}
		}
	}
}
