package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 68: The Physical Medical Engine
// Bridges Biology (Vitals), Economy (Wealth), and Social Hierarchy (Hooks).
// Doctors physically pathfind to injured NPCs, heal them, and charge fees.
// If the patient is bankrupt, generates negative social hooks.

type MedicalSystem struct {
	doctorFilter  ecs.Filter
	patientFilter ecs.Filter

	jobID    ecs.ID
	posID    ecs.ID
	vitalsID ecs.ID
	needsID  ecs.ID
	identID  ecs.ID

	pathQueue *engine.PathRequestQueue
	hooks     *engine.SparseHookGraph
}

func NewMedicalSystem(world *ecs.World, pathQueue *engine.PathRequestQueue, hooks *engine.SparseHookGraph) *MedicalSystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	identID := ecs.ComponentID[components.Identity](world)

	dMask := ecs.All(jobID, posID, needsID, identID)
	pMask := ecs.All(posID, vitalsID, needsID, identID)

	return &MedicalSystem{
		doctorFilter:  &dMask,
		patientFilter: &pMask,
		jobID:         jobID,
		posID:         posID,
		vitalsID:      vitalsID,
		needsID:       needsID,
		identID:       identID,
		pathQueue:     pathQueue,
		hooks:         hooks,
	}
}

type patientData struct {
	entity ecs.Entity
	x      float32
	y      float32
	id     uint64
}

func (s *MedicalSystem) Update(world *ecs.World) {
	// Pre-cache injured patients to avoid locking world during iterations
	var injuredPatients []patientData

	pQuery := world.Query(s.patientFilter)
	for pQuery.Next() {
		vitals := (*components.VitalsComponent)(pQuery.Get(s.vitalsID))

		// Determine if treatment is needed
		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))

			injuredPatients = append(injuredPatients, patientData{
				entity: pQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				id:     ident.ID,
			})
		}
	}

	if len(injuredPatients) == 0 {
		return // No one to heal
	}

	dQuery := world.Query(s.doctorFilter)
	for dQuery.Next() {
		job := (*components.JobComponent)(dQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		dPos := (*components.Position)(dQuery.Get(s.posID))
		dIdent := (*components.Identity)(dQuery.Get(s.identID))
		dNeeds := (*components.Needs)(dQuery.Get(s.needsID))

		// Find the closest injured patient
		var bestPatient *patientData
		var bestDistSq float32 = 9999999.0

		for i := range injuredPatients {
			p := &injuredPatients[i]

			// Re-verify patient is still alive (in case another system killed them)
			if !world.Alive(p.entity) {
				continue
			}

			dx := dPos.X - p.x
			dy := dPos.Y - p.y
			distSq := dx*dx + dy*dy

			if distSq < bestDistSq {
				bestDistSq = distSq
				bestPatient = p
			}
		}

		if bestPatient == nil {
			continue
		}

		// If adjacent, execute treatment
		if bestDistSq <= 2.0 {
			// Must use world.Get to safely access patient components since we are in a doctor query loop
			// It is safe because we are just modifying values, not doing structural changes.
			pVitals := (*components.VitalsComponent)(world.Get(bestPatient.entity, s.vitalsID))
			pNeeds := (*components.Needs)(world.Get(bestPatient.entity, s.needsID))

			// Treat Patient
			pVitals.Blood += 30.0
			if pVitals.Blood > 100.0 {
				pVitals.Blood = 100.0
			}

			pVitals.Pain -= 30.0
			if pVitals.Pain < 0 {
				pVitals.Pain = 0
			}

			// Charge Fee
			fee := float32(50.0)
			if pNeeds.Wealth >= fee {
				pNeeds.Wealth -= fee
				dNeeds.Wealth += fee
			} else {
				// Patient is bankrupt, transfer what they have
				paid := pNeeds.Wealth
				dNeeds.Wealth += paid
				pNeeds.Wealth = 0

				// Generate negative hook (Medical Debt/Resentment)
				// Doctor holds resentment against the patient for not paying
				if s.hooks != nil {
					s.hooks.AddHook(dIdent.ID, bestPatient.id, -50)
				}
			}

			// Patient is treated, remove from list to prevent multiple doctors treating same entity this tick
			// Simple swap-and-pop for DOD efficiency
			for i := range injuredPatients {
				if injuredPatients[i].entity == bestPatient.entity {
					injuredPatients[i] = injuredPatients[len(injuredPatients)-1]
					injuredPatients = injuredPatients[:len(injuredPatients)-1]
					break
				}
			}

		} else {
			// Not adjacent, pathfind to patient
			if s.pathQueue != nil {
				s.pathQueue.Enqueue(engine.PathRequest{
					EntityID: dIdent.ID,
					StartX:   dPos.X,
					StartY:   dPos.Y,
					TargetX:  bestPatient.x,
					TargetY:  bestPatient.y,
					IsNaval:  false,
				})
			}
		}
	}
}
