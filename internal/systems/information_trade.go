package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 34.1: The Information Broker Engine (Information Trade System)
// Treats information (Secrets) as a tangible commodity in the ECS.
// NPCs with low wealth but high-value secrets will explicitly seek out and sell
// unknown secrets to wealthier NPCs in their vicinity, bridging the Memetic and Economic pillars.

type InformationTradeSystem struct {
	tickCounter uint64
	HookGraph   *engine.SparseHookGraph

	// Component IDs
	posID    ecs.ID
	secretID ecs.ID
	needsID  ecs.ID
	identID  ecs.ID
	ruinID   ecs.ID
	memoryID ecs.ID
}

// NewInformationTradeSystem creates a new InformationTradeSystem.
func NewInformationTradeSystem(world *ecs.World, hookGraph *engine.SparseHookGraph) *InformationTradeSystem {
	return &InformationTradeSystem{
		HookGraph: hookGraph,
		posID:     ecs.ComponentID[components.Position](world),
		secretID:  ecs.ComponentID[components.SecretComponent](world),
		needsID:   ecs.ComponentID[components.Needs](world),
		identID:   ecs.ComponentID[components.Identity](world),
		ruinID:    ecs.ComponentID[components.RuinComponent](world),
		memoryID:  ecs.ComponentID[components.Memory](world),
	}
}

// nodeTradeData is a flat cache for DOD optimized proximity checking
type nodeTradeData struct {
	entity ecs.Entity
	pos    *components.Position
	secret *components.SecretComponent
	needs  *components.Needs
	ident  *components.Identity
	memory *components.Memory
}

// Update evaluates entities for information trading.
func (s *InformationTradeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Runs on an offset tick execution to avoid bottlenecking alongside GossipDistributionSystem
	if s.tickCounter%15 != 0 {
		return
	}

	filter := ecs.All(s.posID, s.secretID, s.needsID, s.identID, s.memoryID).Without(s.ruinID)
	query := world.Query(&filter)

	var nodes []nodeTradeData

	for query.Next() {
		nodes = append(nodes, nodeTradeData{
			entity: query.Entity(),
			pos:    (*components.Position)(query.Get(s.posID)),
			secret: (*components.SecretComponent)(query.Get(s.secretID)),
			needs:  (*components.Needs)(query.Get(s.needsID)),
			ident:  (*components.Identity)(query.Get(s.identID)),
			memory: (*components.Memory)(query.Get(s.memoryID)),
		})
	}

	for i := 0; i < len(nodes); i++ {
		seller := nodes[i]

		// Must have secrets to sell
		if len(seller.secret.Secrets) == 0 {
			continue
		}

		// Information trading is driven by economic necessity or sheer opportunism (Gossip trait).
		// We only trigger selling if the NPC's wealth is low or they have the Gossip trait.
		isOpportunist := seller.ident.BaseTraits&components.TraitGossip != 0
		if seller.needs.Wealth >= 100.0 && !isOpportunist {
			continue
		}

		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			buyer := nodes[j]

			// Buyer must have wealth to afford information
			if buyer.needs.Wealth < 10.0 {
				continue
			}

			// Distance check (Squared to avoid sqrt overhead)
			dx := seller.pos.X - buyer.pos.X
			dy := seller.pos.Y - buyer.pos.Y
			distSq := dx*dx + dy*dy

			// Overlap defined as close proximity (e.g., in a tavern or street corner)
			if distSq < 4.0 {
				// Find a secret the buyer doesn't know
				traded := false

				for _, secret := range seller.secret.Secrets {
					alreadyKnown := false
					for _, known := range buyer.secret.Secrets {
						if known.SecretID == secret.SecretID {
							alreadyKnown = true
							break
						}
					}

					if alreadyKnown {
						continue
					}

					// We found a secret to sell. Calculate market value based on virality.
					// Highly viral secrets are worth more.
					value := float32(secret.Virality) / 10.0
					if value < 5.0 {
						value = 5.0 // Minimum price
					}

					// Can the buyer afford it?
					if buyer.needs.Wealth >= value {
						// Execute Trade
						buyer.needs.Wealth -= value
						seller.needs.Wealth += value

						// Transfer Knowledge
						buyer.secret.Secrets = append(buyer.secret.Secrets, components.Secret{
							OriginID: secret.OriginID,
							SecretID: secret.SecretID,
							Virality: secret.Virality,
							BeliefID: secret.BeliefID, // Preserve metadata
						})

						// Memory Telemetry
						head := buyer.memory.Head
						buyer.memory.Events[head] = components.MemoryEvent{
							TargetID:        uint64(seller.entity.ID()),
							TickStamp:       s.tickCounter,
							InteractionType: components.InteractionGossip,
							LanguageID:      0, // Agnostic for trade
							Value:           int32(secret.SecretID),
						}
						buyer.memory.Head = (head + 1) % 50

						// Positive Social Feedback (A successful transaction builds rapport)
						if s.HookGraph != nil {
							s.HookGraph.AddHook(seller.ident.ID, buyer.ident.ID, 1)
							s.HookGraph.AddHook(buyer.ident.ID, seller.ident.ID, 1)
						}

						traded = true
						break // Only sell one secret per interaction
					}
				}

				if traded && !isOpportunist {
					break // If they sold a secret to survive, move on to the next seller
				}
			}
		}
	}
}
