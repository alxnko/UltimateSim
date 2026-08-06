package engine

import (
	"os"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase: round-trip coverage for every component added by the
// grand-strategy save-parity pass (review finding: the parity commit shipped
// with zero test coverage), plus the stale-row deletion regression.
func TestSaveLoadGrandStrategyComponents(t *testing.T) {
	dbPath := "test_save_gs.db"
	os.Remove(dbPath)
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	world := ecs.NewWorld()
	idID := ecs.ComponentID[components.Identity](&world)
	dynID := ecs.ComponentID[components.DynastyComponent](&world)
	plotID := ecs.ComponentID[components.PlotComponent](&world)
	councilID := ecs.ComponentID[components.CouncilComponent](&world)
	diploID := ecs.ComponentID[components.DiplomacyComponent](&world)
	taxID := ecs.ComponentID[components.TaxPolicyComponent](&world)
	pendID := ecs.ComponentID[components.PendingEventsComponent](&world)
	routeID := ecs.ComponentID[components.TradeRouteComponent](&world)

	ent := world.NewEntity(idID, dynID, plotID, councilID, diploID, taxID, pendID)
	(*components.Identity)(world.Get(ent, idID)).ID = 777
	(*components.Identity)(world.Get(ent, idID)).Name = "GSHolder"

	d := (*components.DynastyComponent)(world.Get(ent, dynID))
	d.SpouseID, d.Children, d.Married = 888, 3, true

	p := (*components.PlotComponent)(world.Get(ent, plotID))
	p.TargetID, p.StartTick, p.Progress, p.Power, p.Kind, p.Exposed = 999, 1200, 40, 12, components.PlotSeizeRule, true

	c := (*components.CouncilComponent)(world.Get(ent, councilID))
	c.Steward, c.Marshal, c.Diplomat, c.Spymaster = 1, 2, 3, 4

	dip := (*components.DiplomacyComponent)(world.Get(ent, diploID))
	dip.Relations = []components.CountryRelation{{TargetCountry: 2, Opinion: -55, WarScore: 30, AtWar: true, TruceUntil: 5000, LastActionTick: 100}}

	(*components.TaxPolicyComponent)(world.Get(ent, taxID)).Rate = 35

	pe := (*components.PendingEventsComponent)(world.Get(ent, pendID))
	pe.Events = []components.GameEvent{{ID: 5, Kind: components.EventTaxDemand, ActorID: 777, CityID: 9, Amount: 42, Tick: 300}}

	route := world.NewEntity(routeID)
	r := (*components.TradeRouteComponent)(world.Get(route, routeID))
	r.FromCity, r.ToCity, r.Volume = 11, 22, 10

	tm := &TickManager{World: &world, Ticks: 4321}
	grid := NewMapGrid(10, 10)
	if err := SaveWorld(tm, grid, 1, db); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load into a fresh world.
	loaded := ecs.NewWorld()
	ltm := &TickManager{World: &loaded}
	if err := LoadWorld(ltm, db); err != nil {
		t.Fatalf("load: %v", err)
	}

	lidID := ecs.ComponentID[components.Identity](&loaded)
	ldynID := ecs.ComponentID[components.DynastyComponent](&loaded)
	lplotID := ecs.ComponentID[components.PlotComponent](&loaded)
	lcouncilID := ecs.ComponentID[components.CouncilComponent](&loaded)
	ldiploID := ecs.ComponentID[components.DiplomacyComponent](&loaded)
	ltaxID := ecs.ComponentID[components.TaxPolicyComponent](&loaded)
	lpendID := ecs.ComponentID[components.PendingEventsComponent](&loaded)
	lrouteID := ecs.ComponentID[components.TradeRouteComponent](&loaded)

	var holder ecs.Entity
	found := false
	q := loaded.Query(ecs.All(lidID))
	for q.Next() {
		if (*components.Identity)(q.Get(lidID)).ID == 777 {
			holder = q.Entity()
			found = true
		}
	}
	if !found {
		t.Fatal("holder entity not restored")
	}

	ld := (*components.DynastyComponent)(loaded.Get(holder, ldynID))
	if ld.SpouseID != 888 || ld.Children != 3 || !ld.Married {
		t.Fatalf("dynasty mismatch: %+v", *ld)
	}
	lp := (*components.PlotComponent)(loaded.Get(holder, lplotID))
	if lp.TargetID != 999 || lp.StartTick != 1200 || lp.Progress != 40 || lp.Power != 12 || lp.Kind != components.PlotSeizeRule || !lp.Exposed {
		t.Fatalf("plot mismatch: %+v", *lp)
	}
	lc := (*components.CouncilComponent)(loaded.Get(holder, lcouncilID))
	if lc.Steward != 1 || lc.Marshal != 2 || lc.Diplomat != 3 || lc.Spymaster != 4 {
		t.Fatalf("council mismatch: %+v", *lc)
	}
	ldip := (*components.DiplomacyComponent)(loaded.Get(holder, ldiploID))
	if len(ldip.Relations) != 1 || ldip.Relations[0] != dip.Relations[0] {
		t.Fatalf("diplomacy mismatch: %+v", ldip.Relations)
	}
	if rate := (*components.TaxPolicyComponent)(loaded.Get(holder, ltaxID)).Rate; rate != 35 {
		t.Fatalf("tax rate mismatch: %d", rate)
	}
	lpe := (*components.PendingEventsComponent)(loaded.Get(holder, lpendID))
	if len(lpe.Events) != 1 || lpe.Events[0] != pe.Events[0] {
		t.Fatalf("pending events mismatch: %+v", lpe.Events)
	}

	routes := 0
	rq := loaded.Query(ecs.All(lrouteID))
	for rq.Next() {
		lr := (*components.TradeRouteComponent)(rq.Get(lrouteID))
		if lr.FromCity != 11 || lr.ToCity != 22 || lr.Volume != 10 {
			t.Fatalf("route mismatch: %+v", *lr)
		}
		routes++
	}
	if routes != 1 {
		t.Fatalf("want 1 trade route, got %d", routes)
	}

	// Regression (review finding): a component REMOVED between saves must not
	// resurrect from its stale row on the next load.
	world.Remove(ent, plotID)
	if err := SaveWorld(tm, grid, 1, db); err != nil {
		t.Fatalf("second save: %v", err)
	}
	reloaded := ecs.NewWorld()
	rtm := &TickManager{World: &reloaded}
	if err := LoadWorld(rtm, db); err != nil {
		t.Fatalf("second load: %v", err)
	}
	rplotID := ecs.ComponentID[components.PlotComponent](&reloaded)
	rq2 := reloaded.Query(ecs.All(rplotID))
	stale := 0
	for rq2.Next() {
		stale++
	}
	if stale != 0 {
		t.Fatalf("resolved plot resurrected from stale row: %d plots after reload", stale)
	}
}
