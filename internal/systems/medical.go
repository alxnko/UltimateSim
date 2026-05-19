package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 68 - The Physical Medical Engine
// Bridges Biology (Vitals), Economy (Wealth), and Justice (Negative Hooks).

type patientData struct {
	entity ecs.Entity
	id     uint64
	x      float32
	y      float32
}

type MedicalSystem struct {
	patientFilter ecs.Filter
	doctorFilter  ecs.Filter
	hookGraph     *engine.SparseHookGraph
	patients      []patientData
}

func (s *MedicalSystem) IsExpensive() bool {
	return true
}

func NewMedicalSystem(world *ecs.World, hooks *engine.SparseHookGraph) *MedicalSystem {
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	idID := ecs.ComponentID[components.Identity](world)

	jobID := ecs.ComponentID[components.JobComponent](world)

	return &MedicalSystem{
		patientFilter: filter.All(posID, vitalsID, needsID, idID),
		doctorFilter:  filter.All(posID, jobID),
		hookGraph:     hooks,
		patients:      make([]patientData, 0, 100),
	}
}

func (s *MedicalSystem) Update(world *ecs.World) {
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	idID := ecs.ComponentID[components.Identity](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	pathID := ecs.ComponentID[components.Path](world)

	s.patients = s.patients[:0]

	// 1. Pre-cache all injured patients
	patientQuery := world.Query(s.patientFilter)
	for patientQuery.Next() {
		vitals := (*components.VitalsComponent)(patientQuery.Get(vitalsID))

		// Determine if entity needs medical attention
		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(patientQuery.Get(posID))
			ident := (*components.Identity)(patientQuery.Get(idID))

			s.patients = append(s.patients, patientData{
				entity: patientQuery.Entity(),
				id:     ident.ID,
				x:      pos.X,
				y:      pos.Y,
			})
		}
	}

	if len(s.patients) == 0 {
		return // Fast exit if no patients
	}

	// 2. Iterate doctors
	doctorQuery := world.Query(s.doctorFilter)

	for doctorQuery.Next() {
		job := (*components.JobComponent)(doctorQuery.Get(jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		dPos := (*components.Position)(doctorQuery.Get(posID))

		var path *components.Path
		if doctorQuery.Has(pathID) {
			path = (*components.Path)(doctorQuery.Get(pathID))
		}

		var dIdent *components.Identity
		if doctorQuery.Has(idID) {
			dIdent = (*components.Identity)(doctorQuery.Get(idID))
		}

		// Find closest patient
		var bestPatient *patientData
		var bestDist float32 = 999999.0

		for i := 0; i < len(s.patients); i++ {
			p := &s.patients[i]

			// Skip if this patient was already healed in this tick by another doctor
			if !world.Alive(p.entity) {
				continue
			}

			dx := dPos.X - p.x
			dy := dPos.Y - p.y
			distSq := (dx * dx) + (dy * dy)

			if distSq < 2.0 {
				// Adjacency reached: Heal the patient!

				// Re-fetch component pointers (ECS safe because we are querying different entities but wait, we are inside a query!
				// Arche allows world.Get for OTHER entities during a query as long as we don't modify structure.)
				pVitals := (*components.VitalsComponent)(world.Get(p.entity, vitalsID))
				pNeeds := (*components.Needs)(world.Get(p.entity, needsID))

				// Apply medical treatment
				pVitals.Blood = 100.0
				pVitals.Pain = 0.0
				pVitals.Consciousness = 100.0

				// Economy: Medical Fee
				fee := float32(100.0)

				if pNeeds.Wealth >= fee {
					pNeeds.Wealth -= fee
					// Doctor gets paid
					if doctorQuery.Has(needsID) {
						dNeeds := (*components.Needs)(doctorQuery.Get(needsID))
						dNeeds.Wealth += fee
					}
				} else {
					// Patient cannot afford it!
					collected := pNeeds.Wealth
					pNeeds.Wealth = 0
					if doctorQuery.Has(needsID) {
						dNeeds := (*components.Needs)(doctorQuery.Get(needsID))
						dNeeds.Wealth += collected
					}

					// Justice/Social Consequence: Medical Debt Resentment
					if s.hookGraph != nil && dIdent != nil {
						// Doctor holds a deep negative hook against the patient for not paying
						s.hookGraph.AddHook(dIdent.ID, p.id, -50)
					}
				}

				// Remove patient from needing care this tick (hacky but works since we don't pop array)
				p.entity = ecs.Entity{} // Invalidate so other doctors don't target
				bestPatient = nil
				break
			}

			if distSq < bestDist {
				bestDist = distSq
				bestPatient = p
			}
		}

		// Pathfind to best patient
		if bestPatient != nil && path != nil {
			path.TargetX = bestPatient.x
			path.TargetY = bestPatient.y
			path.HasPath = false // Trigger movement system repath
		}
	}
}
