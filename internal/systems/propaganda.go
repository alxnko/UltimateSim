package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 04.5: The Epistemological Layer (Propaganda & Erasure)
// Scours Jurisdiction boundaries. If a Jurisdiction enforces a BannedSecretID,
// it actively deletes this secret from young NPCs (< 30), executes elders (>= 60),
// and burns LedgerEntities holding the truth.

type adminPropagandaData struct {
	Entity         ecs.Entity
	X              float32
	Y              float32
	RadiusSquared  float32
	BannedSecretID uint32
}

type PropagandaSystem struct {
	jurisdictions []adminPropagandaData
	tickCounter   uint64
	toRemove      []ecs.Entity
}

func NewPropagandaSystem(world *ecs.World) *PropagandaSystem {
	return &PropagandaSystem{
		jurisdictions: make([]adminPropagandaData, 0, 20),
		tickCounter:   0,
		toRemove:      make([]ecs.Entity, 0, 100),
	}
}

func (s *PropagandaSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run every 20 ticks to reduce massive O(N^2) overhead
	if s.tickCounter%20 != 0 {
		return
	}

	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	posID := ecs.ComponentID[components.Position](world)

	// 1. Cache active Propaganda states
	s.jurisdictions = s.jurisdictions[:0]
	jurQuery := world.Query(ecs.All(jurID, posID))
	for jurQuery.Next() {
		jur := (*components.JurisdictionComponent)(jurQuery.Get(jurID))
		if jur.BannedSecretID == 0 {
			continue // No active erasure
		}

		pos := (*components.Position)(jurQuery.Get(posID))
		s.jurisdictions = append(s.jurisdictions, adminPropagandaData{
			Entity:         jurQuery.Entity(),
			X:              pos.X,
			Y:              pos.Y,
			RadiusSquared:  jur.RadiusSquared,
			BannedSecretID: jur.BannedSecretID,
		})
	}

	if len(s.jurisdictions) == 0 {
		return
	}

	// 2. Erase from youth and kill elders (NPC memory check)
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	secretID := ecs.ComponentID[components.SecretComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)

	npcQuery := world.Query(ecs.All(npcID, identID, posID, secretID, needsID))

	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(posID))
		secrets := (*components.SecretComponent)(npcQuery.Get(secretID))
		ident := (*components.Identity)(npcQuery.Get(identID))

		// Check if they hold any secrets at all
		if len(secrets.Secrets) == 0 {
			continue
		}

		// Find if NPC is in a propaganda zone
		var activeBannedID uint32 = 0
		for i := 0; i < len(s.jurisdictions); i++ {
			j := &s.jurisdictions[i]
			dx := pos.X - j.X
			dy := pos.Y - j.Y
			if (dx*dx)+(dy*dy) <= j.RadiusSquared {
				activeBannedID = j.BannedSecretID
				break // Assume first overlapping jurisdiction wins for speed
			}
		}

		if activeBannedID > 0 {
			// Check if NPC holds the banned secret
			hasSecret := false
			secretIndex := -1

			for i := 0; i < len(secrets.Secrets); i++ {
				if secrets.Secrets[i].SecretID == activeBannedID {
					hasSecret = true
					secretIndex = i
					break
				}
			}

			if hasSecret {
				if ident.Age < 30 {
					// State-sponsored forgetting (Erasure)
					// Remove the secret efficiently (swap with last element and shrink)
					lastIdx := len(secrets.Secrets) - 1
					secrets.Secrets[secretIndex] = secrets.Secrets[lastIdx]
					secrets.Secrets = secrets.Secrets[:lastIdx]
				} else if ident.Age >= 60 {
					// Killing elders who remember the truth
					needs := (*components.Needs)(npcQuery.Get(needsID))
					needs.Food = 0 // Let DeathSystem handle the actual removal
				}
			}
		}
	}

	// 3. Burn Ledgers (Physical Items)
	ledgerTagID := ecs.ComponentID[components.Ledger](world)
	ledgerCompID := ecs.ComponentID[components.LedgerComponent](world)

	ledgerQuery := world.Query(ecs.All(ledgerTagID, ledgerCompID, posID))
	s.toRemove = s.toRemove[:0]

	for ledgerQuery.Next() {
		pos := (*components.Position)(ledgerQuery.Get(posID))
		ledger := (*components.LedgerComponent)(ledgerQuery.Get(ledgerCompID))

		var activeBannedID uint32 = 0
		for i := 0; i < len(s.jurisdictions); i++ {
			j := &s.jurisdictions[i]
			dx := pos.X - j.X
			dy := pos.Y - j.Y
			if (dx*dx)+(dy*dy) <= j.RadiusSquared {
				activeBannedID = j.BannedSecretID
				break
			}
		}

		if activeBannedID > 0 {
			// Check if ledger contains banned secret
			burn := false
			for _, sid := range ledger.Secrets {
				if sid == activeBannedID {
					burn = true
					break
				}
			}

			if burn {
				s.toRemove = append(s.toRemove, ledgerQuery.Entity())
			}
		}
	}

	// Clean up destroyed ledgers
	for _, entity := range s.toRemove {
		if world.Alive(entity) {
			world.RemoveEntity(entity)
		}
	}
}
