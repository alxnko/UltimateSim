package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 68 - The Physical Medical Engine
// MedicalSystem bridges Biology, Economy, and Justice by having JobDoctor NPCs physically pathfind to injured patients
// (Blood < 50 or Pain > 20), restoring health, and deducting Wealth (or generating negative hooks if poor).

type MedicalSystem struct {
	doctorFilter  ecs.Filter
	patientFilter ecs.Filter
	hooks         *engine.SparseHookGraph

	// Component IDs
	jobID    ecs.ID
	posID    ecs.ID
	pathID   ecs.ID
	identID  ecs.ID
	vitalsID ecs.ID
	needsID  ecs.ID
}

func NewMedicalSystem(world *ecs.World, hooks *engine.SparseHookGraph) *MedicalSystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)
	identID := ecs.ComponentID[components.Identity](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)

	dMask := filter.All(jobID, posID, pathID, identID, needsID)
	pMask := filter.All(vitalsID, posID, needsID, identID)

	return &MedicalSystem{
		doctorFilter:  dMask,
		patientFilter: pMask,
		hooks:         hooks,

		jobID:    jobID,
		posID:    posID,
		pathID:   pathID,
		identID:  identID,
		vitalsID: vitalsID,
		needsID:  needsID,
	}
}

type patientData struct {
	entity ecs.Entity
	x      float32
	y      float32
	id     uint64
	vitals *components.VitalsComponent
	needs  *components.Needs
}

func (s *MedicalSystem) Update(world *ecs.World) {
	// 1. Pre-cache all potential patients (Blood < 50 or Pain > 20)
	patients := make([]patientData, 0, 50)
	pQuery := world.Query(s.patientFilter)

	for pQuery.Next() {
		vitals := (*components.VitalsComponent)(pQuery.Get(s.vitalsID))
		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))
			needs := (*components.Needs)(pQuery.Get(s.needsID))

			patients = append(patients, patientData{
				entity: pQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				id:     ident.ID,
				vitals: vitals,
				needs:  needs,
			})
		}
	}

	if len(patients) == 0 {
		return // No patients to heal
	}

	// 2. Iterate over all JobDoctor NPCs
	dQuery := world.Query(s.doctorFilter)

	for dQuery.Next() {
		job := (*components.JobComponent)(dQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		dPos := (*components.Position)(dQuery.Get(s.posID))
		dPath := (*components.Path)(dQuery.Get(s.pathID))
		dIdent := (*components.Identity)(dQuery.Get(s.identID))
		dNeeds := (*components.Needs)(dQuery.Get(s.needsID))

		// Find nearest patient
		var bestPatient *patientData
		var bestDist float32 = 9999999.0
		var bestIdx int = -1

		for i := 0; i < len(patients); i++ {
			p := &patients[i]

			// Don't heal yourself in this logic block
			if p.id == dIdent.ID {
				continue
			}

			// Don't target already fully healed patients (handled by other doctors this tick)
			if p.vitals.Blood >= 100.0 && p.vitals.Pain <= 0.0 {
				continue
			}

			dx := dPos.X - p.x
			dy := dPos.Y - p.y
			distSq := dx*dx + dy*dy

			if distSq < bestDist {
				bestDist = distSq
				bestPatient = p
				bestIdx = i
			}
		}

		if bestPatient != nil {
			if bestDist <= 2.0 {
				// Heal the patient physically
				bestPatient.vitals.Blood = 100.0
				bestPatient.vitals.Pain = 0.0

				// Economy & Justice (Medical Bill)
				var fee float32 = 50.0

				if bestPatient.needs.Wealth >= fee {
					bestPatient.needs.Wealth -= fee
					dNeeds.Wealth += fee
				} else {
					// Patient cannot fully pay
					collected := bestPatient.needs.Wealth
					bestPatient.needs.Wealth = 0.0
					dNeeds.Wealth += collected

					// Bridge to Justice/Social Hierarchy Engine
					if s.hooks != nil {
						// Generate a negative hook (Grudge/Debt) against the patient
						s.hooks.AddHook(dIdent.ID, bestPatient.id, -20)
					}
				}

				// Remove patient from list to prevent other doctors from targeting them this tick
				if bestIdx != -1 {
					patients = append(patients[:bestIdx], patients[bestIdx+1:]...)
				}

				// Clear doctor's path since task is complete
				dPath.HasPath = false
				dPath.Nodes = dPath.Nodes[:0]

			} else {
				// Pathfind to patient
				if dPath.TargetX != bestPatient.x || dPath.TargetY != bestPatient.y {
					dPath.TargetX = bestPatient.x
					dPath.TargetY = bestPatient.y
					dPath.HasPath = false // Trigger movement system repathing
				}
			}
		}
	}
}
