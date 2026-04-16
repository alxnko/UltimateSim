package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 59 - The Physical Construction Engine
// ConstructionSystem acts as the real-time bridge between macro-economics and micro-logistics.
// Wealthy villages autonomously generate ConstructionSiteComponent entities.
// JobBuilder NPCs walk to these sites, drain physical resources (Wood/Stone) from the village,
// and progress the site. When complete, it becomes a StructureComponent and increases
// the village's PeakPopulation, directly tying infrastructure to demographics.

type villageConstructionData struct {
	Treasury *components.TreasuryComponent
	Storage  *components.StorageComponent
	Demo     *components.DemographicsComponent
	X        float32
	Y        float32
	Entity   ecs.Entity
	CityID   uint32
}

type ConstructionSystem struct {
	world       *ecs.World
	pathQueue   *engine.PathRequestQueue
	tickCounter uint64

	villageFilter ecs.Filter
	siteFilter    ecs.Filter
	builderFilter ecs.Filter

	villageCache map[uint32]*villageConstructionData
}

func NewConstructionSystem(world *ecs.World, pathQueue *engine.PathRequestQueue) *ConstructionSystem {
	vID := ecs.ComponentID[components.Village](world)
	tID := ecs.ComponentID[components.TreasuryComponent](world)
	sID := ecs.ComponentID[components.StorageComponent](world)
	dID := ecs.ComponentID[components.DemographicsComponent](world)
	pID := ecs.ComponentID[components.Position](world)
	aID := ecs.ComponentID[components.Affiliation](world)

	villageMask := filter.All(vID, tID, sID, dID, pID, aID)

	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	siteMask := filter.All(siteID, pID, aID)

	npcID := ecs.ComponentID[components.NPC](world)
	jID := ecs.ComponentID[components.JobComponent](world)
	identID := ecs.ComponentID[components.Identity](world)

	builderMask := filter.All(npcID, jID, pID, aID, identID)

	return &ConstructionSystem{
		world:         world,
		pathQueue:     pathQueue,
		tickCounter:   0,
		villageFilter: villageMask,
		siteFilter:    siteMask,
		builderFilter: builderMask,
		villageCache:  make(map[uint32]*villageConstructionData),
	}
}

func (s *ConstructionSystem) Update(world *ecs.World) {
	s.tickCounter++

	// 1. Update Village Cache
	s.updateVillageCache()

	// 2. Spawn New Construction Sites (Macro demand)
	if s.tickCounter%100 == 0 {
		s.spawnSites()
	}

	// 3. Process Builder Labor (Micro physical action)
	s.processBuilders()
}

func (s *ConstructionSystem) updateVillageCache() {
	clear(s.villageCache)

	tID := ecs.ComponentID[components.TreasuryComponent](s.world)
	sID := ecs.ComponentID[components.StorageComponent](s.world)
	dID := ecs.ComponentID[components.DemographicsComponent](s.world)
	pID := ecs.ComponentID[components.Position](s.world)
	aID := ecs.ComponentID[components.Affiliation](s.world)

	q := s.world.Query(s.villageFilter)
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(aID))
		treas := (*components.TreasuryComponent)(q.Get(tID))
		stor := (*components.StorageComponent)(q.Get(sID))
		demo := (*components.DemographicsComponent)(q.Get(dID))
		pos := (*components.Position)(q.Get(pID))

		s.villageCache[aff.CityID] = &villageConstructionData{
			Treasury: treas,
			Storage:  stor,
			Demo:     demo,
			X:        pos.X,
			Y:        pos.Y,
			Entity:   q.Entity(),
			CityID:   aff.CityID,
		}
	}
}

func (s *ConstructionSystem) spawnSites() {
	siteCompID := ecs.ComponentID[components.ConstructionSiteComponent](s.world)
	posID := ecs.ComponentID[components.Position](s.world)
	affID := ecs.ComponentID[components.Affiliation](s.world)

	for _, vData := range s.villageCache {
		// If village is wealthy enough, start a construction site
		if vData.Treasury.Wealth > 500 {
			vData.Treasury.Wealth -= 500

			ent := s.world.NewEntity(siteCompID, posID, affID)

			site := (*components.ConstructionSiteComponent)(s.world.Get(ent, siteCompID))
			site.WoodRequired = 50
			site.StoneRequired = 50
			site.MaxProgress = 100
			site.BuilderID = 0 // Unclaimed

			pos := (*components.Position)(s.world.Get(ent, posID))
			pos.X = vData.X + 2.0 // Placed slightly offset from center
			pos.Y = vData.Y + 2.0

			aff := (*components.Affiliation)(s.world.Get(ent, affID))
			aff.CityID = vData.CityID
		}
	}
}

func (s *ConstructionSystem) processBuilders() {
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](s.world)
	structID := ecs.ComponentID[components.StructureComponent](s.world)
	pID := ecs.ComponentID[components.Position](s.world)
	aID := ecs.ComponentID[components.Affiliation](s.world)
	jID := ecs.ComponentID[components.JobComponent](s.world)
	identID := ecs.ComponentID[components.Identity](s.world)
	pathID := ecs.ComponentID[components.Path](s.world)

	// Map of unclaimed sites
	type SiteData struct {
		Entity ecs.Entity
		Site   *components.ConstructionSiteComponent
		X      float32
		Y      float32
	}

	citySites := make(map[uint32][]SiteData)

	// Collect all sites
	qSite := s.world.Query(s.siteFilter)
	var completedSites []ecs.Entity

	for qSite.Next() {
		site := (*components.ConstructionSiteComponent)(qSite.Get(siteID))
		pos := (*components.Position)(qSite.Get(pID))
		aff := (*components.Affiliation)(qSite.Get(aID))

		if site.Progress >= site.MaxProgress {
			completedSites = append(completedSites, qSite.Entity())
			continue
		}

		citySites[aff.CityID] = append(citySites[aff.CityID], SiteData{
			Entity: qSite.Entity(),
			Site:   site,
			X:      pos.X,
			Y:      pos.Y,
		})
	}

	// Process builders
	qBuilder := s.world.Query(s.builderFilter)
	for qBuilder.Next() {
		job := (*components.JobComponent)(qBuilder.Get(jID))
		if job.JobID != components.JobBuilder {
			continue
		}

		aff := (*components.Affiliation)(qBuilder.Get(aID))
		ident := (*components.Identity)(qBuilder.Get(identID))
		pos := (*components.Position)(qBuilder.Get(pID))

		var mySite *SiteData

		// Find site for this builder
		sites := citySites[aff.CityID]
		for i := range sites {
			if sites[i].Site.BuilderID == ident.ID {
				mySite = &sites[i]
				break
			}
		}

		// Claim a new site if we don't have one
		if mySite == nil {
			for i := range sites {
				if sites[i].Site.BuilderID == 0 {
					sites[i].Site.BuilderID = ident.ID
					mySite = &sites[i]
					break
				}
			}
		}

		if mySite != nil {
			// Pathfind to site
			dx := pos.X - mySite.X
			dy := pos.Y - mySite.Y
			distSq := dx*dx + dy*dy

			if distSq > 1.0 {
				// Move towards site
				if s.world.Has(qBuilder.Entity(), pathID) && s.pathQueue != nil {
					path := (*components.Path)(s.world.Get(qBuilder.Entity(), pathID))
					if !path.HasPath || path.TargetX != mySite.X || path.TargetY != mySite.Y {
						req := engine.PathRequest{
							EntityID: ident.ID,
							StartX:   pos.X,
							StartY:   pos.Y,
							TargetX:  mySite.X,
							TargetY:  mySite.Y,
						}
						s.pathQueue.Enqueue(req)
						path.HasPath = true
						path.TargetX = mySite.X
						path.TargetY = mySite.Y
					}
				} else {
					// Fallback simple movement if pathing is unavailable
					if dx > 0 { pos.X -= 0.5 } else if dx < 0 { pos.X += 0.5 }
					if dy > 0 { pos.Y -= 0.5 } else if dy < 0 { pos.Y += 0.5 }
				}
			} else {
				// At site, build!
				vData, vExists := s.villageCache[aff.CityID]
				if !vExists { continue }

				// Drain resources
				if mySite.Site.WoodGathered < mySite.Site.WoodRequired && vData.Storage.Wood > 0 {
					vData.Storage.Wood--
					mySite.Site.WoodGathered++
				} else if mySite.Site.StoneGathered < mySite.Site.StoneRequired && vData.Storage.Stone > 0 {
					vData.Storage.Stone--
					mySite.Site.StoneGathered++
				} else if mySite.Site.WoodGathered >= mySite.Site.WoodRequired && mySite.Site.StoneGathered >= mySite.Site.StoneRequired {
					// Hammer progress
					mySite.Site.Progress++
				}
			}
		}
	}

	// Transform completed sites
	for _, ent := range completedSites {
		s.world.Remove(ent, siteID)
		s.world.Add(ent, structID)

		str := (*components.StructureComponent)(s.world.Get(ent, structID))
		str.StructureType = uint32(components.StructureHouse)
		str.Integrity = 100.0

		// Boost demographics
		if s.world.Has(ent, aID) {
			aff := (*components.Affiliation)(s.world.Get(ent, aID))
			if vData, exists := s.villageCache[aff.CityID]; exists {
				// Each new house increases peak population
				vData.Demo.PeakPopulation += 50
			}
		}
	}
}
