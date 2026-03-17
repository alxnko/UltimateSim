package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 39.2: The Epistemological Engine (LedgerDiscoverySystem)
// Allows NPCs to discover and read material Ledger entities on the map,
// injecting old, potentially banned, or forgotten secrets back into the
// active GossipDistributionSystem as highly contagious rumors.

type ledgerNodeData struct {
	entity  ecs.Entity
	x       float32
	y       float32
	secrets []uint32
}

type LedgerDiscoverySystem struct {
	tickCounter uint64

	// Component IDs
	npcID        ecs.ID
	posID        ecs.ID
	secretID     ecs.ID
	identID      ecs.ID
	ledgerTagID  ecs.ID
	ledgerCompID ecs.ID
}

// NewLedgerDiscoverySystem creates a new LedgerDiscoverySystem.
func NewLedgerDiscoverySystem(world *ecs.World) *LedgerDiscoverySystem {
	return &LedgerDiscoverySystem{
		tickCounter:  0,
		npcID:        ecs.ComponentID[components.NPC](world),
		posID:        ecs.ComponentID[components.Position](world),
		secretID:     ecs.ComponentID[components.SecretComponent](world),
		identID:      ecs.ComponentID[components.Identity](world),
		ledgerTagID:  ecs.ComponentID[components.Ledger](world),
		ledgerCompID: ecs.ComponentID[components.LedgerComponent](world),
	}
}

// Update allows NPCs to read ledgers if they stand on them.
func (s *LedgerDiscoverySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Only process periodically to reduce overhead
	if s.tickCounter%20 != 0 {
		return
	}

	// 1. Cache all physical ledgers on the map
	ledgerFilter := ecs.All(s.ledgerTagID, s.ledgerCompID, s.posID)
	ledgerQuery := world.Query(ledgerFilter)

	var ledgers []ledgerNodeData

	for ledgerQuery.Next() {
		pos := (*components.Position)(ledgerQuery.Get(s.posID))
		ledger := (*components.LedgerComponent)(ledgerQuery.Get(s.ledgerCompID))

		ledgers = append(ledgers, ledgerNodeData{
			entity:  ledgerQuery.Entity(),
			x:       pos.X,
			y:       pos.Y,
			secrets: ledger.Secrets,
		})
	}

	if len(ledgers) == 0 {
		return
	}

	// 2. Iterate NPCs and check for proximity to ledgers
	npcFilter := ecs.All(s.npcID, s.posID, s.secretID, s.identID)
	npcQuery := world.Query(npcFilter)

	for npcQuery.Next() {
		pos := (*components.Position)(npcQuery.Get(s.posID))
		secrets := (*components.SecretComponent)(npcQuery.Get(s.secretID))
		ident := (*components.Identity)(npcQuery.Get(s.identID))

		for i := 0; i < len(ledgers); i++ {
			l := ledgers[i]

			dx := pos.X - l.x
			dy := pos.Y - l.y
			distSq := dx*dx + dy*dy

			// Within ~1.4 tiles (sqrt(2))
			if distSq < 2.0 {
				// The NPC "reads" the ledger. Check each secret.
				for _, secretID := range l.secrets {
					alreadyKnown := false
					for _, known := range secrets.Secrets {
						if known.SecretID == secretID {
							alreadyKnown = true
							break
						}
					}

					// If the NPC doesn't know it, they learn it as a viral rumor
					if !alreadyKnown {
						secrets.Secrets = append(secrets.Secrets, components.Secret{
							OriginID: ident.ID,   // They act as the origin of their new knowledge
							SecretID: secretID,
							Virality: 255,        // Highly contagious when rediscovered
							BeliefID: 0,          // Can be expanded later for religious texts
						})
					}
				}
			}
		}
	}
}
