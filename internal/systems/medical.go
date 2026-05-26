package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 68 - The Physical Medical Engine
// Bridges Biology (Healing), Economy (Wealth Transfer), and Justice (Debt Resentment).

type MedicalSystem struct {
	patientFilter ecs.Filter
	doctorFilter  ecs.Filter

	identID  ecs.ID
	vitalsID ecs.ID
	posID    ecs.ID
	needsID  ecs.ID
	jobID    ecs.ID
	pathID   ecs.ID

	patients []patientData
	hooks    *engine.SparseHookGraph
}

type patientData struct {
	entity ecs.Entity
	x      float32
	y      float32
	ident  *components.Identity
	vitals *components.VitalsComponent
	needs  *components.Needs
}

func (s *MedicalSystem) IsExpensive() bool {
	return true
}

func NewMedicalSystem(world *ecs.World, hooks *engine.SparseHookGraph) *MedicalSystem {
	identID := ecs.ComponentID[components.Identity](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	pathID := ecs.ComponentID[components.Path](world)

	return &MedicalSystem{
		patientFilter: filter.All(identID, vitalsID, posID, needsID),
		doctorFilter:  filter.All(identID, posID, needsID, jobID),
		identID:       identID,
		vitalsID:      vitalsID,
		posID:         posID,
		needsID:       needsID,
		jobID:         jobID,
		pathID:        pathID,
		patients:      make([]patientData, 0, 100),
		hooks:         hooks,
	}
}

func (s *MedicalSystem) Update(world *ecs.World) {
	s.patients = s.patients[:0]

	// 1. Pre-cache all valid patients (Low blood or high pain)
	pQuery := world.Query(s.patientFilter)
	for pQuery.Next() {
		vitals := (*components.VitalsComponent)(pQuery.Get(s.vitalsID))
		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))
			needs := (*components.Needs)(pQuery.Get(s.needsID))

			s.patients = append(s.patients, patientData{
				entity: pQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				ident:  ident,
				vitals: vitals,
				needs:  needs,
			})
		}
	}

	if len(s.patients) == 0 {
		return
	}

	// 2. Iterate Doctors to pathfind and heal
	dQuery := world.Query(s.doctorFilter)
	for dQuery.Next() {
		job := (*components.JobComponent)(dQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		pos := (*components.Position)(dQuery.Get(s.posID))
		docIdent := (*components.Identity)(dQuery.Get(s.identID))
		docNeeds := (*components.Needs)(dQuery.Get(s.needsID))

		var docPath *components.Path
		if dQuery.Has(s.pathID) {
			docPath = (*components.Path)(dQuery.Get(s.pathID))
		}

		var bestPatient *patientData
		var bestDistSq float32 = 999999.0

		for i := 0; i < len(s.patients); i++ {
			p := &s.patients[i]
			// Don't treat fully healed or dead people
			if p.vitals.Blood >= 100.0 && p.vitals.Pain == 0.0 {
				continue
			}

			dx := pos.X - p.x
			dy := pos.Y - p.y
			distSq := dx*dx + dy*dy

			if distSq < bestDistSq {
				bestDistSq = distSq
				bestPatient = p
			}
		}

		if bestPatient == nil {
			continue
		}

		// If adjacent, perform healing
		if bestDistSq <= 2.0 {
			// Biology: Restore vitals
			bestPatient.vitals.Blood = 100.0
			bestPatient.vitals.Pain = 0.0

			// Economy/Justice: Deduct wealth or generate debt resentment
			treatmentCost := float32(10.0)
			if bestPatient.needs.Wealth >= treatmentCost {
				bestPatient.needs.Wealth -= treatmentCost
				docNeeds.Wealth += treatmentCost
			} else {
				// Patient is too poor. Doctor still heals but logs a negative hook via SparseHookGraph
				s.hooks.AddHook(docIdent.ID, bestPatient.ident.ID, -50)
			}
		} else if docPath != nil {
			// Pathfind to patient
			docPath.TargetX = bestPatient.x
			docPath.TargetY = bestPatient.y
			docPath.HasPath = false // Trigger repathing
		}
	}
}
