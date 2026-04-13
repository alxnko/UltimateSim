package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 57 - The Class Warfare Engine (Guillotine)
// ClassWarfareSystem bridges Biology (Starvation) and Macroeconomics (Hyperinflation)
// to Social Hierarchy (Hooks/Grudges).
// When peasants starve while the local Ruler hoards food and prices are exorbitant,
// massive negative hooks are organically generated against the Ruler, inevitably triggering Blood Feuds.

type ClassWarfareSystem struct {
	hookGraph   *engine.SparseHookGraph
	tickCounter uint64

	// Pre-cached filters to satisfy DOD iteration speeds
	npcFilter   ecs.Filter
}

func NewClassWarfareSystem(world *ecs.World, hooks *engine.SparseHookGraph) *ClassWarfareSystem {
	npcID := ecs.ComponentID[components.NPC](world)
	needsID := ecs.ComponentID[components.Needs](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	idID := ecs.ComponentID[components.Identity](world)

	npcMask := ecs.All(npcID, needsID, affID, idID)

	return &ClassWarfareSystem{
		hookGraph:   hooks,
		tickCounter: 0,
		npcFilter:   &npcMask,
	}
}

type rulerData struct {
	RulerID   uint64
	FoodHoard uint32
	FoodPrice float32
}

func (s *ClassWarfareSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process logic periodically to save CPU
	if s.tickCounter%50 != 0 {
		return
	}

	if s.hookGraph == nil {
		return
	}

	// DOD O(1) Pre-caching strategy:
	// We map CityID -> RulerData (Ruler ID, City Storage, City Market Price)
	// This avoids nested O(N^2) arche-go queries inside the main loop.

	cityRulers := make(map[uint32]rulerData)

	// 1. Gather all Administration Rulers
	adminID := ecs.ComponentID[components.AdministrationMarker](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	idID := ecs.ComponentID[components.Identity](world)

	adminQuery := world.Query(ecs.All(adminID, affID, idID))
	for adminQuery.Next() {
		aff := (*components.Affiliation)(adminQuery.Get(affID))
		ident := (*components.Identity)(adminQuery.Get(idID))

		// Initialize with Ruler ID, we'll populate market/storage next
		cityRulers[aff.CityID] = rulerData{
			RulerID:   ident.ID,
			FoodHoard: 0,
			FoodPrice: 0.0,
		}
	}

	// 2. Gather City Market & Storage logic (often held by the Village entity)
	villageID := ecs.ComponentID[components.Village](world)
	marketID := ecs.ComponentID[components.MarketComponent](world)
	storID := ecs.ComponentID[components.StorageComponent](world)


	villageQuery := world.Query(ecs.All(villageID, affID, marketID, storID))
	for villageQuery.Next() {
		aff := (*components.Affiliation)(villageQuery.Get(affID))

		// If this village has a known ruler, update its hoard and price
		if data, exists := cityRulers[aff.CityID]; exists {
			market := (*components.MarketComponent)(villageQuery.Get(marketID))
			stor := (*components.StorageComponent)(villageQuery.Get(storID))

			data.FoodPrice = market.FoodPrice
			data.FoodHoard = stor.Food

			cityRulers[aff.CityID] = data
		}
	}

	// 3. Main Loop: Iterate all peasants
	needsID := ecs.ComponentID[components.Needs](world)
	npcQuery := world.Query(s.npcFilter)

	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(affID))
		needs := (*components.Needs)(npcQuery.Get(needsID))
		ident := (*components.Identity)(npcQuery.Get(idID))

		// Look up local ruler
		ruler, hasRuler := cityRulers[aff.CityID]
		if !hasRuler {
			continue
		}

		// Prevent generating hooks against oneself
		if ident.ID == ruler.RulerID {
			continue
		}

		// The Core Formula of Revolution:
		// IF peasant is starving (< 20 food)
		// AND they are too poor to buy food at current market prices
		// AND the Ruler is hoarding massive amounts of food (> 500)

		if needs.Food < 20.0 && needs.Wealth < ruler.FoodPrice && ruler.FoodHoard > 500 {
			// Deep negative hook generated against the sovereign
			// Repeated starvation ticks quickly compound into revolutionary blood feuds.
			s.hookGraph.AddHook(ident.ID, ruler.RulerID, -5)
		}
	}
}
