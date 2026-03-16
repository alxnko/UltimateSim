package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 27.1: The Military Revolt Engine
// If a Guard learns a highly viral, negative secret about a Capital or Country (e.g. the BannedSecretID),
// they revolt by dropping their JobGuard status (switching to JobBandit) and gaining a massive negative hook
// against the Capital's ruler or country's owner in the SparseHookGraph, triggering the BloodFeudSystem.

type militaryRevoltNodeData struct {
	entity ecs.Entity
	id     uint64
	x      float32
	y      float32
	job    *components.JobComponent
	secret *components.SecretComponent
	affil  *components.Affiliation
}

type adminJurisdictionRevoltData struct {
	Entity         ecs.Entity
	ID             uint64
	X              float32
	Y              float32
	RadiusSquared  float32
	BannedSecretID uint32
	CityID         uint32
	Legitimacy     uint32 // Phase 35.1: Sovereign Legitimacy Engine
}

type MilitaryRevoltSystem struct {
	hooks         *engine.SparseHookGraph
	jurisdictions []adminJurisdictionRevoltData
	tickCounter   uint64

	// Component IDs
	posID     ecs.ID
	identID   ecs.ID
	jobID     ecs.ID
	secretID  ecs.ID
	affID     ecs.ID
	jurID     ecs.ID
	capID     ecs.ID
	legitID   ecs.ID
}

// NewMilitaryRevoltSystem creates a new MilitaryRevoltSystem.
func NewMilitaryRevoltSystem(world *ecs.World, hooks *engine.SparseHookGraph) *MilitaryRevoltSystem {
	return &MilitaryRevoltSystem{
		hooks:         hooks,
		jurisdictions: make([]adminJurisdictionRevoltData, 0, 20),
		tickCounter:   0,

		posID:     ecs.ComponentID[components.Position](world),
		identID:   ecs.ComponentID[components.Identity](world),
		jobID:     ecs.ComponentID[components.JobComponent](world),
		secretID:  ecs.ComponentID[components.SecretComponent](world),
		affID:     ecs.ComponentID[components.Affiliation](world),
		jurID:     ecs.ComponentID[components.JurisdictionComponent](world),
		capID:     ecs.ComponentID[components.CapitalComponent](world),
		legitID:   ecs.ComponentID[components.LegitimacyComponent](world),
	}
}

// Update runs the system every 10 ticks to reduce overhead.
func (s *MilitaryRevoltSystem) Update(world *ecs.World) {
	s.tickCounter++

	if s.tickCounter%10 != 0 {
		return
	}

	// 1. Pre-cache all Jurisdiction boundaries that have a BannedSecretID or LegitimacyComponent
	s.jurisdictions = s.jurisdictions[:0]
	jurQuery := world.Query(ecs.All(s.jurID, s.posID, s.identID))
	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(s.jurID))

		var legitimacy uint32 = 100 // Default high if no component exists
		hasLegitimacy := false
		if world.Has(jurQuery.Entity(), s.legitID) {
			legitComp := (*components.LegitimacyComponent)(world.Get(jurQuery.Entity(), s.legitID))
			legitimacy = legitComp.Score
			hasLegitimacy = true
		}

		if jur.BannedSecretID == 0 && !hasLegitimacy {
			continue // No active secret that threatens the state, and no legitimacy to fail
		}

		pos := (*components.Position)(jurQuery.Get(s.posID))
		ident := (*components.Identity)(jurQuery.Get(s.identID))

		var cityID uint32 = 0
		if world.Has(jurQuery.Entity(), s.affID) {
			aff := (*components.Affiliation)(world.Get(jurQuery.Entity(), s.affID))
			cityID = aff.CityID
		}

		s.jurisdictions = append(s.jurisdictions, adminJurisdictionRevoltData{
			Entity:         jurQuery.Entity(),
			ID:             ident.ID,
			X:              pos.X,
			Y:              pos.Y,
			RadiusSquared:  jur.RadiusSquared,
			BannedSecretID: jur.BannedSecretID,
			CityID:         cityID,
			Legitimacy:     legitimacy,
		})
	}

	if len(s.jurisdictions) == 0 {
		return
	}

	// 2. Extract Guards into a flat DOD slice
	guardQuery := world.Query(ecs.All(s.posID, s.identID, s.jobID, s.secretID, s.affID))
	var guards []militaryRevoltNodeData

	for guardQuery.Next() {
		job := (*components.JobComponent)(guardQuery.Get(s.jobID))
		if job.JobID != components.JobGuard {
			continue
		}

		pos := (*components.Position)(guardQuery.Get(s.posID))
		ident := (*components.Identity)(guardQuery.Get(s.identID))
		secret := (*components.SecretComponent)(guardQuery.Get(s.secretID))
		aff := (*components.Affiliation)(guardQuery.Get(s.affID))

		// We still process guards with no secrets now, because Legitimacy drop can trigger revolt even without secrets.
		guards = append(guards, militaryRevoltNodeData{
			entity: guardQuery.Entity(),
			id:     ident.ID,
			x:      pos.X,
			y:      pos.Y,
			job:    job,
			secret: secret,
			affil:  aff,
		})
	}

	// 3. Evaluate Guards against Banned Secrets
	for i := 0; i < len(guards); i++ {
		guard := guards[i]

		// Determine if the Guard is employed by a jurisdiction with a banned secret
		// We can check if they are inside the jurisdiction or if they belong to its city
		var targetJur *adminJurisdictionRevoltData
		for j := 0; j < len(s.jurisdictions); j++ {
			jur := &s.jurisdictions[j]
			if guard.affil.CityID == jur.CityID && jur.CityID != 0 {
				targetJur = jur
				break
			}

			// Fallback to spatial check if CityID doesn't match but they are inside
			dx := guard.x - jur.X
			dy := guard.y - jur.Y
			if (dx*dx)+(dy*dy) <= jur.RadiusSquared {
				targetJur = jur
				break
			}
		}

		if targetJur != nil {
			// Phase 35.1: Revolt due to Low Legitimacy (Score < 20)
			if targetJur.Legitimacy < 20 {
				guard.job.JobID = components.JobBandit
				guard.job.EmployerID = 0
				if s.hooks != nil {
					s.hooks.AddHook(guard.id, targetJur.ID, -100)
				}
				continue // Already revolted
			}

			// Phase 27.1: Revolt due to Banned Secret
			if targetJur.BannedSecretID != 0 {
				hasSecret := false
				for k := 0; k < len(guard.secret.Secrets); k++ {
					if guard.secret.Secrets[k].SecretID == targetJur.BannedSecretID {
						hasSecret = true
						break
					}
				}

				if hasSecret {
					guard.job.JobID = components.JobBandit
					guard.job.EmployerID = 0
					if s.hooks != nil {
						s.hooks.AddHook(guard.id, targetJur.ID, -100)
					}
				}
			}
		}
	}
}
