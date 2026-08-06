package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 34.1: The Information Broker Engine (Information Trade System)
// Phase 34.2: The Lingua Franca Engine
// Treats information (Secrets) as a tangible commodity in the ECS.
// NPCs with low wealth but high-value secrets will explicitly seek out and sell
// unknown secrets to wealthier NPCs in their vicinity, bridging the Memetic and Economic pillars.

type InformationTradeSystem struct {
	tickCounter uint64
	HookGraph   *engine.SparseHookGraph

	// Component IDs
	posID      ecs.ID
	secretID   ecs.ID
	needsID    ecs.ID
	identID    ecs.ID
	ruinID     ecs.ID
	memoryID   ecs.ID
	affilID    ecs.ID
	cultureID  ecs.ID
	treasuryID ecs.ID
}

// NewInformationTradeSystem creates a new InformationTradeSystem.
func NewInformationTradeSystem(world *ecs.World, hookGraph *engine.SparseHookGraph) *InformationTradeSystem {
	return &InformationTradeSystem{
		HookGraph:  hookGraph,
		posID:      ecs.ComponentID[components.Position](world),
		secretID:   ecs.ComponentID[components.SecretComponent](world),
		needsID:    ecs.ComponentID[components.Needs](world),
		identID:    ecs.ComponentID[components.Identity](world),
		ruinID:     ecs.ComponentID[components.RuinComponent](world),
		memoryID:   ecs.ComponentID[components.Memory](world),
		affilID:    ecs.ComponentID[components.Affiliation](world),
		cultureID:  ecs.ComponentID[components.CultureComponent](world),
		treasuryID: ecs.ComponentID[components.TreasuryComponent](world),
	}
}

// nodeTradeData is a flat cache for DOD optimized proximity checking
// cache values, not component pointers — GC corruption class, see banditry.go
// Mutable components (Secret/Needs/Memory/Culture) are re-fetched via the entity handle at use time.
type nodeTradeData struct {
	entity     ecs.Entity
	x          float32
	y          float32
	identID    uint64
	baseTraits uint32
	cityID     uint32
	hasAffil   bool
	hasCulture bool
}

// Update evaluates entities for information trading.
func (s *InformationTradeSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Runs on an offset tick execution to avoid bottlenecking alongside GossipDistributionSystem
	if s.tickCounter%15 != 0 {
		return
	}

	// Pre-cache Treasury wealth by CityID
	// cache values, not component pointers — GC corruption class, see banditry.go
	cityTreasuries := make(map[uint32]float32)
	treasuryQuery := world.Query(ecs.All(s.affilID, s.treasuryID))
	for treasuryQuery.Next() {
		affil := (*components.Affiliation)(treasuryQuery.Get(s.affilID))
		treasury := (*components.TreasuryComponent)(treasuryQuery.Get(s.treasuryID))
		cityTreasuries[affil.CityID] = treasury.Wealth
	}

	filter := ecs.All(s.posID, s.secretID, s.needsID, s.identID, s.memoryID).Without(s.ruinID)
	query := world.Query(&filter)

	var nodes []nodeTradeData

	for query.Next() {
		hasAffil := false
		var cityID uint32
		if world.Has(query.Entity(), s.affilID) {
			affil := (*components.Affiliation)(query.Get(s.affilID))
			hasAffil = true
			cityID = affil.CityID
		}

		hasCulture := world.Has(query.Entity(), s.cultureID)

		pos := (*components.Position)(query.Get(s.posID))
		ident := (*components.Identity)(query.Get(s.identID))

		nodes = append(nodes, nodeTradeData{
			entity:     query.Entity(),
			x:          pos.X,
			y:          pos.Y,
			identID:    ident.ID,
			baseTraits: ident.BaseTraits,
			cityID:     cityID,
			hasAffil:   hasAffil,
			hasCulture: hasCulture,
		})
	}

	for i := 0; i < len(nodes); i++ {
		seller := nodes[i]

		if !world.Alive(seller.entity) {
			continue
		}
		sellerSecret := (*components.SecretComponent)(world.Get(seller.entity, s.secretID))
		sellerNeeds := (*components.Needs)(world.Get(seller.entity, s.needsID))

		// Must have secrets to sell
		if len(sellerSecret.Secrets) == 0 {
			continue
		}

		// Information trading is driven by economic necessity or sheer opportunism (Gossip trait).
		// We only trigger selling if the NPC's wealth is low or they have the Gossip trait.
		isOpportunist := seller.baseTraits&components.TraitGossip != 0
		if sellerNeeds.Wealth >= 100.0 && !isOpportunist {
			continue
		}

		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			buyer := nodes[j]

			if !world.Alive(buyer.entity) {
				continue
			}
			buyerNeeds := (*components.Needs)(world.Get(buyer.entity, s.needsID))

			// Buyer must have wealth to afford information
			if buyerNeeds.Wealth < 10.0 {
				continue
			}

			// Distance check (Squared to avoid sqrt overhead)
			dx := seller.x - buyer.x
			dy := seller.y - buyer.y
			distSq := dx*dx + dy*dy

			// Overlap defined as close proximity (e.g., in a tavern or street corner)
			if distSq < 4.0 {
				// Phase 41: The Ostracization Engine
				// Check for deep grudges before executing a trade. If they hate each other, no trade occurs.
				if s.HookGraph != nil {
					buyerHatesSeller := s.HookGraph.GetHook(buyer.identID, seller.identID)
					sellerHatesBuyer := s.HookGraph.GetHook(seller.identID, buyer.identID)

					if buyerHatesSeller <= -40 || sellerHatesBuyer <= -40 {
						continue // Block trade due to ostracization
					}
				}

				// Find a secret the buyer doesn't know
				traded := false
				buyerSecret := (*components.SecretComponent)(world.Get(buyer.entity, s.secretID))

				for _, secret := range sellerSecret.Secrets {
					alreadyKnown := false
					for _, known := range buyerSecret.Secrets {
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
					if buyerNeeds.Wealth >= value {
						// Execute Trade
						buyerNeeds.Wealth -= value
						sellerNeeds.Wealth += value

						// Transfer Knowledge
						buyerSecret.Secrets = append(buyerSecret.Secrets, components.Secret{
							OriginID: secret.OriginID,
							SecretID: secret.SecretID,
							Virality: secret.Virality,
							BeliefID: secret.BeliefID, // Preserve metadata
						})

						// Memory Telemetry
						buyerMemory := (*components.Memory)(world.Get(buyer.entity, s.memoryID))
						head := buyerMemory.Head
						buyerMemory.Events[head] = components.MemoryEvent{
							TargetID:        uint64(seller.entity.ID()),
							TickStamp:       s.tickCounter,
							InteractionType: components.InteractionGossip,
							LanguageID:      0, // Agnostic for trade
							Value:           int32(secret.SecretID),
						}
						buyerMemory.Head = (head + 1) % 50

						// Positive Social Feedback (A successful transaction builds rapport)
						if s.HookGraph != nil {
							s.HookGraph.AddHook(seller.identID, buyer.identID, 1)
							s.HookGraph.AddHook(buyer.identID, seller.identID, 1)
						}

						// Phase 34.2: The Lingua Franca Engine
						// If the seller's faction is massively wealthier than the buyer's (> 5000 wealth and > 5x buyer's wealth),
						// the seller imposes their LanguageID on the buyer.
						if seller.hasAffil && buyer.hasAffil && seller.hasCulture && buyer.hasCulture {
							sellerTreasuryWealth, sHasTreas := cityTreasuries[seller.cityID]
							buyerTreasuryWealth, bHasTreas := cityTreasuries[buyer.cityID]

							if sHasTreas && bHasTreas {
								if sellerTreasuryWealth > 5000.0 && sellerTreasuryWealth > 5.0*buyerTreasuryWealth {
									// Impose language
									sellerCulture := (*components.CultureComponent)(world.Get(seller.entity, s.cultureID))
									buyerCulture := (*components.CultureComponent)(world.Get(buyer.entity, s.cultureID))
									buyerCulture.ForeignLanguageID = sellerCulture.LanguageID
									buyerCulture.ForeignInteractionTicks += 50
								}
							}
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
