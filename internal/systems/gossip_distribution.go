package systems

import (
	"github.com/mlange-42/arche/ecs"
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
)

// Phase 07.2: Information Leakage (GossipDistributionSystem)
// Propagates secrets between entities based on proximity, secret virality, and identity traits.

type GossipDistributionSystem struct {
	tickCounter uint64
	HookGraph   *engine.SparseHookGraph

	// Component IDs
	posID     ecs.ID
	secretID  ecs.ID
	memoryID  ecs.ID
	identID   ecs.ID
	ruinID    ecs.ID
	cultureID ecs.ID
	beliefID  ecs.ID // Phase 07.5: Ideological Infection
}

// NewGossipDistributionSystem creates a new GossipDistributionSystem.
func NewGossipDistributionSystem(world *ecs.World, hookGraph *engine.SparseHookGraph) *GossipDistributionSystem {
	return &GossipDistributionSystem{
		HookGraph: hookGraph,
		posID:     ecs.ComponentID[components.Position](world),
		secretID:  ecs.ComponentID[components.SecretComponent](world),
		memoryID:  ecs.ComponentID[components.Memory](world),
		identID:   ecs.ComponentID[components.Identity](world),
		ruinID:    ecs.ComponentID[components.RuinComponent](world),
		cultureID: ecs.ComponentID[components.CultureComponent](world),
		beliefID:  ecs.ComponentID[components.BeliefComponent](world),
	}
}

// Update runs the system every 10 ticks.
func (s *GossipDistributionSystem) Update(world *ecs.World) {
	s.tickCounter++

	// nodeData represents extracted data for DOD optimized proximity checking
	type nodeData struct {
		entity  ecs.Entity
		pos     *components.Position
		secret  *components.SecretComponent
		memory  *components.Memory
		ident   *components.Identity
		culture *components.CultureComponent
		belief  *components.BeliefComponent // Optional, might be nil
	}

	// Runs on a slower tick execution (every 10 Ticks)
	if s.tickCounter%10 != 0 {
		return
	}

	// Filter all valid actors capable of gossiping
	filter := ecs.All(s.posID, s.secretID, s.memoryID, s.identID, s.cultureID).Without(s.ruinID)
	query := world.Query(&filter)

	// Extract into a flat slice cache to prevent nested O(N^2) arche queries
	// and preserve L1/L2 hits during the proximity loop.
	var nodes []nodeData

	for query.Next() {
		var belief *components.BeliefComponent
		if query.Has(s.beliefID) {
			belief = (*components.BeliefComponent)(query.Get(s.beliefID))
		}

		nodes = append(nodes, nodeData{
			entity:  query.Entity(),
			pos:     (*components.Position)(query.Get(s.posID)),
			secret:  (*components.SecretComponent)(query.Get(s.secretID)),
			memory:  (*components.Memory)(query.Get(s.memoryID)),
			ident:   (*components.Identity)(query.Get(s.identID)),
			culture: (*components.CultureComponent)(query.Get(s.cultureID)),
			belief:  belief,
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
				languageMismatch := sender.culture.LanguageID != receiver.culture.LanguageID

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

					// Apply Phase 07.4 Translation Penalty (90% reduction) if languages do not match
					if languageMismatch {
						chance *= 0.10
					}

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
							LanguageID:      sender.culture.LanguageID,
							Value:           int32(secret.SecretID), // Safe because SecretID is uint32 and we use int32
						}

						// Increment ring buffer head
						receiver.memory.Head = (head + 1) % 50

						// Give the receiver the secret as well
						receiver.secret.Secrets = append(receiver.secret.Secrets, components.Secret{
							OriginID: secret.OriginID,
							SecretID: secret.SecretID,
							Virality: secret.Virality,
							BeliefID: secret.BeliefID, // Preserve metadata flag
						})

						// Phase 07.5: Ideological Infection
						// If the secret carries a BeliefID, spread the ideology
						if secret.BeliefID != 0 && receiver.belief != nil {
							found := false
							for k := range receiver.belief.Beliefs {
								if receiver.belief.Beliefs[k].BeliefID == secret.BeliefID {
									receiver.belief.Beliefs[k].Weight += 1 // Linearly modify weight
									found = true
									break
								}
							}
							if !found {
								// First time encountering this belief
								receiver.belief.Beliefs = append(receiver.belief.Beliefs, components.Belief{
									BeliefID: secret.BeliefID,
									Weight:   1,
								})
							}
						}
					} else if languageMismatch {
						// Phase 07.4: Silent Hooks
						// Even if language fails (or gossip fails to pass), physical trades can occur.
						// 25% chance of a "Silent Hook" occurring when there's an overlap but mismatched languages.
						if engine.GetRandomFloat32() < 0.25 {
							if s.HookGraph != nil {
								s.HookGraph.AddHook(sender.ident.ID, receiver.ident.ID, 1)
							}
						}
					}
				}
			}
		}
	}
}
