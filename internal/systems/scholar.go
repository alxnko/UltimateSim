package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 39.1: The Epistemological Engine (ScholarSystem)
// Allows highly intellectual or wealthy NPCs to physically scribe their memories
// into material Ledger entities. This natively bridges Information (Phase 07) to
// Physical Items and Geography (Phase 01/02), allowing lost secrets to survive death.

type ScholarSystem struct {
	tickCounter uint64

	// Component IDs
	npcID     ecs.ID
	posID     ecs.ID
	genID     ecs.ID
	needsID   ecs.ID
	secretID  ecs.ID
}

// NewScholarSystem creates a new ScholarSystem.
func NewScholarSystem(world *ecs.World) *ScholarSystem {
	return &ScholarSystem{
		tickCounter: 0,
		npcID:       ecs.ComponentID[components.NPC](world),
		posID:       ecs.ComponentID[components.Position](world),
		genID:       ecs.ComponentID[components.GenomeComponent](world),
		needsID:     ecs.ComponentID[components.Needs](world),
		secretID:    ecs.ComponentID[components.SecretComponent](world),
	}
}

// Update evaluates intellectual NPCs for ledger creation.
func (s *ScholarSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Only process periodically to reduce overhead
	if s.tickCounter%100 != 0 {
		return
	}

	filter := ecs.All(s.npcID, s.posID, s.genID, s.needsID, s.secretID)
	query := world.Query(filter)

	type spawnData struct {
		x       float32
		y       float32
		secrets []uint32
	}

	var toSpawn []spawnData

	for query.Next() {
		gen := (*components.GenomeComponent)(query.Get(s.genID))
		needs := (*components.Needs)(query.Get(s.needsID))
		secrets := (*components.SecretComponent)(query.Get(s.secretID))

		// Criterion: Must be intelligent and wealthy enough to afford materials, and actually have secrets
		if gen.Intellect >= 150 && needs.Wealth >= 50.0 && len(secrets.Secrets) > 0 {
			pos := (*components.Position)(query.Get(s.posID))

			// Deduct material cost
			needs.Wealth -= 50.0

			// Extract secrets to write
			writtenSecrets := make([]uint32, 0, len(secrets.Secrets))
			for _, sec := range secrets.Secrets {
				writtenSecrets = append(writtenSecrets, sec.SecretID)
			}

			toSpawn = append(toSpawn, spawnData{
				x:       pos.X,
				y:       pos.Y,
				secrets: writtenSecrets,
			})
		}
	}

	// Structural modifications must occur outside the query loop
	if len(toSpawn) > 0 {
		ledgerTagID := ecs.ComponentID[components.Ledger](world)
		ledgerCompID := ecs.ComponentID[components.LedgerComponent](world)
		posID := ecs.ComponentID[components.Position](world)

		for _, data := range toSpawn {
			newEnt := world.NewEntity(ledgerTagID, posID, ledgerCompID)

			pos := (*components.Position)(world.Get(newEnt, posID))
			pos.X = data.x
			pos.Y = data.y

			ledger := (*components.LedgerComponent)(world.Get(newEnt, ledgerCompID))
			// Ensure secrets slice is independent (already uniquely allocated when extracted)
			ledger.Secrets = data.secrets
		}
	}
}
