package systems

import (
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 07.2: Information Leakage (GossipDistributionSystem)
// Propagates secrets between entities based on proximity, secret virality, and identity traits.

// GossipDistributionSystem handles the physical passing of secrets through the map.
type GossipDistributionSystem struct {
	tickCounter uint64

	// Component IDs
	posID    ecs.ID
	secretID ecs.ID
	memoryID ecs.ID
	identID  ecs.ID
	ruinID   ecs.ID
}

// nodeData represents extracted data for DOD optimized proximity checking
type nodeData struct {
	entity ecs.Entity
	pos    *components.Position
	secret *components.SecretComponent
	memory *components.Memory
	ident  *components.Identity
}

// Update runs the system every 10 ticks.
func (s *GossipDistributionSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Runs on a slower tick execution (every 10 Ticks)
	if s.tickCounter%10 != 0 {
		return
	}

	s.posID = ecs.ComponentID[components.Position](world)
	s.secretID = ecs.ComponentID[components.SecretComponent](world)
	s.memoryID = ecs.ComponentID[components.Memory](world)
	s.identID = ecs.ComponentID[components.Identity](world)
	s.ruinID = ecs.ComponentID[components.RuinComponent](world)

	// Filter all valid actors capable of gossiping
	filter := ecs.All(s.posID, s.secretID, s.memoryID, s.identID).Without(s.ruinID)
	query := world.Query(&filter)

	// Extract into a flat slice cache to prevent nested O(N^2) arche queries
	// and preserve L1/L2 hits during the proximity loop.
	var nodes []nodeData

	for query.Next() {
		nodes = append(nodes, nodeData{
			entity: query.Entity(),
			pos:    (*components.Position)(query.Get(s.posID)),
			secret: (*components.SecretComponent)(query.Get(s.secretID)),
			memory: (*components.Memory)(query.Get(s.memoryID)),
			ident:  (*components.Identity)(query.Get(s.identID)),
		})
	}

	// O(N^2) proximity check across the flat slice cache
	// In the future this should utilize spatial partitioning, but for now we iterate sequentially
	for i := 0; i < len(nodes); i++ {
		sender := nodes[i]

		if len(sender.secret.Secrets) == 0 {
			continue
		}

		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			receiver := nodes[j]

			// Distance check (Squared to avoid sqrt overhead)
			dx := sender.pos.X - receiver.pos.X
			dy := sender.pos.Y - receiver.pos.Y
			distSq := dx*dx + dy*dy

			// Overlap defined as distance < 2.0 (distSq < 4.0)
			if distSq < 4.0 {
				// Evaluate each secret the sender holds
				for _, secret := range sender.secret.Secrets {
					// Check if receiver already knows the secret
					alreadyKnown := false
					for _, known := range receiver.secret.Secrets {
						if known.SecretID == secret.SecretID {
							alreadyKnown = true
							break
						}
					}

					if alreadyKnown {
						continue
					}

					// Calculate chance
					chance := float32(secret.Virality) / 255.0

					// Apply TraitGossip modifier
					modifier := float32(1.0)
					if sender.ident.BaseTraits&components.TraitGossip != 0 {
						modifier = 2.0
					}

					// RNG Pass
					if engine.GetRandomFloat32() < chance*modifier {
						// Pass the secret

						// Inject SecretID into neighbor's MemoryComponent buffer
						head := receiver.memory.Head
						receiver.memory.Events[head] = components.MemoryEvent{
							TargetID:        uint64(sender.entity.ID()), // Storing ECS entity ID for reference
							TickStamp:       s.tickCounter,
							InteractionType: components.InteractionGossip,
							Value:           int32(secret.SecretID), // Safe because SecretID is uint32 and we use int32
						}

						// Increment ring buffer head
						receiver.memory.Head = (head + 1) % 50

						// Give the receiver the secret as well
						receiver.secret.Secrets = append(receiver.secret.Secrets, components.Secret{
							OriginID: secret.OriginID,
							SecretID: secret.SecretID,
							Virality: secret.Virality,
						})
					}
				}
			}
		}
	}
}
