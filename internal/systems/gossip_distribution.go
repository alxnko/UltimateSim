package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
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
	// cache values, not component pointers — GC corruption class, see banditry.go
	// Mutable components (Secret/Memory/Belief) are re-fetched via the entity handle at use time.
	type nodeData struct {
		entity         ecs.Entity
		x              float32
		y              float32
		identID        uint64
		baseTraits     uint32
		languageID     uint16
		hasBelief      bool
		hasEquipment   bool
		weaponPrestige uint32
	}

	// Runs on a slower tick execution (every 10 Ticks)
	if s.tickCounter%10 != 0 {
		return
	}

	equipID := ecs.ComponentID[components.EquipmentComponent](world)

	// Filter all valid actors capable of gossiping
	filter := ecs.All(s.posID, s.secretID, s.memoryID, s.identID, s.cultureID).Without(s.ruinID)
	query := world.Query(&filter)

	// Extract into a flat slice cache to prevent nested O(N^2) arche queries
	// and preserve L1/L2 hits during the proximity loop.
	var nodes []nodeData

	for query.Next() {
		hasBelief := query.Has(s.beliefID)

		hasEquipment := false
		var weaponPrestige uint32
		if query.Has(equipID) {
			equip := (*components.EquipmentComponent)(query.Get(equipID))
			if equip.Equipped {
				hasEquipment = true
				weaponPrestige = equip.Weapon.Prestige
			}
		}

		pos := (*components.Position)(query.Get(s.posID))
		ident := (*components.Identity)(query.Get(s.identID))
		culture := (*components.CultureComponent)(query.Get(s.cultureID))

		nodes = append(nodes, nodeData{
			entity:         query.Entity(),
			x:              pos.X,
			y:              pos.Y,
			identID:        ident.ID,
			baseTraits:     ident.BaseTraits,
			languageID:     culture.LanguageID,
			hasBelief:      hasBelief,
			hasEquipment:   hasEquipment,
			weaponPrestige: weaponPrestige,
		})
	}

	// O(N^2) proximity check across the flat slice cache
	// In the future this should utilize spatial partitioning, but for now we iterate sequentially
	for i := 0; i < len(nodes); i++ {
		sender := nodes[i]

		if !world.Alive(sender.entity) {
			continue
		}
		senderSecret := (*components.SecretComponent)(world.Get(sender.entity, s.secretID))

		if len(senderSecret.Secrets) == 0 {
			continue
		}

		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			receiver := nodes[j]

			// Distance check (Squared to avoid sqrt overhead)
			dx := sender.x - receiver.x
			dy := sender.y - receiver.y
			distSq := dx*dx + dy*dy

			// Overlap defined as distance < 2.0 (distSq < 4.0)
			if distSq < 4.0 {
				if !world.Alive(receiver.entity) {
					continue
				}
				receiverSecret := (*components.SecretComponent)(world.Get(receiver.entity, s.secretID))
				receiverMemory := (*components.Memory)(world.Get(receiver.entity, s.memoryID))
				var receiverBelief *components.BeliefComponent
				if receiver.hasBelief {
					receiverBelief = (*components.BeliefComponent)(world.Get(receiver.entity, s.beliefID))
				}

				languageMismatch := sender.languageID != receiver.languageID

				// Evaluate each secret the sender holds
				for _, secret := range senderSecret.Secrets {
					// Check if receiver already knows the secret
					alreadyKnown := false
					for _, known := range receiverSecret.Secrets {
						if known.SecretID == secret.SecretID {
							alreadyKnown = true
							break
						}
					}

					if alreadyKnown {
						continue
					}

					// Calculate chance
					originalChance := float32(secret.Virality) / 255.0
					chance := originalChance

					// Apply Phase 07.4 Translation Penalty (90% reduction) if languages do not match
					if languageMismatch {
						chance *= 0.10
					}

					// Apply TraitGossip modifier
					modifier := float32(1.0)
					if sender.baseTraits&components.TraitGossip != 0 {
						modifier = 2.0
					}

					// Phase 32.1: Aura of Legitimacy
					// A highly prestigious artifact multiplies your influence
					if sender.hasEquipment && sender.weaponPrestige >= components.ExtremePrestigeThreshold {
						modifier *= 3.0
					}

					roll := engine.GetRandomFloat32()

					// RNG Pass
					if roll < chance*modifier {
						// Pass the secret

						// Inject SecretID into neighbor's MemoryComponent buffer
						head := receiverMemory.Head
						receiverMemory.Events[head] = components.MemoryEvent{
							TargetID:        uint64(sender.entity.ID()), // Storing ECS entity ID for reference
							TickStamp:       s.tickCounter,
							InteractionType: components.InteractionGossip,
							LanguageID:      sender.languageID,
							Value:           int32(secret.SecretID), // Safe because SecretID is uint32 and we use int32
						}

						// Increment ring buffer head
						receiverMemory.Head = (head + 1) % 50

						// Give the receiver the secret as well
						receiverSecret.Secrets = append(receiverSecret.Secrets, components.Secret{
							OriginID: secret.OriginID,
							SecretID: secret.SecretID,
							Virality: secret.Virality,
							BeliefID: secret.BeliefID, // Preserve metadata flag
						})

						// Phase 07.5: Ideological Infection
						// If the secret carries a BeliefID, spread the ideology
						if secret.BeliefID != 0 && receiverBelief != nil {
							found := false
							for k := range receiverBelief.Beliefs {
								if receiverBelief.Beliefs[k].BeliefID == secret.BeliefID {
									receiverBelief.Beliefs[k].Weight += 1 // Linearly modify weight
									found = true
									break
								}
							}
							if !found {
								// First time encountering this belief
								receiverBelief.Beliefs = append(receiverBelief.Beliefs, components.Belief{
									BeliefID: secret.BeliefID,
									Weight:   1,
								})
							}
						}
					} else if languageMismatch {
						// Phase 07.4: Misunderstandings & Translation Penalties
						// If the roll would have passed normally, but failed purely due to the translation penalty,
						// a Misunderstanding occurs. A mutated secret is passed and a negative hook is generated.
						if roll < originalChance*modifier {
							// Generate mutated secret
							secretStr, exists := engine.GetSecretRegistry().GetSecret(secret.SecretID)
							if !exists {
								secretStr = "unknown"
							}
							misunderstoodStr := "misunderstood_" + secretStr
							misunderstoodID := engine.GetSecretRegistry().RegisterSecret(misunderstoodStr)

							receiverSecret.Secrets = append(receiverSecret.Secrets, components.Secret{
								OriginID: receiver.identID, // The receiver originated this mutated rumor
								SecretID: misunderstoodID,
								Virality: secret.Virality,
								BeliefID: secret.BeliefID,
							})

							// Systemic Emergence: Misunderstandings cause diplomatic friction
							// Receiver gains a negative hook against the sender (-10) due to offensive misunderstanding
							if s.HookGraph != nil {
								s.HookGraph.AddHook(receiver.identID, sender.identID, -10)
							}
						} else {
							// Phase 07.4: Silent Hooks
							// Even if language fails completely, physical trades can occur.
							// 25% chance of a "Silent Hook" occurring when there's an overlap but mismatched languages.
							if engine.GetRandomFloat32() < 0.25 {
								if s.HookGraph != nil {
									s.HookGraph.AddHook(sender.identID, receiver.identID, 1)
								}
							}
						}
					}
				}
			}
		}
	}
}
