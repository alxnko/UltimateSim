package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 63: The Refugee Crisis Engine
// Desperate NPCs abandon their homes to seek asylum in wealthy cities.

type cityData struct {
	Entity ecs.Entity
	CityID uint32
	X      float32
	Y      float32
	Wealth float32
}

type RefugeeSystem struct {
	npcFilter  ecs.Filter
	cityFilter ecs.Filter
	toAdd      []ecs.Entity
	toRemove   []ecs.Entity
	targetIDs  []uint32
	cities     []cityData
}

func NewRefugeeSystem(world *ecs.World) *RefugeeSystem {
	npcMask := ecs.All(
		ecs.ComponentID[components.Position](world),
		ecs.ComponentID[components.Needs](world),
		ecs.ComponentID[components.DesperationComponent](world),
		ecs.ComponentID[components.Affiliation](world),
	).Without(ecs.ComponentID[components.AsylumSeekerComponent](world))

	cityMask := ecs.All(
		ecs.ComponentID[components.Village](world),
		ecs.ComponentID[components.Position](world),
		ecs.ComponentID[components.Affiliation](world),
		ecs.ComponentID[components.TreasuryComponent](world),
	)

	return &RefugeeSystem{
		npcFilter:  &npcMask,
		cityFilter: &cityMask,
		toAdd:      make([]ecs.Entity, 0, 100),
		toRemove:   make([]ecs.Entity, 0, 100),
		targetIDs:  make([]uint32, 0, 100),
		cities:     make([]cityData, 0, 50),
	}
}

func (s *RefugeeSystem) Update(world *ecs.World) {
	s.toAdd = s.toAdd[:0]
	s.toRemove = s.toRemove[:0]
	s.targetIDs = s.targetIDs[:0]
	s.cities = s.cities[:0]

	posID := ecs.ComponentID[components.Position](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	treasID := ecs.ComponentID[components.TreasuryComponent](world)

	// 1. Pre-cache wealthy cities
	cq := world.Query(s.cityFilter)
	for cq.Next() {
		treas := (*components.TreasuryComponent)(cq.Get(treasID))
		if treas.Wealth > 100.0 { // Threshold for prosperous city
			pos := (*components.Position)(cq.Get(posID))
			aff := (*components.Affiliation)(cq.Get(affilID))
			s.cities = append(s.cities, cityData{
				Entity: cq.Entity(),
				CityID: aff.CityID,
				X:      pos.X,
				Y:      pos.Y,
				Wealth: treas.Wealth,
			})
		}
	}

	if len(s.cities) == 0 {
		return // Nowhere to run
	}

	// 2. Identify new refugees
	needsID := ecs.ComponentID[components.Needs](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	jobID := ecs.ComponentID[components.JobComponent](world)

	nq := world.Query(s.npcFilter)
	for nq.Next() {
		desp := (*components.DesperationComponent)(nq.Get(despID))
		needs := (*components.Needs)(nq.Get(needsID))
		aff := (*components.Affiliation)(nq.Get(affilID))

		// Ensure not already a bandit or guard
		if world.Has(nq.Entity(), jobID) {
			job := (*components.JobComponent)(nq.Get(jobID))
			if job.JobID == components.JobBandit || job.JobID == components.JobGuard || job.JobID == components.JobPenalLabor {
				continue
			}
		}

		if desp.Level >= 80 && needs.Wealth < 10.0 {
			pos := (*components.Position)(nq.Get(posID))

			// Find nearest wealthy city
			var bestCity *cityData
			var bestDist float32 = 999999.0

			for i := range s.cities {
				c := &s.cities[i]
				if c.CityID == aff.CityID {
					continue // Don't run to own city
				}
				dx := pos.X - c.X
				dy := pos.Y - c.Y
				distSq := dx*dx + dy*dy
				if distSq < bestDist {
					bestDist = distSq
					bestCity = c
				}
			}

			if bestCity != nil {
				s.toAdd = append(s.toAdd, nq.Entity())
				s.targetIDs = append(s.targetIDs, bestCity.CityID)

				// State abandonment: clear CityID while querying
				aff.CityID = 0
			}
		}
	}

	// 3. Process existing refugees (Assimilation)
	asylumID := ecs.ComponentID[components.AsylumSeekerComponent](world)
	pathID := ecs.ComponentID[components.Path](world)

	refMask := ecs.All(posID, asylumID, affilID)
	rq := world.Query(refMask)
	for rq.Next() {
		pos := (*components.Position)(rq.Get(posID))
		asylum := (*components.AsylumSeekerComponent)(rq.Get(asylumID))

		asylum.TicksFleeing++

		var targetCity *cityData
		for i := range s.cities {
			if s.cities[i].CityID == asylum.TargetCityID {
				targetCity = &s.cities[i]
				break
			}
		}

		if targetCity != nil {
			dx := pos.X - targetCity.X
			dy := pos.Y - targetCity.Y
			distSq := dx*dx + dy*dy

			if distSq < 4.0 { // Reached destination
				aff := (*components.Affiliation)(rq.Get(affilID))
				aff.CityID = targetCity.CityID
				s.toRemove = append(s.toRemove, rq.Entity())

				// Natively trigger Xenophobia check if applicable
				// This relies on the core simulation loop correctly identifying
				// the NPC's new presence in the city during XenophobiaSystem updates.
			} else {
				// Pathfind towards target
				if world.Has(rq.Entity(), pathID) {
					path := (*components.Path)(rq.Get(pathID))
					if path.TargetX != targetCity.X || path.TargetY != targetCity.Y {
						path.TargetX = targetCity.X
						path.TargetY = targetCity.Y
						path.HasPath = false // force recalculation
					}
				}
			}
		} else {
			// Target city destroyed or no longer wealthy, abandon asylum seeking for now
			s.toRemove = append(s.toRemove, rq.Entity())
		}
	}

	// Structural modifications
	for i, e := range s.toAdd {
		if world.Alive(e) {
			world.Add(e, asylumID)
			asc := (*components.AsylumSeekerComponent)(world.Get(e, asylumID))
			asc.TargetCityID = s.targetIDs[i]
			asc.TicksFleeing = 0
		}
	}

	for _, e := range s.toRemove {
		if world.Alive(e) {
			world.Remove(e, asylumID)
		}
	}
}
