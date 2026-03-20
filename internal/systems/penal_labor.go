package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 45 - The Penal Labor Engine
// PenalLaborSystem enforces state servitude. It iterates over entities with PenalLaborComponent,
// harvests resources for the StateCityID, and generates abolitionist backlash.

type PenalLaborSystem struct {
	penalFilter ecs.Filter
	cityFilter  ecs.Filter
	hookGraph   *engine.SparseHookGraph
	tickCounter uint64
}

func NewPenalLaborSystem(world *ecs.World, hooks *engine.SparseHookGraph) *PenalLaborSystem {
	penalID := ecs.ComponentID[components.PenalLaborComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	idID := ecs.ComponentID[components.Identity](world)

	pMask := ecs.All(penalID, posID, idID)

	cityID := ecs.ComponentID[components.Village](world) // Assuming Villages act as StateCityID
	cPosID := ecs.ComponentID[components.Position](world)
	cAffID := ecs.ComponentID[components.Affiliation](world)
	cStorID := ecs.ComponentID[components.StorageComponent](world)

	cMask := ecs.All(cityID, cPosID, cAffID, cStorID)

	return &PenalLaborSystem{
		penalFilter: &pMask,
		cityFilter:  &cMask,
		hookGraph:   hooks,
	}
}

func (s *PenalLaborSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Pre-cache all City/State storage pointers to avoid nested queries
	cAffID := ecs.ComponentID[components.Affiliation](world)
	cStorID := ecs.ComponentID[components.StorageComponent](world)
	cIdentID := ecs.ComponentID[components.Identity](world) // To get ruler for hooks
	adminID := ecs.ComponentID[components.AdministrationMarker](world)

	cityQuery := world.Query(s.cityFilter)
	type cityData struct {
		Storage *components.StorageComponent
		RulerID uint64
	}

	cities := make(map[uint32]cityData)

	for cityQuery.Next() {
		aff := (*components.Affiliation)(cityQuery.Get(cAffID))
		stor := (*components.StorageComponent)(cityQuery.Get(cStorID))

		// Find Ruler ID (either the Capital's own ID or an emergent AdministrationMarker)
		var rulerID uint64 = 0
		if world.Has(cityQuery.Entity(), cIdentID) {
			ruler := (*components.Identity)(cityQuery.Get(cIdentID))
			rulerID = ruler.ID // Default to city identity ID
		}

		cities[aff.CityID] = cityData{
			Storage: stor,
			RulerID: rulerID,
		}
	}

	// Now we also need to find the emergent rulers explicitly, as they hold the AdministrationMarker, not the Village entity.
	adminQuery := world.Query(ecs.All(adminID, cAffID, cIdentID))
	for adminQuery.Next() {
		aff := (*components.Affiliation)(adminQuery.Get(cAffID))
		ident := (*components.Identity)(adminQuery.Get(cIdentID))

		if cd, exists := cities[aff.CityID]; exists {
			cd.RulerID = ident.ID
			cities[aff.CityID] = cd
		}
	}

	// Step 2: Iterate over all convicts
	penalID := ecs.ComponentID[components.PenalLaborComponent](world)
	idID := ecs.ComponentID[components.Identity](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	posID := ecs.ComponentID[components.Position](world)

	penalQuery := world.Query(s.penalFilter)

	var finishedConvicts []ecs.Entity

	// We cache positions for Abolitionist hook generation
	type laborEvent struct {
		X float32
		Y float32
		RulerID uint64
	}
	var laborEvents []laborEvent

	for penalQuery.Next() {
		ent := penalQuery.Entity()
		penal := (*components.PenalLaborComponent)(penalQuery.Get(penalID))

		// Process labor
		if penal.RemainingSentence > 0 {
			penal.RemainingSentence--

			// Provide minimal metabolism sustenance
			if world.Has(ent, needsID) {
				needs := (*components.Needs)(world.Get(ent, needsID))
				needs.Food += 0.5 // Bare minimum to survive, completely suppressing free market demand
			}

			// Generate state resources (Free Labor driving down market wages)
			if cd, exists := cities[penal.StateCityID]; exists {
				cd.Storage.Stone += 1.0 // Forced quarrying

				// Record event for social backlash
				if s.tickCounter % 10 == 0 && cd.RulerID != 0 {
					pos := (*components.Position)(penalQuery.Get(posID))
					laborEvents = append(laborEvents, laborEvent{
						X: pos.X,
						Y: pos.Y,
						RulerID: cd.RulerID,
					})
				}
			}
		}

		if penal.RemainingSentence == 0 {
			finishedConvicts = append(finishedConvicts, ent)
			if world.Has(ent, jobID) {
				job := (*components.JobComponent)(world.Get(ent, jobID))
				job.JobID = components.JobNone // Released
			}
		}
	}

	// Structural removal
	for _, e := range finishedConvicts {
		if world.Alive(e) && world.Has(e, penalID) {
			world.Remove(e, penalID)
		}
	}

	// Step 3: Social Backlash (Abolitionists)
	if len(laborEvents) > 0 && s.hookGraph != nil {
		abolitionistQuery := world.Query(ecs.All(idID, posID))

		for abolitionistQuery.Next() {
			ident := (*components.Identity)(abolitionistQuery.Get(idID))

			if ident.BaseTraits & components.TraitAbolitionist != 0 {
				pos := (*components.Position)(abolitionistQuery.Get(posID))

				// Check distance to any labor event
				for _, event := range laborEvents {
					dx := pos.X - event.X
					dy := pos.Y - event.Y
					distSq := dx*dx + dy*dy

					if distSq < 100.0 { // Witnessed penal labor
						// Generate massive negative grudge against the State Ruler
						s.hookGraph.AddHook(ident.ID, event.RulerID, -50)
					}
				}
			}
		}
	}
}
