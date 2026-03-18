package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.1: The Country Entity (Macro-State)
// Evolution: Phase 42 - The Tax Evasion Engine
type capitalTaxData struct {
	Treasury   *components.TreasuryComponent
	Corruption uint32
	RulerID    uint64
}

type npcTaxData struct {
	IdentityID uint64
	CityID     uint32
}

// TaxationSystem manages the transfer of wealth from sub-cities to their governing Country Capital.
type TaxationSystem struct {
	world *ecs.World
	hooks *engine.SparseHookGraph

	// Component IDs
	countryID  ecs.ID
	capitalID  ecs.ID
	affilID    ecs.ID
	treasuryID ecs.ID
	villageID  ecs.ID
	marketID   ecs.ID
	loyaltyID  ecs.ID
	jurID      ecs.ID
	npcID      ecs.ID
	identID    ecs.ID

	tickStamp uint64
}

// NewTaxationSystem initializes the taxation loop.
func NewTaxationSystem(world *ecs.World, hooks *engine.SparseHookGraph) *TaxationSystem {
	return &TaxationSystem{
		world:      world,
		hooks:      hooks,
		countryID:  ecs.ComponentID[components.CountryComponent](world),
		capitalID:  ecs.ComponentID[components.CapitalComponent](world),
		affilID:    ecs.ComponentID[components.Affiliation](world),
		treasuryID: ecs.ComponentID[components.TreasuryComponent](world),
		villageID:  ecs.ComponentID[components.Village](world),
		marketID:   ecs.ComponentID[components.MarketComponent](world),
		loyaltyID:  ecs.ComponentID[components.LoyaltyComponent](world),
		jurID:      ecs.ComponentID[components.JurisdictionComponent](world),
		npcID:      ecs.ComponentID[components.NPC](world),
		identID:    ecs.ComponentID[components.Identity](world),
	}
}

// Update processes taxation strictly every 100 ticks to avoid simulation loops lag.
func (s *TaxationSystem) Update() {
	s.tickStamp++

	if s.tickStamp%100 != 0 {
		return
	}

	// 1. Build a flat map of active Country Treasuries for O(1) matching.
	capitalDataMap := make(map[uint32]capitalTaxData)

	// A Country Capital must have CountryComponent, CapitalComponent, Affiliation, TreasuryComponent, JurisdictionComponent, and Identity.
	capitalQuery := s.world.Query(filter.All(s.countryID, s.capitalID, s.affilID, s.treasuryID, s.jurID, s.identID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(s.affilID))
		treasury := (*components.TreasuryComponent)(capitalQuery.Get(s.treasuryID))
		jur := (*components.JurisdictionComponent)(capitalQuery.Get(s.jurID))
		ident := (*components.Identity)(capitalQuery.Get(s.identID))

		// Map the capital data using the CountryID
		capitalDataMap[affil.CountryID] = capitalTaxData{
			Treasury:   treasury,
			Corruption: jur.Corruption,
			RulerID:    ident.ID,
		}
	}

	// If no countries exist, skip village taxation completely to save CPU cycles.
	if len(capitalDataMap) == 0 {
		return
	}

	// Evolution: Phase 42 - The Tax Evasion Engine
	// Pre-cache all NPCs for rapid iteration to prevent nested ECS queries during evasion logic.
	npcQuery := s.world.Query(filter.All(s.npcID, s.identID, s.affilID))
	var npcsData []npcTaxData
	for npcQuery.Next() {
		npcIdent := (*components.Identity)(npcQuery.Get(s.identID))
		npcAffil := (*components.Affiliation)(npcQuery.Get(s.affilID))
		npcsData = append(npcsData, npcTaxData{
			IdentityID: npcIdent.ID,
			CityID:     npcAffil.CityID,
		})
	}

	// 2. Iterate over all Villages with an Affiliation, MarketComponent, TreasuryComponent, and LoyaltyComponent.
	villageQuery := s.world.Query(filter.All(s.villageID, s.affilID, s.marketID, s.treasuryID, s.loyaltyID))
	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))
		market := (*components.MarketComponent)(villageQuery.Get(s.marketID))
		treasury := (*components.TreasuryComponent)(villageQuery.Get(s.treasuryID))
		loyalty := (*components.LoyaltyComponent)(villageQuery.Get(s.loyaltyID))

		// Check if the village belongs to a valid country.
		if countryData, ok := capitalDataMap[affil.CountryID]; ok {

			// Evolution: Phase 42 - The Tax Evasion Engine
			// If village loyalty is less than state corruption, they refuse to pay.
			if loyalty.Value < countryData.Corruption {
				if s.hooks != nil {
					// Iterate through pre-cached NPCs and issue grudges against the Capital Ruler
					for i := 0; i < len(npcsData); i++ {
						if npcsData[i].CityID == affil.CityID {
							s.hooks.AddHook(npcsData[i].IdentityID, countryData.RulerID, -50)
						}
					}
				}
			} else {
				// Sub-cities transfer a portion of their MarketComponent revenue to the Country's TreasuryComponent.
				// Calculate tax base linearly off current local market prices (e.g. higher demand = higher tax).
				taxAmount := (market.FoodPrice + market.WoodPrice + market.StonePrice + market.IronPrice) * 1.0

				// Deduct the tax if the Village has sufficient wealth.
				if treasury.Wealth >= taxAmount {
					treasury.Wealth -= taxAmount
					countryData.Treasury.Wealth += taxAmount
				} else {
					// If insufficient wealth, drain whatever is left.
					countryData.Treasury.Wealth += treasury.Wealth
					treasury.Wealth = 0.0
				}
			}
		}
	}
}
