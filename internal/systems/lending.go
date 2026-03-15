package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 15.3 - Predatory Lending Engine
// Links Economic Agency (Phase 15.1) directly to State Failure (Phase 10.1).
// Wealthy NPCs (Creditors) actively identify starving/desperate NPCs and issue
// predatory loans. The LoanContractComponent forces repayment. If the Debtor
// defaults when the DueTick arrives, the DebtDefaultSystem structurally transfers
// their Affiliation.GuildID, functionally reducing them to indentured servants.

type lendingNodeData struct {
	entity  ecs.Entity
	id      uint64
	guildID uint32
	wealth  float32
	x       float32
	y       float32
}

type LendingSystem struct {
	tickCounter uint64

	creditorFilter ecs.Filter
	debtorFilter   ecs.Filter

	// Component IDs
	posID     ecs.ID
	needsID   ecs.ID
	affID     ecs.ID
	identID   ecs.ID
	despID    ecs.ID
	loanID    ecs.ID
	storageID ecs.ID
}

func NewLendingSystem(world *ecs.World) *LendingSystem {
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	identID := ecs.ComponentID[components.Identity](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	loanID := ecs.ComponentID[components.LoanContractComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)

	// Creditors: Must have Position, Needs, Affiliation, Identity.
	// We will manually filter high wealth and GuildID > 0 in the query loop.
	creditorMask := ecs.All(posID, needsID, affID, identID)

	// Debtors: Must have Position, Needs, Affiliation, Identity, Desperation.
	// Crucially, they must NOT already have an active LoanContractComponent to prevent double-dipping.
	debtorMask := ecs.All(posID, needsID, affID, identID, despID).Without(loanID)

	return &LendingSystem{
		tickCounter:    0,
		creditorFilter: &creditorMask,
		debtorFilter:   &debtorMask,
		posID:          posID,
		needsID:        needsID,
		affID:          affID,
		identID:        identID,
		despID:         despID,
		loanID:         loanID,
		storageID:      storageID,
	}
}

func (s *LendingSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Run every 10 ticks to avoid O(N^2) spam on every frame
	if s.tickCounter%10 != 0 {
		return
	}

	// 1. Pre-cache all potential Creditors into a flat DOD slice.
	creditorQuery := world.Query(s.creditorFilter)
	creditors := make([]lendingNodeData, 0, 50)

	for creditorQuery.Next() {
		needs := (*components.Needs)(creditorQuery.Get(s.needsID))
		aff := (*components.Affiliation)(creditorQuery.Get(s.affID))

		// A Creditor must have extreme wealth and belong to a Guild (so they can seize assets)
		if needs.Wealth >= 500.0 && aff.GuildID != 0 {
			pos := (*components.Position)(creditorQuery.Get(s.posID))
			ident := (*components.Identity)(creditorQuery.Get(s.identID))

			creditors = append(creditors, lendingNodeData{
				entity:  creditorQuery.Entity(),
				id:      ident.ID,
				guildID: aff.GuildID,
				wealth:  needs.Wealth,
				x:       pos.X,
				y:       pos.Y,
			})
		}
	}

	if len(creditors) == 0 {
		return // Fast exit if no wealthy creditors exist
	}

	// 2. Iterate over potential Debtors
	// Debtors are starving/desperate NPCs lacking wealth.
	debtorQuery := world.Query(s.debtorFilter)

	type loanIssuance struct {
		DebtorEntity ecs.Entity
		CreditorID   uint64
		AssetID      uint32
	}

	var issuances []loanIssuance

	for debtorQuery.Next() {
		needs := (*components.Needs)(debtorQuery.Get(s.needsID))
		desp := (*components.DesperationComponent)(debtorQuery.Get(s.despID))

		// Debtor condition: Poor and actively desperate (close to committing a crime)
		if needs.Wealth < 50.0 && desp.Level >= 20 {
			pos := (*components.Position)(debtorQuery.Get(s.posID))

			// Find the nearest Creditor
			var bestCreditor *lendingNodeData
			var bestDist float32 = 9999999.0
			var bestIdx int = -1

			for i := 0; i < len(creditors); i++ {
				c := &creditors[i]

				// Creditor must have enough wealth to issue a 100 coin loan
				if c.wealth < 100.0 {
					continue
				}

				dx := pos.X - c.x
				dy := pos.Y - c.y
				distSq := dx*dx + dy*dy

				// Check proximity (e.g., within 5 tiles)
				if distSq < 25.0 && distSq < bestDist {
					bestDist = distSq
					bestCreditor = c
					bestIdx = i
				}
			}

			if bestCreditor != nil {
				// Execute physical wealth transfer immediately
				loanAmount := float32(100.0)
				bestCreditor.wealth -= loanAmount

				// The Creditor entity's Needs component is updated in the real ECS array
				cNeeds := (*components.Needs)(world.Get(bestCreditor.entity, s.needsID))
				cNeeds.Wealth -= loanAmount

				needs.Wealth += loanAmount
				desp.Level = 0 // The Debtor is temporarily saved from starvation/crime

				// Queue the structural modification
				issuances = append(issuances, loanIssuance{
					DebtorEntity: debtorQuery.Entity(),
					CreditorID:   bestCreditor.id,
					AssetID:      bestCreditor.guildID,
				})

				// Update our local cache so one creditor doesn't infinitely lend beyond their actual wealth
				creditors[bestIdx].wealth = bestCreditor.wealth
			}
		}
	}

	// 3. Apply structural ECS modifications (adding LoanContractComponent) outside the query loop
	for _, issuance := range issuances {
		if world.Alive(issuance.DebtorEntity) {
			// A Debtor must have a StorageComponent for DebtDefaultSystem to evaluate their repayment.
			// Add it if they don't have it.
			if !world.Has(issuance.DebtorEntity, s.storageID) {
				world.Add(issuance.DebtorEntity, s.storageID)
			}

			world.Add(issuance.DebtorEntity, s.loanID)

			loan := (*components.LoanContractComponent)(world.Get(issuance.DebtorEntity, s.loanID))
			loan.CreditorID = issuance.CreditorID
			loan.AssetID = issuance.AssetID // The Creditor's GuildID
			loan.DueTick = s.tickCounter + 1000 // Loan comes due in 1000 ticks
		}
	}
}
