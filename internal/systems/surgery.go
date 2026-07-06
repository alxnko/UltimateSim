package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 19.5: The Surgical Amputation Engine
// Bridges Biology (Anatomy), Geography (Pathfinding), Economy (Wealth), and Psychology (Sanity).

type SurgerySystem struct {
	doctorFilter  ecs.Filter
	patientFilter ecs.Filter

	jobID     ecs.ID
	posID     ecs.ID
	anatomyID ecs.ID
	needsID   ecs.ID
	sanityID  ecs.ID
	identID   ecs.ID

	pathQueue *engine.PathRequestQueue
	hooks     *engine.SparseHookGraph
}

func NewSurgerySystem(world *ecs.World, pathQueue *engine.PathRequestQueue, hooks *engine.SparseHookGraph) *SurgerySystem {
	jobID := ecs.ComponentID[components.JobComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	anatomyID := ecs.ComponentID[components.AnatomyComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	identID := ecs.ComponentID[components.Identity](world)

	dMask := ecs.All(jobID, posID, needsID, identID)
	pMask := ecs.All(posID, anatomyID, needsID, sanityID, identID)

	return &SurgerySystem{
		doctorFilter:  &dMask,
		patientFilter: &pMask,
		jobID:         jobID,
		posID:         posID,
		anatomyID:     anatomyID,
		needsID:       needsID,
		sanityID:      sanityID,
		identID:       identID,
		pathQueue:     pathQueue,
		hooks:         hooks,
	}
}

type surgeryPatientData struct {
	entity ecs.Entity
	x      float32
	y      float32
	id     uint64
}

func (s *SurgerySystem) Update(world *ecs.World) {
	var infectedPatients []surgeryPatientData

	pQuery := world.Query(s.patientFilter)
	for pQuery.Next() {
		anatomy := (*components.AnatomyComponent)(pQuery.Get(s.anatomyID))
		if anatomy.InfectedLimbs > 0 {
			pos := (*components.Position)(pQuery.Get(s.posID))
			ident := (*components.Identity)(pQuery.Get(s.identID))

			infectedPatients = append(infectedPatients, surgeryPatientData{
				entity: pQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				id:     ident.ID,
			})
		}
	}

	if len(infectedPatients) == 0 {
		return
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

		var bestPatient *surgeryPatientData
		var bestDistSq float32 = 9999999.0

		for i := range infectedPatients {
			p := &infectedPatients[i]

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

		if bestDistSq <= 2.0 {
			pAnatomy := (*components.AnatomyComponent)(world.Get(bestPatient.entity, s.anatomyID))
			pNeeds := (*components.Needs)(world.Get(bestPatient.entity, s.needsID))
			pSanity := (*components.SanityComponent)(world.Get(bestPatient.entity, s.sanityID))

			// Amputate limb
			pAnatomy.InfectedLimbs--
			pAnatomy.MissingLimbs++
			pAnatomy.InfectionProg = 0

			// Massive psychological trauma
			pSanity.Stress += 50.0
			if pSanity.Stress > pSanity.MaxStress {
				pSanity.Stress = pSanity.MaxStress
			}

			// Charge Fee
			fee := float32(100.0)
			if pNeeds.Wealth >= fee {
				pNeeds.Wealth -= fee
				dNeeds.Wealth += fee
			} else {
				paid := pNeeds.Wealth
				dNeeds.Wealth += paid
				pNeeds.Wealth = 0

				// Debt resentment hook
				if s.hooks != nil {
					s.hooks.AddHook(dIdent.ID, bestPatient.id, -50)
				}
			}

			// Remove treated patient from local cache to prevent multiple surgeries in one tick
			for i := range infectedPatients {
				if infectedPatients[i].entity == bestPatient.entity {
					infectedPatients[i] = infectedPatients[len(infectedPatients)-1]
					infectedPatients = infectedPatients[:len(infectedPatients)-1]
					break
				}
			}
		} else {
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
