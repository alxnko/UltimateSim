package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 63 - The Refugee Crisis Engine
// Scans for desperate NPCs. If their desperation reaches a critical threshold
// due to localized economic collapse or famine, they sever their ties to their
// current Affiliation.CityID, gain an AsylumSeekerComponent, and pathfind to
// the nearest prosperous Village to beg for integration.

type villageRefugeeData struct {
	CityID  uint32
	X       float32
	Y       float32
	Storage *components.StorageComponent
}

type RefugeeSystem struct {
	pathQueue   *engine.PathRequestQueue
	tickCounter uint64

	villageFilter ecs.Filter
	npcFilter     ecs.Filter

	villageCache []villageRefugeeData
}

func NewRefugeeSystem(world *ecs.World, pathQueue *engine.PathRequestQueue) *RefugeeSystem {
	vID := ecs.ComponentID[components.Village](world)
	pID := ecs.ComponentID[components.Position](world)
	aID := ecs.ComponentID[components.Affiliation](world)
	sID := ecs.ComponentID[components.StorageComponent](world)

	villageMask := filter.All(vID, pID, aID, sID)

	npcID := ecs.ComponentID[components.NPC](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	// We want to query all NPCs with Desperation, Identity, Affiliation, Position.
	// We do not strictly need Path in the filter, but we need to fetch it or add it.

	npcMask := filter.All(npcID, despID, identID, pID, aID)

	return &RefugeeSystem{
		pathQueue:     pathQueue,
		tickCounter:   0,
		villageFilter: villageMask,
		npcFilter:     npcMask,
		villageCache:  make([]villageRefugeeData, 0, 100),
	}
}

func (s *RefugeeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle to avoid excessive pathfinding calculations every tick
	if s.tickCounter%30 != 0 {
		return
	}

	pID := ecs.ComponentID[components.Position](world)
	aID := ecs.ComponentID[components.Affiliation](world)
	sID := ecs.ComponentID[components.StorageComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	asylumID := ecs.ComponentID[components.AsylumSeekerComponent](world)
	pathID := ecs.ComponentID[components.Path](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	bizID := ecs.ComponentID[components.BusinessComponent](world)

	// Step 1: Cache all active, prosperous villages (Food > 100)
	s.villageCache = s.villageCache[:0]
	qV := world.Query(s.villageFilter)
	for qV.Next() {
		stor := (*components.StorageComponent)(qV.Get(sID))
		if stor.Food > 100 {
			pos := (*components.Position)(qV.Get(pID))
			aff := (*components.Affiliation)(qV.Get(aID))
			s.villageCache = append(s.villageCache, villageRefugeeData{
				CityID:  aff.CityID,
				X:       pos.X,
				Y:       pos.Y,
				Storage: stor,
			})
		}
	}

	if len(s.villageCache) == 0 {
		return // Nowhere to run
	}

	// Step 2: Iterate all NPCs
	qN := world.Query(s.npcFilter)

	type refugeeAction struct {
		Entity ecs.Entity
		Target villageRefugeeData
	}

	var toBecomeRefugees []refugeeAction
	var toIntegrate []ecs.Entity

	for qN.Next() {
		ent := qN.Entity()
		pos := (*components.Position)(qN.Get(pID))
		aff := (*components.Affiliation)(qN.Get(aID))

		// Are they already a refugee?
		if world.Has(ent, asylumID) {
			asylum := (*components.AsylumSeekerComponent)(world.Get(ent, asylumID))

			// Check if they have reached their destination
			dx := pos.X - asylum.TargetX
			dy := pos.Y - asylum.TargetY
			distSq := dx*dx + dy*dy


			if distSq < 2.0 {
				toIntegrate = append(toIntegrate, ent)
			} else {
				// Re-path if necessary (lost path)
				if world.Has(ent, pathID) && s.pathQueue != nil {
					path := (*components.Path)(world.Get(ent, pathID))
					if !path.HasPath || path.TargetX != asylum.TargetX || path.TargetY != asylum.TargetY {
						ident := (*components.Identity)(qN.Get(identID))
						req := engine.PathRequest{
							EntityID: ident.ID,
							StartX:   pos.X,
							StartY:   pos.Y,
							TargetX:  asylum.TargetX,
							TargetY:  asylum.TargetY,
						}
						s.pathQueue.Enqueue(req)
						path.HasPath = true
						path.TargetX = asylum.TargetX
						path.TargetY = asylum.TargetY
					}
				}
			}
			continue
		}

		// Not a refugee. Should they become one?
		desp := (*components.DesperationComponent)(qN.Get(despID))
		if desp.Level >= 80 {
			// Find nearest prosperous village that is NOT their current city
			var bestV villageRefugeeData
			var bestDist float32 = 9999999.0
			found := false

			for _, v := range s.villageCache {
				if v.CityID == aff.CityID {
					continue
				}

				dx := pos.X - v.X
				dy := pos.Y - v.Y
				distSq := dx*dx + dy*dy

				if distSq < bestDist {
					bestDist = distSq
					bestV = v
					found = true
				}
			}

			if found {
				toBecomeRefugees = append(toBecomeRefugees, refugeeAction{
					Entity: ent,
					Target: bestV,
				})
			}
		}
	}

	// Step 3: Apply structural changes outside the query loop
	for _, action := range toBecomeRefugees {
		if !world.Alive(action.Entity) {
			continue
		}

		// Sever ties to old city
		aff := (*components.Affiliation)(world.Get(action.Entity, aID))
		aff.CityID = 0

		// They quit their job / close business
		if world.Has(action.Entity, jobID) {
			world.Remove(action.Entity, jobID)
		}
		if world.Has(action.Entity, bizID) {
			world.Remove(action.Entity, bizID)
		}

		// Add AsylumSeekerComponent
		if !world.Has(action.Entity, asylumID) {
			world.Add(action.Entity, asylumID)
		}
		asylum := (*components.AsylumSeekerComponent)(world.Get(action.Entity, asylumID))
		asylum.TargetCityID = action.Target.CityID
		asylum.TargetX = action.Target.X
		asylum.TargetY = action.Target.Y

		// Ensure they have a Path component
		if !world.Has(action.Entity, pathID) {
			world.Add(action.Entity, pathID)
		}
		path := (*components.Path)(world.Get(action.Entity, pathID))
		path.HasPath = true
		path.TargetX = action.Target.X
		path.TargetY = action.Target.Y

		// Enqueue path request
		ident := (*components.Identity)(world.Get(action.Entity, identID))
		pos := (*components.Position)(world.Get(action.Entity, pID))
		if s.pathQueue != nil {
			req := engine.PathRequest{
				EntityID: ident.ID,
				StartX:   pos.X,
				StartY:   pos.Y,
				TargetX:  action.Target.X,
				TargetY:  action.Target.Y,
			}
			s.pathQueue.Enqueue(req)
		}
	}

	for _, ent := range toIntegrate {
		if !world.Alive(ent) {
			continue
		}

		// Read Asylum Target City
		asylum := (*components.AsylumSeekerComponent)(world.Get(ent, asylumID))
		targetCity := asylum.TargetCityID

		// Remove AsylumSeekerComponent
		world.Remove(ent, asylumID)

		// Integrate into new city
		aff := (*components.Affiliation)(world.Get(ent, aID))
		aff.CityID = targetCity

		// Reset desperation
		desp := (*components.DesperationComponent)(world.Get(ent, despID))
		desp.Level = 0

		// Clear path
		if world.Has(ent, pathID) {
			path := (*components.Path)(world.Get(ent, pathID))
			path.HasPath = false
		}
	}
}
