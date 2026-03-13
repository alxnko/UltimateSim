package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 16.3: Profit-Driven Unification
// TestAdministrationSystem_Deterministic ensures that the administrative hook generation logic
// is perfectly reproducible, deterministic, and works as described.

func TestAdministrationSystem_Deterministic(t *testing.T) {
	world1 := ecs.NewWorld()
	hooks1 := engine.NewSparseHookGraph()
	sys1 := NewAdministrationSystem(&world1, hooks1)

	world2 := ecs.NewWorld()
	hooks2 := engine.NewSparseHookGraph()
	sys2 := NewAdministrationSystem(&world2, hooks2)

	setupWorld := func(w *ecs.World) {
		identID := ecs.ComponentID[components.Identity](w)
		posID := ecs.ComponentID[components.Position](w)
		marketID := ecs.ComponentID[components.MarketComponent](w)
		affilID := ecs.ComponentID[components.Affiliation](w)
		villageID := ecs.ComponentID[components.Village](w)

		// City 1 - High Prices
		city1 := w.NewEntity(villageID, identID, posID, marketID, affilID)
		(*components.Identity)(w.Get(city1, identID)).ID = 101
		pos1 := (*components.Position)(w.Get(city1, posID))
		pos1.X = 10
		pos1.Y = 10
		market1 := (*components.MarketComponent)(w.Get(city1, marketID))
		market1.FoodPrice = 15.0
		market1.WoodPrice = 10.0
		market1.StonePrice = 10.0
		market1.IronPrice = 10.0

		// City 2 - Low Prices (Nearby, < 100 distance, large disparity > 15%)
		city2 := w.NewEntity(villageID, identID, posID, marketID, affilID)
		(*components.Identity)(w.Get(city2, identID)).ID = 202
		pos2 := (*components.Position)(w.Get(city2, posID))
		pos2.X = 15 // Close to city 1 (dx=5, dy=0)
		pos2.Y = 10
		market2 := (*components.MarketComponent)(w.Get(city2, marketID))
		market2.FoodPrice = 2.0
		market2.WoodPrice = 2.0
		market2.StonePrice = 2.0
		market2.IronPrice = 2.0

		// City 3 - Low Prices (Far away, > 100 distance, large disparity but out of range)
		city3 := w.NewEntity(villageID, identID, posID, marketID, affilID)
		(*components.Identity)(w.Get(city3, identID)).ID = 303
		pos3 := (*components.Position)(w.Get(city3, posID))
		pos3.X = 500 // Very far
		pos3.Y = 500
		market3 := (*components.MarketComponent)(w.Get(city3, marketID))
		market3.FoodPrice = 2.0
		market3.WoodPrice = 2.0
		market3.StonePrice = 2.0
		market3.IronPrice = 2.0

		// City 4 - Same prices as City 1 (Nearby, < 100 distance, NO disparity < 15%)
		city4 := w.NewEntity(villageID, identID, posID, marketID, affilID)
		(*components.Identity)(w.Get(city4, identID)).ID = 404
		pos4 := (*components.Position)(w.Get(city4, posID))
		pos4.X = 12
		pos4.Y = 12
		market4 := (*components.MarketComponent)(w.Get(city4, marketID))
		market4.FoodPrice = 15.0
		market4.WoodPrice = 10.0
		market4.StonePrice = 10.0
		market4.IronPrice = 10.0
	}

	setupWorld(&world1)
	setupWorld(&world2)

	// Need to step tickStamp to AdministrationTickRate (1000) for logic to run
	sys1.tickStamp = AdministrationTickRate - 1
	sys2.tickStamp = AdministrationTickRate - 1

	sys1.Update(&world1)
	sys2.Update(&world2)

	// City 1 and City 2 should have a Diplomatic Hook
	hook1_2 := hooks1.GetHook(101, 202)
	if hook1_2 == 0 {
		t.Errorf("Expected City 1 to generate Diplomatic Hook against City 2, got %d", hook1_2)
	}

	hook2_1 := hooks1.GetHook(202, 101)
	if hook2_1 == 0 {
		t.Errorf("Expected City 2 to generate Reciprocal Hook against City 1, got %d", hook2_1)
	}

	// City 1 and City 3 should NOT have a hook (out of range)
	hook1_3 := hooks1.GetHook(101, 303)
	if hook1_3 != 0 {
		t.Errorf("Expected City 1 and 3 to have NO hooks (out of range), got %d", hook1_3)
	}

	// City 1 and City 4 should NOT have a hook (no price disparity)
	hook1_4 := hooks1.GetHook(101, 404)
	if hook1_4 != 0 {
		t.Errorf("Expected City 1 and 4 to have NO hooks (no price disparity), got %d", hook1_4)
	}

	// Determinism Check
	if hooks1.GetHook(101, 202) != hooks2.GetHook(101, 202) {
		t.Errorf("Non-deterministic behavior: World 1 Hook %d != World 2 Hook %d", hooks1.GetHook(101, 202), hooks2.GetHook(101, 202))
	}
}
