package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// SurgerySystem implements the Phase 19.5 Surgical Amputation Engine.
type SurgerySystem struct {
	doctorFilter  ecs.Filter
	patientFilter ecs.Filter

	jobID     ecs.ID
	posID     ecs.ID
	anatomyID ecs.ID
	vitalsID  ecs.ID
	needsID   ecs.ID
	identID   ecs.ID
	sanityID  ecs.ID

	pathQueue        *engine.PathRequestQueue
	hooks            *engine.SparseHookGraph
	infectedPatients []infectedPatientData
}

// NewSurgerySystem creates a new SurgerySystem instance.
func NewSurgerySystem(world *ecs.World, pathQueue *engine.PathRequestQueue, hooks *engine.SparseHookGraph) *SurgerySystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	anatomyID := ecs.ComponentID[components.AnatomyComponent](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	identID := ecs.ComponentID[components.Identity](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)

	dMask := ecs.All(jobID, posID, needsID, identID)
	pMask := ecs.All(posID, anatomyID, vitalsID, needsID, identID, sanityID)

	return &SurgerySystem{
		doctorFilter:     &dMask,
		patientFilter:    &pMask,
		jobID:            jobID,
		posID:            posID,
		anatomyID:        anatomyID,
		vitalsID:         vitalsID,
		needsID:          needsID,
		identID:          identID,
		sanityID:         sanityID,
		pathQueue:        pathQueue,
		hooks:            hooks,
		infectedPatients: make([]infectedPatientData, 0, 100),
	}
}

type infectedPatientData struct {
	entity ecs.Entity
	x      float32
	y      float32
	id     uint64
}

func (s *SurgerySystem) Update(world *ecs.World) {
	const (
		progressionRate          = 0.5
		infectionLethalThreshold = 100.0
		bloodDrainRate           = 2.0
		amputationFee            = 50.0
		amputationStress         = 50.0
		resentmentHookAmount     = -50
	)

	s.infectedPatients = s.infectedPatients[:0]

	// 1. Progress infection for all patients
	pQuery := world.Query(s.patientFilter)
	for pQuery.Next() {
		anatomy := (*components.AnatomyComponent)(pQuery.Get(s.anatomyID))
		vitals := (*components.VitalsComponent)(pQuery.Get(s.vitalsID))

		if anatomy.InfectedLimbs > 0 {
			anatomy.InfectionProg += progressionRate

			if anatomy.InfectionProg >= infectionLethalThreshold {
				vitals.Blood -= bloodDrainRate
				if vitals.Blood < 0 {
					vitals.Blood = 0
				}
			}

			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))

			s.infectedPatients = append(s.infectedPatients, infectedPatientData{
				entity: pQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				id:     ident.ID,
			})
		}
	}

	if len(s.infectedPatients) == 0 {
		return // No one needs surgery
	}

	// 2. Doctors pathfind and amputate
	dQuery := world.Query(s.doctorFilter)
	for dQuery.Next() {
		job := (*components.JobComponent)(dQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		dPos := (*components.Position)(dQuery.Get(s.posID))
		dIdent := (*components.Identity)(dQuery.Get(s.identID))
		dNeeds := (*components.Needs)(dQuery.Get(s.needsID))

		// Find closest infected patient
		var bestPatient *infectedPatientData
		var bestDistSq float32 = 9999999.0

		for i := range s.infectedPatients {
			p := &s.infectedPatients[i]

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

		// Amputate if adjacent
		if bestDistSq <= 2.0 {
			pAnatomy := (*components.AnatomyComponent)(world.Get(bestPatient.entity, s.anatomyID))
			pNeeds := (*components.Needs)(world.Get(bestPatient.entity, s.needsID))
			pSanity := (*components.SanityComponent)(world.Get(bestPatient.entity, s.sanityID))

			// We need an infected limb to amputate. Let's find the first bit set in InfectedLimbs.
			if pAnatomy.InfectedLimbs > 0 {
				var limbBit uint8 = 1
				for i := 0; i < 8; i++ {
					if (pAnatomy.InfectedLimbs & (limbBit << i)) != 0 {
						limbBit <<= i
						break
					}
				}

				// Amputate
				pAnatomy.InfectedLimbs &^= limbBit
				pAnatomy.MissingLimbs |= limbBit

				// Reset progression only if no other infected limbs
				if pAnatomy.InfectedLimbs == 0 {
					pAnatomy.InfectionProg = 0.0
				}

				// Psychological shock
				pSanity.Stress += amputationStress
				if pSanity.Stress > pSanity.MaxStress {
					pSanity.Stress = pSanity.MaxStress
				}

				// Economy transfer
				if pNeeds.Wealth >= amputationFee {
					pNeeds.Wealth -= amputationFee
					dNeeds.Wealth += amputationFee
				} else {
					paid := pNeeds.Wealth
					dNeeds.Wealth += paid
					pNeeds.Wealth = 0

					if s.hooks != nil {
						s.hooks.AddHook(dIdent.ID, bestPatient.id, resentmentHookAmount)
					}
				}
			}

			// Patient treated, remove from list to prevent duplicate operations this tick
			for i := range s.infectedPatients {
				if s.infectedPatients[i].entity == bestPatient.entity {
					s.infectedPatients[i] = s.infectedPatients[len(s.infectedPatients)-1]
					s.infectedPatients = s.infectedPatients[:len(s.infectedPatients)-1]
					break
				}
			}

		} else {
			// Not adjacent, pathfind
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
