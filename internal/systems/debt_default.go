package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 10.1: Debt Default Execution (The Hook Trap)
// DebtDefaultSystem checks if current_tick >= DueTick and internal StorageComponent
// metrics fail repayment logic. If they fail, it transfers the mapped AssetID
// (e.g., changes CityID or GuildID bounds mapping to Guild arrays). For our
// explicit logic, we transfer the entity's GuildID to the AssetID.

type DebtDefaultSystem struct {
	Tick      uint64
	toRemove  []ecs.Entity
	hookGraph *engine.SparseHookGraph
}

func NewDebtDefaultSystem(hookGraph *engine.SparseHookGraph) *DebtDefaultSystem {
	return &DebtDefaultSystem{
		Tick:      0,
		toRemove:  make([]ecs.Entity, 0, 100),
		hookGraph: hookGraph,
	}
}

func (s *DebtDefaultSystem) Update(world *ecs.World) {
	s.Tick++

	loanID := ecs.ComponentID[components.LoanContractComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)
	affiliationID := ecs.ComponentID[components.Affiliation](world)
	identID := ecs.ComponentID[components.Identity](world)

	// Phase 10.1: Blacklisting Engine pre-cache
	type defaultedDebtor struct {
		ID      uint64
		GuildID uint32
	}

	var defaultedDebtors []defaultedDebtor

	filter := ecs.All(loanID, storageID, affiliationID)
	query := world.Query(filter)

	s.toRemove = s.toRemove[:0] // Clear slice to reuse capacity

	for query.Next() {
		loan := (*components.LoanContractComponent)(query.Get(loanID))

		if s.Tick >= loan.DueTick {
			storage := (*components.StorageComponent)(query.Get(storageID))
			affiliation := (*components.Affiliation)(query.Get(affiliationID))

			// Repayment Logic: 100 total resources needed
			totalStorage := uint64(storage.Food) + uint64(storage.Wood) + uint64(storage.Stone) + uint64(storage.Iron)

			if totalStorage >= 100 {
				// Repayment succeeds: deduct 100 total resources.
				// We deduct from Food first, then Wood, Stone, Iron.
				remainingToDeduct := uint32(100)

				if storage.Food >= remainingToDeduct {
					storage.Food -= remainingToDeduct
					remainingToDeduct = 0
				} else {
					remainingToDeduct -= storage.Food
					storage.Food = 0
				}

				if storage.Wood >= remainingToDeduct {
					storage.Wood -= remainingToDeduct
					remainingToDeduct = 0
				} else {
					remainingToDeduct -= storage.Wood
					storage.Wood = 0
				}

				if storage.Stone >= remainingToDeduct {
					storage.Stone -= remainingToDeduct
					remainingToDeduct = 0
				} else {
					remainingToDeduct -= storage.Stone
					storage.Stone = 0
				}

				if storage.Iron >= remainingToDeduct {
					storage.Iron -= remainingToDeduct
					remainingToDeduct = 0
				} else {
					remainingToDeduct -= storage.Iron
					storage.Iron = 0
				}

			} else {
				// Repayment fails: default logic executes
				affiliation.GuildID = loan.AssetID

				// Pre-cache the defaulted debtor to generate hooks outside the main loop
				if world.Has(query.Entity(), identID) {
					ident := (*components.Identity)(query.Get(identID))
					defaultedDebtors = append(defaultedDebtors, defaultedDebtor{
						ID:      ident.ID,
						GuildID: loan.AssetID, // The Guild they defaulted against (Creditor's Guild)
					})
				}
			}

			// Regardless of success or failure, the contract is evaluated and closed
			s.toRemove = append(s.toRemove, query.Entity())
		}
	}

	// Remove LoanContractComponent from processed entities outside the query loop
	for _, e := range s.toRemove {
		world.Remove(e, loanID)
	}

	// Phase 10.1: Contractual Law & Blacklisting (The Butterfly Effect)
	// Iterate over all NPCs in the Creditor's Guild and generate massive negative hooks against the debtor.
	if len(defaultedDebtors) > 0 && s.hookGraph != nil {
		npcID := ecs.ComponentID[components.NPC](world)
		guildMembersFilter := ecs.All(npcID, affiliationID, identID)
		memberQuery := world.Query(guildMembersFilter)

		for memberQuery.Next() {
			memberAff := (*components.Affiliation)(memberQuery.Get(affiliationID))
			memberIdent := (*components.Identity)(memberQuery.Get(identID))

			if memberAff.GuildID == 0 {
				continue
			}

			for _, debtor := range defaultedDebtors {
				// Generate hook from the member to the debtor, provided they are in the offended guild,
				// and they are not the debtor themselves.
				if memberAff.GuildID == debtor.GuildID && memberIdent.ID != debtor.ID {
					s.hookGraph.AddHook(memberIdent.ID, debtor.ID, -50)
				}
			}
		}
	}
}
