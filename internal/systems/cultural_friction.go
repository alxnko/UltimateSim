package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 33.1: Cultural Friction & Ideological Secession Engine
// This system bridges Phase 07 (Linguistic Drift & Memetics) and Phase 16.4 (Administrative Reach).
// It evaluates the cultural distance between a Capital city and its vassal Villages.
// If a Village speaks a different LanguageID or holds a different primary BeliefID,
// it suffers localized resentment (LoyaltyComponent drain).
// This naturally triggers the VassalRebellionSystem (Phase 28.1), fracturing massive empires
// along ethnic and religious lines unless they actively use Propaganda (Phase 04.5) to assimilate them.

type adminCultureData struct {
	CountryID      uint32
	LanguageID     uint16
	PrimaryBelief  uint32
}

type CulturalFrictionSystem struct {
	capitalData []adminCultureData
	tickCounter uint64
}

// NewCulturalFrictionSystem creates a new CulturalFrictionSystem.
func NewCulturalFrictionSystem() *CulturalFrictionSystem {
	return &CulturalFrictionSystem{
		capitalData: make([]adminCultureData, 0, 20),
		tickCounter: 0,
	}
}

// Update executes the system logic every tick, but throttles processing to every 50 ticks.
func (s *CulturalFrictionSystem) Update(world *ecs.World) {
	s.tickCounter++
	if s.tickCounter%50 != 0 {
		return
	}

	capID := ecs.ComponentID[components.CapitalComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	cultID := ecs.ComponentID[components.CultureComponent](world)
	beliefID := ecs.ComponentID[components.BeliefComponent](world)

	// Step 1: Cache the cultural profile of all active Capitals to avoid nested queries.
	s.capitalData = s.capitalData[:0]

	capFilter := ecs.All(capID, affID, cultID, beliefID)
	capQuery := world.Query(&capFilter)

	for capQuery.Next() {
		aff := (*components.Affiliation)(capQuery.Get(affID))
		cult := (*components.CultureComponent)(capQuery.Get(cultID))
		beliefs := (*components.BeliefComponent)(capQuery.Get(beliefID))

		// Find dominant belief
		var domBelief uint32 = 0
		var maxWeight int32 = -1
		for i := 0; i < len(beliefs.Beliefs); i++ {
			if beliefs.Beliefs[i].Weight > maxWeight {
				maxWeight = beliefs.Beliefs[i].Weight
				domBelief = beliefs.Beliefs[i].BeliefID
			}
		}

		s.capitalData = append(s.capitalData, adminCultureData{
			CountryID:     aff.CountryID,
			LanguageID:    cult.LanguageID,
			PrimaryBelief: domBelief,
		})
	}

	if len(s.capitalData) == 0 {
		return // No active empires to evaluate
	}

	// Step 2: Iterate through all vassal Villages and evaluate friction
	villageID := ecs.ComponentID[components.Village](world)
	loyaltyID := ecs.ComponentID[components.LoyaltyComponent](world)

	villFilter := ecs.All(villageID, affID, loyaltyID, cultID, beliefID).Without(capID)
	villQuery := world.Query(&villFilter)

	for villQuery.Next() {
		aff := (*components.Affiliation)(villQuery.Get(affID))

		if aff.CountryID == 0 {
			continue // Independent villages don't have friction with an overlord
		}

		// Find the overlord's cultural profile
		var overlord *adminCultureData
		for i := 0; i < len(s.capitalData); i++ {
			if s.capitalData[i].CountryID == aff.CountryID {
				overlord = &s.capitalData[i]
				break
			}
		}

		if overlord == nil {
			continue // Orphaned vassal (should be handled by administrative fracture)
		}

		cult := (*components.CultureComponent)(villQuery.Get(cultID))
		beliefs := (*components.BeliefComponent)(villQuery.Get(beliefID))
		loyalty := (*components.LoyaltyComponent)(villQuery.Get(loyaltyID))

		// Find village dominant belief
		var villDomBelief uint32 = 0
		var maxWeight int32 = -1
		for i := 0; i < len(beliefs.Beliefs); i++ {
			if beliefs.Beliefs[i].Weight > maxWeight {
				maxWeight = beliefs.Beliefs[i].Weight
				villDomBelief = beliefs.Beliefs[i].BeliefID
			}
		}

		// Evaluate friction penalties
		frictionPenalty := uint32(0)

		// Linguistic Drift Friction (Phase 07.3)
		if cult.LanguageID != overlord.LanguageID {
			frictionPenalty += 2
		}

		// Religious/Ideological Friction (Phase 07.5)
		if villDomBelief != overlord.PrimaryBelief {
			frictionPenalty += 3
		}

		// Apply Loyalty Drain
		if frictionPenalty > 0 {
			if loyalty.Value > frictionPenalty {
				loyalty.Value -= frictionPenalty
			} else {
				loyalty.Value = 0 // Will trigger VassalRebellionSystem next tick
			}
		}
	}
}
