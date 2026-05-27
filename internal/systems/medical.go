package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 68 - The Physical Medical Engine
// MedicalSystem bridges Biology, Economy, and Justice.
// Doctors physically pathfind to injured patients, restore health,
// and extract wealth. If the patient is poor, it generates a negative hook.

type medicalPatientData struct {
	Entity ecs.Entity
	X      float32
	Y      float32
	Blood  float32
	Pain   float32
}

type MedicalSystem struct {
	world     *ecs.World
	pathQueue *engine.PathRequestQueue
	hooks     *engine.SparseHookGraph

	doctorFilter  ecs.Filter
	patientFilter ecs.Filter

	patientCache []medicalPatientData
}

func NewMedicalSystem(world *ecs.World, pathQueue *engine.PathRequestQueue, hooks *engine.SparseHookGraph) *MedicalSystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	pathID := ecs.ComponentID[components.Path](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	idID := ecs.ComponentID[components.Identity](world)

	dMask := filter.All(jobID, posID, pathID)
	pMask := filter.All(vitalsID, posID, treasuryID, idID)

	return &MedicalSystem{
		world:         world,
		pathQueue:     pathQueue,
		hooks:         hooks,
		doctorFilter:  dMask,
		patientFilter: pMask,
		patientCache:  make([]medicalPatientData, 0, 100),
	}
}

func (s *MedicalSystem) IsExpensive() bool {
	return true
}

func (s *MedicalSystem) Update(world *ecs.World) {
	// Pre-cache IDs
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	pathID := ecs.ComponentID[components.Path](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)
	idID := ecs.ComponentID[components.Identity](world)

	// Step 1: Pre-cache all valid patients (Blood < 50 || Pain > 20)
	s.patientCache = s.patientCache[:0]
	patientQuery := world.Query(s.patientFilter)
	for patientQuery.Next() {
		vitals := (*components.VitalsComponent)(patientQuery.Get(vitalsID))
		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(patientQuery.Get(posID))
			s.patientCache = append(s.patientCache, medicalPatientData{
				Entity: patientQuery.Entity(),
				X:      pos.X,
				Y:      pos.Y,
				Blood:  vitals.Blood,
				Pain:   vitals.Pain,
			})
		}
	}

	if len(s.patientCache) == 0 {
		return
	}

	// Step 2: Evaluate Doctors
	doctorQuery := world.Query(s.doctorFilter)
	for doctorQuery.Next() {
		job := (*components.JobComponent)(doctorQuery.Get(jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		docPos := (*components.Position)(doctorQuery.Get(posID))
		docPath := (*components.Path)(doctorQuery.Get(pathID))

		// Find the closest patient
		var bestPatient *medicalPatientData
		minDistSq := float32(999999.0)

		for i := range s.patientCache {
			p := &s.patientCache[i]

			// Skip if another doctor already treated this patient this tick
			if p.Blood == 100.0 && p.Pain == 0.0 {
				continue
			}

			dx := p.X - docPos.X
			dy := p.Y - docPos.Y
			distSq := dx*dx + dy*dy

			if distSq < minDistSq {
				minDistSq = distSq
				bestPatient = p
			}
		}

		if bestPatient == nil {
			continue
		}

		// If adjacent, perform treatment
		if minDistSq < 4.0 {
			// Stop moving
			docPath.HasPath = false
			docPath.Nodes = nil

			// Perform Treatment - Get Pointers via world.Get() to avoid invalidation
			pVitals := (*components.VitalsComponent)(world.Get(bestPatient.Entity, vitalsID))
			pTreasury := (*components.TreasuryComponent)(world.Get(bestPatient.Entity, treasuryID))
			pId := (*components.Identity)(world.Get(bestPatient.Entity, idID))

			// We also need the doctor's identity for the hook
			var docId *components.Identity
			if world.Has(doctorQuery.Entity(), idID) {
				docId = (*components.Identity)(world.Get(doctorQuery.Entity(), idID))
			}

			// Apply Treatment
			pVitals.Blood = 100.0
			pVitals.Pain = 0.0

			// Economics & Justice
			treatmentCost := float32(20.0)
			if pTreasury.Wealth >= treatmentCost {
				pTreasury.Wealth -= treatmentCost
				// Note: Ideally the doctor would receive this wealth, but for simplicity
				// we just deduct it or we could give it to the doctor's employer
			} else {
				// Debt Default -> Negative Hook
				pTreasury.Wealth = 0.0
				if docId != nil && s.hooks != nil {
					// Doctor holds a grudge against the patient
					s.hooks.AddHook(docId.ID, pId.ID, -50)
				}
			}

			// Remove from cache to prevent multiple doctors targeting the same patient this tick
			bestPatient.Blood = 100.0 // invalidate in cache
		} else {
			// Enqueue path request to patient
			s.pathQueue.Enqueue(engine.PathRequest{
				EntityID: uint64(doctorQuery.Entity().ID()),
				StartX:   docPos.X,
				StartY:   docPos.Y,
				TargetX:  bestPatient.X,
				TargetY:  bestPatient.Y,
			})
		}
	}
}
