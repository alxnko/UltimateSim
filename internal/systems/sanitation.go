package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 65: The Physical Sanitation Engine
// Bridges Death, Psychology (Stress), Biology (Disease), and Logistics (Gravedigging).

type SanitationSystem struct {
	corpseFilter ecs.Filter
	sanityFilter ecs.Filter
	diggerFilter ecs.Filter

	corpses []corpseData
}

type corpseData struct {
	entity ecs.Entity
	x      float32
	y      float32
}

func (s *SanitationSystem) IsExpensive() bool {
	return true
}

func NewSanitationSystem(world *ecs.World) *SanitationSystem {
	posID := ecs.ComponentID[components.Position](world)
	corpseID := ecs.ComponentID[components.CorpseComponent](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	jobID := ecs.ComponentID[components.JobComponent](world)

	return &SanitationSystem{
		corpseFilter: filter.All(posID, corpseID),
		sanityFilter: filter.All(posID, sanityID),
		diggerFilter: filter.All(posID, jobID),
		corpses:      make([]corpseData, 0, 100),
	}
}

func (s *SanitationSystem) Update(world *ecs.World) {
	posID := ecs.ComponentID[components.Position](world)
	corpseID := ecs.ComponentID[components.CorpseComponent](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	diseaseID := ecs.ComponentID[components.DiseaseEntity](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	pathID := ecs.ComponentID[components.Path](world)

	s.corpses = s.corpses[:0]

	// 1. Pre-cache all corpses and process biological decay
	corpseQuery := world.Query(s.corpseFilter)
	var toRemoveCorpses []ecs.Entity
	var newDiseases []corpseData

	for corpseQuery.Next() {
		pos := (*components.Position)(corpseQuery.Get(posID))
		corpse := (*components.CorpseComponent)(corpseQuery.Get(corpseID))

		corpse.DecayProgress += 1.0

		if corpse.DecayProgress >= corpse.MaxDecay {
			toRemoveCorpses = append(toRemoveCorpses, corpseQuery.Entity())
			newDiseases = append(newDiseases, corpseData{
				x: pos.X,
				y: pos.Y,
			})
		} else {
			s.corpses = append(s.corpses, corpseData{
				entity: corpseQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
			})
		}
	}

	// Process decay removals and spawn diseases outside of query lock
	for _, e := range toRemoveCorpses {
		world.RemoveEntity(e)
	}

	for _, d := range newDiseases {
		diseaseEnt := world.NewEntity(posID, diseaseID)

		dPos := (*components.Position)(world.Get(diseaseEnt, posID))
		dPos.X = d.x
		dPos.Y = d.y

		disease := (*components.DiseaseEntity)(world.Get(diseaseEnt, diseaseID))
		disease.ID = uint32(engine.GetRandomInt())
		disease.Lethality = uint8(75 + (engine.GetRandomInt() % 26)) // 75 to 100
	}

	// If no active corpses remain, exit early
	if len(s.corpses) == 0 {
		return
	}

	// 2. Psychological Impact: Add stress to nearby living NPCs
	sanityQuery := world.Query(s.sanityFilter)
	for sanityQuery.Next() {
		pos := (*components.Position)(sanityQuery.Get(posID))
		sanity := (*components.SanityComponent)(sanityQuery.Get(sanityID))

		for i := 0; i < len(s.corpses); i++ {
			c := &s.corpses[i]
			dx := pos.X - c.x
			dy := pos.Y - c.y
			distSq := (dx * dx) + (dy * dy)

			// 5 tile radius for smelling/seeing rot
			if distSq <= 25.0 {
				sanity.Stress += 0.5 // Continually adds stress if lingering
				break // One corpse is enough to cause stress this tick
			}
		}
	}

	// 3. Logistical Counter: Gravediggers destroy corpses
	diggerQuery := world.Query(s.diggerFilter)
	var diggerDestroyedCorpses []ecs.Entity

	for diggerQuery.Next() {
		job := (*components.JobComponent)(diggerQuery.Get(jobID))
		if job.JobID != components.JobGravedigger {
			continue
		}

		dPos := (*components.Position)(diggerQuery.Get(posID))

		var path *components.Path
		if diggerQuery.Has(pathID) {
			path = (*components.Path)(diggerQuery.Get(pathID))
		}

		var bestCorpse *corpseData
		var bestDist float32 = 999999.0

		for i := 0; i < len(s.corpses); i++ {
			c := &s.corpses[i]

			// Skip if already flagged for destruction
			alreadyDestroyed := false
			for _, e := range diggerDestroyedCorpses {
				if e == c.entity {
					alreadyDestroyed = true
					break
				}
			}
			if alreadyDestroyed {
				continue
			}

			dx := dPos.X - c.x
			dy := dPos.Y - c.y
			distSq := (dx * dx) + (dy * dy)

			if distSq < 2.0 {
				// Destroy it
				diggerDestroyedCorpses = append(diggerDestroyedCorpses, c.entity)
				bestCorpse = nil
				break
			}

			if distSq < bestDist {
				bestDist = distSq
				bestCorpse = c
			}
		}

		if bestCorpse != nil && path != nil {
			path.TargetX = bestCorpse.x
			path.TargetY = bestCorpse.y
			path.HasPath = false // Trigger repathing
		}
	}

	// Apply digger destructions
	for _, e := range diggerDestroyedCorpses {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}
}
