package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 68: The Physical Medical Engine
// Bridges Biology, Economy, and Justice by having JobDoctor NPCs
// physically pathfind to injured patients (Blood < 50 or Pain > 20),
// restoring health and deducting Wealth (or generating negative hooks) when adjacent.

type patientData struct {
	entity ecs.Entity
	id     uint64
	x      float32
	y      float32
	vitals *components.VitalsComponent
	needs  *components.Needs
}

type MedicalSystem struct {
	patientFilter ecs.Filter
	doctorFilter  ecs.Filter
	hookGraph     *engine.SparseHookGraph

	identID ecs.ID
	posID   ecs.ID
	vitalsID ecs.ID
	needsID ecs.ID
	jobID   ecs.ID
	pathID  ecs.ID
}

func NewMedicalSystem(world *ecs.World, hooks *engine.SparseHookGraph) *MedicalSystem {
	identID := ecs.ComponentID[components.Identity](world)
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	pathID := ecs.ComponentID[components.Path](world)
	npcID := ecs.ComponentID[components.NPC](world)

	pMask := ecs.All(npcID, identID, posID, vitalsID, needsID)
	dMask := ecs.All(npcID, identID, posID, jobID, pathID, needsID)

	return &MedicalSystem{
		patientFilter: &pMask,
		doctorFilter:  &dMask,
		hookGraph:     hooks,
		identID:       identID,
		posID:         posID,
		vitalsID:      vitalsID,
		needsID:       needsID,
		jobID:         jobID,
		pathID:        pathID,
	}
}

func (s *MedicalSystem) Update(world *ecs.World) {
	// 1. Cache all injured patients
	patients := make([]patientData, 0, 50)
	pQuery := world.Query(s.patientFilter)
	for pQuery.Next() {
		vitals := (*components.VitalsComponent)(pQuery.Get(s.vitalsID))
		if vitals.Blood < 50 || vitals.Pain > 20 {
			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))
			needs := (*components.Needs)(pQuery.Get(s.needsID))

			patients = append(patients, patientData{
				entity: pQuery.Entity(),
				id:     ident.ID,
				x:      pos.X,
				y:      pos.Y,
				vitals: vitals,
				needs:  needs,
			})
		}
	}

	if len(patients) == 0 {
		return
	}

	// 2. Iterate Doctors
	dQuery := world.Query(s.doctorFilter)
	for dQuery.Next() {
		job := (*components.JobComponent)(dQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		dPos := (*components.Position)(dQuery.Get(s.posID))
		dIdent := (*components.Identity)(dQuery.Get(s.identID))
		dNeeds := (*components.Needs)(dQuery.Get(s.needsID))
		dPath := (*components.Path)(dQuery.Get(s.pathID))

		// Find closest patient
		var bestPatient *patientData
		var bestDistSq float32 = 9999999.0

		for i := 0; i < len(patients); i++ {
			p := &patients[i]
			dx := dPos.X - p.x
			dy := dPos.Y - p.y
			distSq := dx*dx + dy*dy

			// Exclude self if doctor is injured
			if p.id == dIdent.ID {
				continue
			}

			if distSq < bestDistSq {
				bestPatient = p
				bestDistSq = distSq
			}
		}

		if bestPatient == nil {
			continue
		}

		// 3. Act on patient
		if bestDistSq <= 2.0 {
			// Heal
			bestPatient.vitals.Blood += 20.0
			if bestPatient.vitals.Blood > 100.0 {
				bestPatient.vitals.Blood = 100.0
			}
			bestPatient.vitals.Pain -= 20.0
			if bestPatient.vitals.Pain < 0.0 {
				bestPatient.vitals.Pain = 0.0
			}

			// Payment
			cost := float32(50.0)
			if bestPatient.needs.Wealth >= cost {
				bestPatient.needs.Wealth -= cost
				dNeeds.Wealth += cost
			} else {
				// Take all they have and log a grudge
				dNeeds.Wealth += bestPatient.needs.Wealth
				bestPatient.needs.Wealth = 0

				// Hook: Doctor hates patient for not paying
				if s.hookGraph != nil {
					s.hookGraph.AddHook(dIdent.ID, bestPatient.id, -50)
				}
			}

			// Remove from patient array to avoid multiple doctors healing the same person in one tick
			// (Optimization, though DOD state handles it too)
			for i := 0; i < len(patients); i++ {
				if patients[i].id == bestPatient.id {
					patients[i] = patients[len(patients)-1]
					patients = patients[:len(patients)-1]
					break
				}
			}
		} else {
			// Pathfind
			if dPath.TargetX != bestPatient.x || dPath.TargetY != bestPatient.y {
				dPath.TargetX = bestPatient.x
				dPath.TargetY = bestPatient.y
				dPath.HasPath = false // Repath
			}
		}
	}
}
