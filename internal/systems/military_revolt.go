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
	}
}

// Update runs the system every 10 ticks to reduce overhead.
func (s *MilitaryRevoltSystem) Update(world *ecs.World) {
	s.tickCounter++

	if s.tickCounter%10 != 0 {
		return
	}

	// 1. Pre-cache all Jurisdiction boundaries that have a BannedSecretID
	s.jurisdictions = s.jurisdictions[:0]
	jurQuery := world.Query(ecs.All(s.jurID, s.posID, s.identID))
	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(s.jurID))
		if jur.BannedSecretID == 0 {
			continue // No active secret that threatens the state
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

		// Check if they hold any secrets at all
		if len(secret.Secrets) == 0 {
			continue
		}

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
			// Check if Guard holds the banned secret
			hasSecret := false
			for k := 0; k < len(guard.secret.Secrets); k++ {
				if guard.secret.Secrets[k].SecretID == targetJur.BannedSecretID {
					hasSecret = true
					break
				}
			}

			if hasSecret {
				// The Guard learns the truth and revolts!

				// 1. Strip JobGuard status, become a Bandit
				guard.job.JobID = components.JobBandit
				guard.job.EmployerID = 0 // Sever employment

				// 2. Generate a massive negative hook against the ruler/capital entity
				if s.hooks != nil {
					// -100 hook triggers BloodFeudSystem (which checks for <= -50)
					s.hooks.AddHook(guard.id, targetJur.ID, -100)
				}
			}
		}
	}
}
