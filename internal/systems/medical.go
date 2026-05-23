package systems

import (
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
	"github.com/mlange-42/arche/generic"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 68: The Physical Medical Engine
// MedicalSystem connects Biology, Economy, and Justice by having JobDoctor NPCs
// physically pathfind to injured patients. If the patient is too poor to pay for
// the healing, the doctor logs a negative hook via SparseHookGraph.

// cachedPatient represents a pre-cached target for doctors
type cachedPatient struct {
	Entity ecs.Entity
	X      float32
	Y      float32
	Blood  float32
	Pain   float32
}

type MedicalSystem struct {
	world            *ecs.World
	pathQueue        *engine.PathRequestQueue
	hookGraph        *engine.SparseHookGraph

	// Pre-cached mappers to avoid allocation
	npcFilter        ecs.Filter
	patientFilter    ecs.Filter

	posID      ecs.ID
	jobID      ecs.ID
	pathID     ecs.ID
	vitalsID   ecs.ID
	treasuryID ecs.ID

	vitalsMapper     generic.Map1[components.VitalsComponent]
	treasuryMapper   generic.Map1[components.TreasuryComponent]

	cachedPatients   []cachedPatient
}

func NewMedicalSystem(world *ecs.World, queue *engine.PathRequestQueue, graph *engine.SparseHookGraph) *MedicalSystem {
	return &MedicalSystem{
		world:     world,
		pathQueue: queue,
		hookGraph: graph,

		npcFilter:     filter.All(ecs.ComponentID[components.NPC](world), ecs.ComponentID[components.Position](world), ecs.ComponentID[components.JobComponent](world), ecs.ComponentID[components.Path](world), ecs.ComponentID[components.Identity](world)),
		patientFilter: filter.All(ecs.ComponentID[components.NPC](world), ecs.ComponentID[components.Position](world), ecs.ComponentID[components.VitalsComponent](world), ecs.ComponentID[components.TreasuryComponent](world), ecs.ComponentID[components.Identity](world)),

		posID:      ecs.ComponentID[components.Position](world),
		jobID:      ecs.ComponentID[components.JobComponent](world),
		pathID:     ecs.ComponentID[components.Path](world),
		vitalsID:   ecs.ComponentID[components.VitalsComponent](world),
		treasuryID: ecs.ComponentID[components.TreasuryComponent](world),

		vitalsMapper:     generic.NewMap1[components.VitalsComponent](world),
		treasuryMapper:   generic.NewMap1[components.TreasuryComponent](world),

		cachedPatients: make([]cachedPatient, 0, 100),
	}
}

func (s *MedicalSystem) Update() {
	s.cachedPatients = s.cachedPatients[:0]

	// Cache patients
	patientQuery := s.world.Query(s.patientFilter)
	for patientQuery.Next() {
		vitals := (*components.VitalsComponent)(patientQuery.Get(s.vitalsID))

		if vitals.Blood < 50.0 || vitals.Pain > 20.0 {
			pos := (*components.Position)(patientQuery.Get(s.posID))
			s.cachedPatients = append(s.cachedPatients, cachedPatient{
				Entity: patientQuery.Entity(),
				X:      pos.X,
				Y:      pos.Y,
				Blood:  vitals.Blood,
				Pain:   vitals.Pain,
			})
		}
	}

	// Early exit
	if len(s.cachedPatients) == 0 {
		return
	}

	// Iterate doctors
	doctorQuery := s.world.Query(s.npcFilter)
	for doctorQuery.Next() {
		job := (*components.JobComponent)(doctorQuery.Get(s.jobID))
		if job.JobID != components.JobDoctor {
			continue
		}

		docPos := (*components.Position)(doctorQuery.Get(s.posID))
		docPath := (*components.Path)(doctorQuery.Get(s.pathID))

		// Find closest patient
		var bestDist float32 = 999999.0
		var bestPatient cachedPatient
		found := false

		for _, p := range s.cachedPatients {
			dx := docPos.X - p.X
			dy := docPos.Y - p.Y
			distSq := dx*dx + dy*dy
			if distSq < bestDist {
				bestDist = distSq
				bestPatient = p
				found = true
			}
		}

		if !found {
			continue
		}

		docIdent := (*components.Identity)(doctorQuery.Get(ecs.ComponentID[components.Identity](s.world)))
		docEntityID := uint64(docIdent.ID)

		// Heal if adjacent
		if bestDist <= 4.0 {
			// Heal
			vitals := s.vitalsMapper.Get(bestPatient.Entity)

			// Guard: Only charge fee if actual healing is needed/performed
			// (Prevents multiple doctors charging the same patient in one tick)
			needsHealing := vitals.Blood < 50.0 || vitals.Pain > 20.0

			if needsHealing {
				if vitals.Blood < 50.0 {
					vitals.Blood = 100.0
				}
				if vitals.Pain > 20.0 {
					vitals.Pain = 0.0
				}

				// Charge Fee
				treasury := s.treasuryMapper.Get(bestPatient.Entity)
				fee := float32(50.0)

				if treasury.Wealth >= fee {
					treasury.Wealth -= fee
				} else {
					// Log debt hook
					// For the medical system, the hook value is proportional to the unpaid debt
					unpaid := fee - treasury.Wealth
					treasury.Wealth = 0.0 // Drain whatever they had

					// Calculate negative hook (-1 hook per unpaid wealth point)
					hookValue := -int32(unpaid)

					patIdent := (*components.Identity)(s.world.Get(bestPatient.Entity, ecs.ComponentID[components.Identity](s.world)))
					patEntityID := uint64(patIdent.ID)

					// Doctor gains a hook over patient (we represent this as patient owing doctor)
					// SparseHookGraph mapping: Source -> Target = HookValue
					s.hookGraph.AddHook(docEntityID, patEntityID, int(hookValue))
				}
			}
		} else {
			// Move to patient
			s.pathQueue.Enqueue(engine.PathRequest{
				EntityID: docEntityID,
				StartX:   docPos.X,
				StartY:   docPos.Y,
				TargetX:  bestPatient.X,
				TargetY:  bestPatient.Y,
				IsNaval:  false,
			})
			docPath.HasPath = true
		}
	}
}
