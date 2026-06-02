package systems

import (
	"math"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 68 - The Physical Medical Engine
// MedicalSystem requires JobDoctor NPCs to physically pathfind to injured patients
// (Blood < 50 or Pain > 20). When physically adjacent, the doctor restores health
// and deducts Wealth from the patient (or generates negative hooks if poor).

type patientData struct {
	Entity ecs.Entity
	ID     uint64
	X      float32
	Y      float32
	Vitals *components.VitalsComponent
}

type MedicalSystem struct {
	world     *ecs.World
	pathQueue *engine.PathRequestQueue
	hookGraph *engine.SparseHookGraph

	doctorFilter  ecs.Filter
	patientFilter ecs.Filter

	patients []patientData

	npcID     ecs.ID
	jobID     ecs.ID
	posID     ecs.ID
	pathID    ecs.ID
	vitalsID  ecs.ID
	identID   ecs.ID
	treasID   ecs.ID
}

func NewMedicalSystem(world *ecs.World, pathQueue *engine.PathRequestQueue, hookGraph *engine.SparseHookGraph) *MedicalSystem {
	npcID := ecs.ComponentID[components.NPC](world)
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	pathID := ecs.ComponentID[components.Path](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	treasID := ecs.ComponentID[components.TreasuryComponent](world)

	// Doctors must have NPC, Job, Position, Path, Identity
	docMask := filter.All(npcID, jobID, posID, pathID, identID)

	// Patients must have Vitals, Position, Identity
	patMask := filter.All(vitalsID, posID, identID)

	return &MedicalSystem{
		world:         world,
		pathQueue:     pathQueue,
		hookGraph:     hookGraph,
		doctorFilter:  docMask,
		patientFilter: patMask,
		patients:      make([]patientData, 0, 100),

		npcID:     npcID,
		jobID:     jobID,
		posID:     posID,
		pathID:    pathID,
		vitalsID:  vitalsID,
		identID:   identID,
		treasID:   treasID,
	}
}

func (s *MedicalSystem) Update(world *ecs.World) {
	s.patients = s.patients[:0] // Clear slice, keep capacity

	// 1. Find all patients needing medical attention
	pQuery := s.world.Query(s.patientFilter)
	for pQuery.Next() {
		vitals := (*components.VitalsComponent)(pQuery.Get(s.vitalsID))
		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))
			s.patients = append(s.patients, patientData{
				Entity: pQuery.Entity(),
				ID:     ident.ID,
				X:      pos.X,
				Y:      pos.Y,
				Vitals: vitals,
			})
		}
	}

	if len(s.patients) == 0 {
		return // No patients, early exit
	}

	// 2. Process doctors
	dQuery := s.world.Query(s.doctorFilter)
	for dQuery.Next() {
		job := (*components.JobComponent)(dQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		pos := (*components.Position)(dQuery.Get(s.posID))
		ident := (*components.Identity)(dQuery.Get(s.identID))
		path := (*components.Path)(dQuery.Get(s.pathID))

		// Find the nearest patient
		var bestPatient *patientData
		var bestDist float32 = math.MaxFloat32

		for i := 0; i < len(s.patients); i++ {
			p := &s.patients[i]
			// Don't treat self
			if p.ID == ident.ID {
				continue
			}

			// Don't treat dead patients or fully healed ones
			if p.Vitals.Blood <= 0.0 || (p.Vitals.Blood >= 100.0 && p.Vitals.Pain <= 0.0) {
				continue
			}

			dx := p.X - pos.X
			dy := p.Y - pos.Y
			distSq := dx*dx + dy*dy
			if distSq < bestDist {
				bestDist = distSq
				bestPatient = p
			}
		}

		if bestPatient != nil {
			if bestDist > 1.0 {
				// Move towards patient
				if !path.HasPath || path.TargetX != bestPatient.X || path.TargetY != bestPatient.Y {
					req := engine.PathRequest{
						EntityID: ident.ID,
						StartX:   pos.X,
						StartY:   pos.Y,
						TargetX:  bestPatient.X,
						TargetY:  bestPatient.Y,
					}
					s.pathQueue.Enqueue(req)
					path.HasPath = true
					path.TargetX = bestPatient.X
					path.TargetY = bestPatient.Y
				}
			} else {
				// We are adjacent! Perform physical healing

				// Heal
				if bestPatient.Vitals.Blood < 100.0 {
					bestPatient.Vitals.Blood += 25.0
					if bestPatient.Vitals.Blood > 100.0 {
						bestPatient.Vitals.Blood = 100.0
					}
				}
				if bestPatient.Vitals.Pain > 0.0 {
					bestPatient.Vitals.Pain -= 25.0
					if bestPatient.Vitals.Pain < 0.0 {
						bestPatient.Vitals.Pain = 0.0
					}
				}

				// Economics & Justice
				cost := float32(10.0)
				var patientTreasury *components.TreasuryComponent
				if s.world.Has(bestPatient.Entity, s.treasID) {
					patientTreasury = (*components.TreasuryComponent)(s.world.Get(bestPatient.Entity, s.treasID))
				}

				var doctorTreasury *components.TreasuryComponent
				if s.world.Has(dQuery.Entity(), s.treasID) {
					doctorTreasury = (*components.TreasuryComponent)(s.world.Get(dQuery.Entity(), s.treasID))
				}

				if patientTreasury != nil && patientTreasury.Wealth >= cost {
					patientTreasury.Wealth -= cost
					if doctorTreasury != nil {
						doctorTreasury.Wealth += cost
					}
				} else {
					// Patient cannot afford! Generate a massive negative hook (medical debt/malpractice grudge depending on perspective, or just a deep societal debt)
					// We'll give the Doctor a positive hook on the patient for saving their life,
					// but also maybe a negative hook if they're mad they didn't get paid?
					// Let's go with: Doctor holds a significant Hook over the patient.
					if s.hookGraph != nil {
						s.hookGraph.AddHook(ident.ID, bestPatient.ID, 50)
					}
				}

				// We handled this patient, but they might still need healing if Blood < 100.
				// We'll let the next tick handle it.
			}
		}
	}
}
