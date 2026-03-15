package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 28.1: The Vassal Rebellion Engine
// VassalRebellionSystem connects Economy (Inflation/Debasement), Sovereignty (Loyalty/Secession), and Justice (Blood Feuds).
// High debasement or extreme local food prices drain a Village's LoyaltyComponent.
// When Loyalty drops to 0, the village secedes. Desperate NPCs inside the seceding village generate
// massive negative hooks (-100) against the Country Capital's ruler, natively triggering the BloodFeudSystem.

const VassalRebellionTickRate = 100

type capitalRulerData struct {
	Debasement float32
	RulerID    uint64
}

type VassalRebellionSystem struct {
	world     *ecs.World
	hooks     *engine.SparseHookGraph
	tickStamp uint64

	// Component IDs
	villageID     ecs.ID
	affilID       ecs.ID
	loyaltyID     ecs.ID
	marketID      ecs.ID
	countryID     ecs.ID
	capitalID     ecs.ID
	identID       ecs.ID
	npcID         ecs.ID
	desperationID ecs.ID
}

// NewVassalRebellionSystem creates a new VassalRebellionSystem.
func NewVassalRebellionSystem(world *ecs.World, hooks *engine.SparseHookGraph) *VassalRebellionSystem {
	return &VassalRebellionSystem{
		world:     world,
		hooks:     hooks,
		tickStamp: 0,

		villageID:     ecs.ComponentID[components.Village](world),
		affilID:       ecs.ComponentID[components.Affiliation](world),
		loyaltyID:     ecs.ComponentID[components.LoyaltyComponent](world),
		marketID:      ecs.ComponentID[components.MarketComponent](world),
		countryID:     ecs.ComponentID[components.CountryComponent](world),
		capitalID:     ecs.ComponentID[components.CapitalComponent](world),
		identID:       ecs.ComponentID[components.Identity](world),
		npcID:         ecs.ComponentID[components.NPC](world),
		desperationID: ecs.ComponentID[components.DesperationComponent](world),
	}
}

// Update executes the system logic every VassalRebellionTickRate ticks.
func (s *VassalRebellionSystem) Update(world *ecs.World) {
	s.tickStamp++

	if s.tickStamp%VassalRebellionTickRate != 0 {
		return
	}

	// 1. Build a flat map of active Countries and their Capital Rulers for O(1) matching.
	capitalDataMap := make(map[uint32]capitalRulerData)

	capitalQuery := world.Query(filter.All(s.countryID, s.capitalID, s.affilID, s.identID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(s.affilID))
		country := (*components.CountryComponent)(capitalQuery.Get(s.countryID))
		ident := (*components.Identity)(capitalQuery.Get(s.identID))

		capitalDataMap[affil.CountryID] = capitalRulerData{
			Debasement: country.Debasement,
			RulerID:    ident.ID,
		}
	}

	// If no countries exist, skip village rebellion logic to save CPU cycles.
	if len(capitalDataMap) == 0 {
		return
	}

	// Track seceded villages to trigger citizen rebellion logic
	secededVillages := make(map[uint32]uint64) // CityID -> RulerID

	// 2. Iterate over all Villages with an Affiliation, LoyaltyComponent, and MarketComponent.
	villageQuery := world.Query(filter.All(s.villageID, s.affilID, s.loyaltyID, s.marketID))
	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))

		if affil.CountryID == 0 {
			continue // Already independent
		}

		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))

		if capData, ok := capitalDataMap[affil.CountryID]; ok {
			// Calculate loyalty drain. High debasement or extreme food prices cause drain.
			var loyaltyDrain uint32 = 0

			// Phase 15.4: Organic Inflation via Debasement
			if capData.Debasement > 0.0 {
				loyaltyDrain += uint32(capData.Debasement * 10)
			}

			// Extreme Food Price (Famine)
			if market.FoodPrice > 5.0 {
				loyaltyDrain += uint32(market.FoodPrice - 5.0)
			}

			if loyaltyDrain > 0 {
				if loyalty.Value >= loyaltyDrain {
					loyalty.Value -= loyaltyDrain
				} else {
					loyalty.Value = 0
				}
			}

			// 3. Rebellion Trigger
			if loyalty.Value == 0 {
				// Phase 28.1: The Vassal Rebellion Engine - Secession!
				secededVillages[affil.CityID] = capData.RulerID
				affil.CountryID = 0 // Village is now independent
			}
		} else {
			// Capital doesn't exist anymore, Country is broken
			affil.CountryID = 0
		}
	}

	// If no villages seceded, we can skip the citizen pass.
	if len(secededVillages) == 0 {
		return
	}

	// 4. Trigger citizen rebellion (massive negative hooks) in newly seceded villages.
	npcQuery := world.Query(filter.All(s.npcID, s.affilID, s.desperationID, s.identID))
	for npcQuery.Next() {
		affil := (*components.Affiliation)(npcQuery.Get(s.affilID))
		desperation := (*components.DesperationComponent)(npcQuery.Get(s.desperationID))
		ident := (*components.Identity)(npcQuery.Get(s.identID))

		if rulerID, seceded := secededVillages[affil.CityID]; seceded {
			// Only highly desperate citizens trigger the active rebellion
			if desperation.Level >= 50 {
				if s.hooks != nil {
					// Phase 28.1: Natively triggers the BloodFeudSystem (Phase 23.1)
					// The citizen forms a massive grudge against the former ruler.
					s.hooks.AddHook(ident.ID, rulerID, -100)
				}
			}
		}
	}
}
