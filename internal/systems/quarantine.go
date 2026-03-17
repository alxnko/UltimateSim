package systems

import (



	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"

)

type diseaseData struct {
	X float32
	Y float32
}




// Phase 37.1: The Quarantine Engine
// QuarantineSystem scours all active jurisdictions for DiseaseEntities within their borders.
// If a Plague is active inside a Jurisdiction, it forcefully enables a QuarantineComponent lock.

type QuarantineSystem struct {
	activeDiseases []diseaseData
	tickCounter uint64
	toRemove    []ecs.Entity
	toAdd       []ecs.Entity
}

func NewQuarantineSystem(world *ecs.World) *QuarantineSystem {




	return &QuarantineSystem{
		tickCounter: 0,
		toRemove:    make([]ecs.Entity, 0, 50),




		toAdd:       make([]ecs.Entity, 0, 50),




		activeDiseases: make([]diseaseData, 0, 100),




	}
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *QuarantineSystem) IsExpensive() bool {





	return true
}

// IsNonEssential returns true to skip this system during fast-forward.
func (s *QuarantineSystem) IsNonEssential() bool {





	return true
}

func (s *QuarantineSystem) Update(world *ecs.World) {





	s.tickCounter++

	// Run every 20 ticks to reduce massive O(N^2) overhead





	if s.tickCounter%20 != 0 {
		return
	}

	// 1. Map all active DiseaseEntities into a flat array for O(1) distance checks





	diseasePosID := ecs.ComponentID[components.Position](world)




	diseaseEntID := ecs.ComponentID[components.DiseaseEntity](world)





	s.activeDiseases = s.activeDiseases[:0]
	diseaseFilter := ecs.All(diseasePosID, diseaseEntID)




	diseaseQuery := world.Query(diseaseFilter)






	for diseaseQuery.Next() {





		pos := (*components.Position)(diseaseQuery.Get(diseasePosID))




		s.activeDiseases = append(s.activeDiseases, diseaseData{
			X: pos.X,
			Y: pos.Y,
		})




	}

	// Fast exit if no diseases are present
	if len(s.activeDiseases) == 0 {





		// We still need to clear any existing quarantines
		jurID := ecs.ComponentID[components.JurisdictionComponent](world)




		quarID := ecs.ComponentID[components.QuarantineComponent](world)




		quarQuery := world.Query(ecs.All(jurID, quarID))





		s.toRemove = s.toRemove[:0]
		for quarQuery.Next() {





			s.toRemove = append(s.toRemove, quarQuery.Entity())




		}
		for _, e := range s.toRemove {
			if world.Alive(e) && world.Has(e, quarID) {





				world.Remove(e, quarID)




			}
		}
		return
	}

	// 2. Evaluate all Jurisdictions
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)




	posID := ecs.ComponentID[components.Position](world)




	quarID := ecs.ComponentID[components.QuarantineComponent](world)





	jurQuery := world.Query(ecs.All(jurID, posID))





	s.toAdd = s.toAdd[:0]
	s.toRemove = s.toRemove[:0]

	for jurQuery.Next() {





		entity := jurQuery.Entity()




		jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))




		pos := (*components.Position)(jurQuery.Get(posID))





		hasDisease := false

		// Check if any disease falls within this jurisdiction's radius
		for i := 0; i < len(s.activeDiseases); i++ {




			d := &s.activeDiseases[i]
			dx := pos.X - d.X
			dy := pos.Y - d.Y
			distSq := (dx * dx) + (dy * dy)






			if distSq <= jur.RadiusSquared {
				hasDisease = true
				break
			}
		}

		hasQuarantine := jurQuery.Has(quarID)





		if hasDisease && !hasQuarantine {
			// Enact Quarantine
			s.toAdd = append(s.toAdd, entity)




		} else if !hasDisease && hasQuarantine {
			// Lift Quarantine
			s.toRemove = append(s.toRemove, entity)




		}
	}

	// 3. Apply structural changes outside the query loop
	for _, e := range s.toAdd {
		if world.Alive(e) {





			world.Add(e, quarID)




			quar := (*components.QuarantineComponent)(world.Get(e, quarID))




			quar.Active = true
		}
	}

	for _, e := range s.toRemove {
		if world.Alive(e) && world.Has(e, quarID) {





			world.Remove(e, quarID)




		}
	}
}
