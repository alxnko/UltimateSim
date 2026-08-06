package ui

import (
	"strings"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/ALXNKO/UltimateSim/internal/render"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Logic-only tests for the strategy-panel chrome (chrome.go,
// panel_diplomacy.go, panel_market.go). Rendering paths are exercised in the
// real game; everything here runs headless on real ecs.Worlds or pure data.

// resetPanelState zeroes the package-level panel UI state so -count=2 runs
// and test order stay independent.
func resetPanelState() {
	diploFilter = SearchableList{}
	diploSelected = 0
	diploTable.SortCol, diploTable.SortAsc = 0, true
	marketFilter = SearchableList{}
	marketTable.SortCol, marketTable.SortAsc = 0, true
	routePickList = SearchableList{}
	routePicking = false
	routeFromCity = 0
}

// newUICapital builds a capital settlement: Village + Capital + Country +
// Affiliation + Treasury + Identity, named and bound to (countryID, cityID).
func newUICapital(t *testing.T, world *ecs.World, countryID, cityID uint32, name string, wealth float32) ecs.Entity {
	t.Helper()
	villageID := ecs.ComponentID[components.Village](world)
	capID := ecs.ComponentID[components.CapitalComponent](world)
	ccID := ecs.ComponentID[components.CountryComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	treasID := ecs.ComponentID[components.TreasuryComponent](world)
	identID := ecs.ComponentID[components.Identity](world)

	e := world.NewEntity(villageID, capID, ccID, affID, treasID, identID)
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.CountryID = countryID
	aff.CityID = cityID
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = uint64(cityID)
	ident.Name = name
	treas := (*components.TreasuryComponent)(world.Get(e, treasID))
	treas.Wealth = wealth
	return e
}

// newUIVillage builds a plain named village (Village + Identity).
func newUIVillage(t *testing.T, world *ecs.World, cityID uint32, name string) ecs.Entity {
	t.Helper()
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	e := world.NewEntity(villageID, identID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = uint64(cityID)
	ident.Name = name
	return e
}

// newUIStatePlaying wraps a world in the minimal StatePlaying harness the
// chrome helpers need (no rendering, no engine build).
func newUIStatePlaying(world *ecs.World) *StatePlaying {
	st := &render.LoadingStatus{TM: &engine.TickManager{World: world}}
	return &StatePlaying{Status: st, PC: &PlayerContext{Status: st}}
}

func TestProgressionHint(t *testing.T) {
	cases := []struct {
		rank      uint8
		city      string
		canClaim  bool
		isCapital bool
		want      string
	}{
		{systems.RankSovereign, "Foo", false, true, "Direct your realm: diplomacy, taxes, war"},
		{systems.RankRuler, "Foo", false, false, "Grow legitimacy; your capital awaits"},
		{systems.RankRuler, "Foo", false, true, "You rule the realm"},
		{systems.RankCitizen, "", false, false, "Find a city to call home"},
		{systems.RankCitizen, "Greenfork", false, false, "Win friends to claim leadership of Greenfork"},
		{systems.RankCitizen, "Greenfork", true, false, "Claim leadership of Greenfork - you are ready!"},
	}
	for _, c := range cases {
		got := progressionHint(c.rank, c.city, c.canClaim, c.isCapital)
		if got != c.want {
			t.Errorf("progressionHint(%d,%q,%v,%v) = %q, want %q",
				c.rank, c.city, c.canClaim, c.isCapital, got, c.want)
		}
	}
}

func TestDiploStatusAndRank(t *testing.T) {
	war := components.CountryRelation{AtWar: true}
	ally := components.CountryRelation{Alliance: true}
	truce := components.CountryRelation{TruceUntil: 100}
	peace := components.CountryRelation{}

	if got := diploStatus(war, 50); !strings.Contains(got, "AT WAR") {
		t.Errorf("war status = %q", got)
	}
	if got := diploStatus(ally, 50); !strings.Contains(got, "Allied") {
		t.Errorf("ally status = %q", got)
	}
	if got := diploStatus(truce, 50); !strings.Contains(got, "Truce") {
		t.Errorf("truce status = %q", got)
	}
	// A truce that has expired (tick >= TruceUntil) is plain peace.
	if got := diploStatus(truce, 100); !strings.Contains(got, "Peace") {
		t.Errorf("expired truce status = %q", got)
	}
	if got := diploStatus(peace, 50); !strings.Contains(got, "Peace") {
		t.Errorf("peace status = %q", got)
	}

	// Sort ranking: wars first, then truce, peace, alliance.
	order := []int{
		diploStatusRank(war, 50),
		diploStatusRank(truce, 50),
		diploStatusRank(peace, 50),
		diploStatusRank(ally, 50),
	}
	for i := 1; i < len(order); i++ {
		if order[i-1] >= order[i] {
			t.Fatalf("status ranks not strictly increasing: %v", order)
		}
	}
}

func TestDiploCells(t *testing.T) {
	row := diploRow{
		rel:     components.CountryRelation{TargetCountry: 2, Opinion: 10},
		realm:   "Realm 2",
		capital: "Bar",
	}
	cells := diploCells(row, 100, 0)
	want := []string{"Realm 2", "Bar", "+10", "  Peace", "-"}
	if len(cells) != len(want) {
		t.Fatalf("cells = %v", cells)
	}
	for i := range want {
		if cells[i] != want[i] {
			t.Errorf("cell[%d] = %q, want %q", i, cells[i], want[i])
		}
	}

	// Selected realms carry the marker; wars surface their score.
	row.rel.AtWar = true
	row.rel.WarScore = -15
	cells = diploCells(row, 100, 2)
	if cells[0] != "> Realm 2" {
		t.Errorf("selected marker missing: %q", cells[0])
	}
	if cells[4] != "-15" {
		t.Errorf("war score cell = %q, want -15", cells[4])
	}
}

func TestSortDiploIndices(t *testing.T) {
	rows := []diploRow{
		{rel: components.CountryRelation{TargetCountry: 2, Opinion: 50}, capital: "Bcap"},
		{rel: components.CountryRelation{TargetCountry: 3, Opinion: -20, AtWar: true, WarScore: 15}, capital: "Acap"},
		{rel: components.CountryRelation{TargetCountry: 5, Opinion: 50}, capital: "Ccap"},
	}
	check := func(name string, got, want []int) {
		t.Helper()
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v want %v", name, got, want)
			}
		}
	}

	idx := allIndices(3)
	sortDiploIndices(idx, rows, 100, 2, true) // opinion ascending
	check("opinion asc", idx, []int{1, 0, 2})  // tie 50/50 broken by country 2 < 5

	idx = allIndices(3)
	sortDiploIndices(idx, rows, 100, 2, false) // opinion descending
	check("opinion desc", idx, []int{0, 2, 1}) // tie-break stays ascending by country

	idx = allIndices(3)
	sortDiploIndices(idx, rows, 100, 3, true) // status: war first
	check("status asc", idx, []int{1, 0, 2})

	idx = allIndices(3)
	sortDiploIndices(idx, rows, 100, 1, true) // capital name
	check("capital asc", idx, []int{1, 0, 2})

	idx = allIndices(3)
	sortDiploIndices(idx, rows, 100, 0, true) // realm = country id
	check("realm asc", idx, []int{0, 1, 2})
}

func TestBuildDiploRowsRealWorld(t *testing.T) {
	resetPanelState()
	world := ecs.NewWorld()
	newUICapital(t, &world, 1, 10, "Foo", 100)
	newUICapital(t, &world, 2, 20, "Bar", 100)

	if err := systems.ImproveRelations(&world, 1, 2, 50); err != nil {
		t.Fatalf("ImproveRelations: %v", err)
	}
	rels := systems.GetRelations(&world, 1)
	if len(rels) != 1 || rels[0].TargetCountry != 2 {
		t.Fatalf("relations = %+v", rels)
	}
	rows := buildDiploRows(&world, rels)
	if rows[0].realm != "Realm 2" || rows[0].capital != "Bar" {
		t.Fatalf("row = %+v", rows[0])
	}
	cells := diploCells(rows[0], 100, 0)
	if cells[2] != "+10" {
		t.Errorf("opinion cell = %q, want +10", cells[2])
	}

	// Unknown countries resolve to the "-" capital placeholder.
	if got := countryCapitalCityName(&world, 99); got != "-" {
		t.Errorf("missing capital name = %q, want -", got)
	}
	if got := countryCapitalCityName(&world, 2); got != "Bar" {
		t.Errorf("capital name = %q, want Bar", got)
	}
}

func TestMarketRowAndSort(t *testing.T) {
	snap := []systems.CityPrices{
		{CityID: 1, Food: 3, Wood: 9},
		{CityID: 2, Food: 1, Wood: 5},
		{CityID: 3, Food: 3, Wood: 7},
	}
	names := []string{"b", "a", "c"}

	row := marketRow("Foo", systems.CityPrices{CityID: 1, Food: 1.25, Wood: 2, Stone: 3, Iron: 4})
	want := []string{"Foo", "1.2", "2.0", "3.0", "4.0"}
	for i := range want {
		if row[i] != want[i] {
			t.Errorf("marketRow[%d] = %q, want %q", i, row[i], want[i])
		}
	}

	check := func(name string, got, wantIdx []int) {
		t.Helper()
		for i := range wantIdx {
			if got[i] != wantIdx[i] {
				t.Fatalf("%s: got %v want %v", name, got, wantIdx)
			}
		}
	}

	idx := allIndices(3)
	sortMarketIndices(idx, snap, names, 1, true) // food ascending
	check("food asc", idx, []int{1, 0, 2})       // tie 3/3 broken by CityID

	idx = allIndices(3)
	sortMarketIndices(idx, snap, names, 1, false) // food descending
	check("food desc", idx, []int{0, 2, 1})

	idx = allIndices(3)
	sortMarketIndices(idx, snap, names, 0, true) // city name ascending
	check("name asc", idx, []int{1, 0, 2})

	idx = allIndices(3)
	sortMarketIndices(idx, snap, names, 2, true) // wood ascending
	check("wood asc", idx, []int{1, 2, 0})
}

func TestBumpTaxRate(t *testing.T) {
	cases := []struct {
		cur   uint8
		delta int
		want  uint8
	}{
		{10, 5, 15},
		{10, -5, 5},
		{3, -5, 0},
		{0, -5, 0},
		{48, 5, components.MaxTaxRatePercent},
		{components.MaxTaxRatePercent, 5, components.MaxTaxRatePercent},
	}
	for _, c := range cases {
		if got := bumpTaxRate(c.cur, c.delta); got != c.want {
			t.Errorf("bumpTaxRate(%d,%d) = %d, want %d", c.cur, c.delta, got, c.want)
		}
	}
}

func TestRouteLabel(t *testing.T) {
	names := map[uint32]string{1: "Foo", 2: "Bar"}
	r := systems.TradeRouteInfo{FromCity: 1, ToCity: 2, Volume: 10}
	if got := routeLabel(names, r); got != "Foo <-> Bar  x10" {
		t.Errorf("routeLabel = %q", got)
	}
	// Unknown endpoints fall back to the numeric name.
	r = systems.TradeRouteInfo{FromCity: 1, ToCity: 9, Volume: 3}
	if got := routeLabel(names, r); got != "Foo <-> City 9  x3" {
		t.Errorf("fallback routeLabel = %q", got)
	}
}

func TestRosterIndices(t *testing.T) {
	small := []string{"a", "b", "c"}
	// Small rosters show every row: no filter box, the query is ignored.
	if got := rosterIndices(small, "zzz"); len(got) != 3 {
		t.Fatalf("small roster filtered: %v", got)
	}

	big := make([]string, rosterFilterThreshold+2)
	for i := range big {
		big[i] = "row"
	}
	big[3] = "needle"
	got := rosterIndices(big, "need")
	if len(got) != 1 || got[0] != 3 {
		t.Fatalf("big roster filter: %v", got)
	}
	if got := rosterIndices(big, ""); len(got) != len(big) {
		t.Fatalf("empty query must match all: %d", len(got))
	}
}

func TestCityNamesFor(t *testing.T) {
	world := ecs.NewWorld()
	newUIVillage(t, &world, 10, "Foo")
	snap := []systems.CityPrices{{CityID: 10}, {CityID: 99}}
	names := cityNamesFor(&world, snap)
	if names[0] != "Foo" || names[1] != "City 99" {
		t.Fatalf("cityNamesFor = %v", names)
	}
}

func TestFindCityVillageAndCapitalFlag(t *testing.T) {
	world := ecs.NewWorld()
	v := newUIVillage(t, &world, 7, "Foo")
	newUICapital(t, &world, 3, 8, "Cap", 0)

	got, found := findCityVillage(&world, 7)
	if !found || got != v {
		t.Fatalf("findCityVillage(7) = %v %v", got, found)
	}
	if _, found := findCityVillage(&world, 42); found {
		t.Fatal("findCityVillage(42) must miss")
	}

	if cityIsCapital(&world, 7) {
		t.Error("plain village must not be a capital")
	}
	if !cityIsCapital(&world, 8) {
		t.Error("capital settlement must report capital")
	}
	if cityIsCapital(&world, 0) {
		t.Error("cityID 0 is never a capital")
	}
}

func TestChromeActiveTabAndToggle(t *testing.T) {
	resetPanelState()
	pc := &PlayerContext{}
	if got := chromeActiveTab(pc); got != -1 {
		t.Fatalf("no panels open: active = %d, want -1", got)
	}
	pc.MarketOpen = true
	if got := chromeActiveTab(pc); got != chromeTabMarket {
		t.Fatalf("market open: active = %d", got)
	}

	world := ecs.NewWorld()
	s := newUIStatePlaying(&world)

	// Opening a tab closes every other chrome panel.
	s.PC.MarketOpen = true
	s.toggleChromeTab(chromeTabDiplomacy)
	if !s.PC.DiploOpen || s.PC.MarketOpen {
		t.Fatalf("diplo toggle: DiploOpen=%v MarketOpen=%v", s.PC.DiploOpen, s.PC.MarketOpen)
	}
	// Toggling the open tab closes it.
	s.toggleChromeTab(chromeTabDiplomacy)
	if s.PC.DiploOpen {
		t.Fatal("second toggle must close the panel")
	}

	s.toggleChromeTab(chromeTabGoals)
	if !s.PC.AmbitionsOpen {
		t.Fatal("goals tab must open ambitions")
	}
	s.toggleChromeTab(chromeTabCharacter)
	if !s.PC.CharOpen || s.PC.AmbitionsOpen {
		t.Fatal("character tab must open char panel and close goals")
	}
	s.toggleChromeTab(chromeTabChronicle)
	if !s.PC.EventLogOpen || s.PC.CharOpen {
		t.Fatal("chronicle tab must open event log and close character")
	}
}

func TestChromeLawsTabTargetsOwnCity(t *testing.T) {
	resetPanelState()
	world := ecs.NewWorld()
	s := newUIStatePlaying(&world)

	// A possessed citizen of city 7 plus that city's village.
	possID := ecs.ComponentID[components.Possessed](&world)
	affID := ecs.ComponentID[components.Affiliation](&world)
	player := world.NewEntity(possID, affID)
	aff := (*components.Affiliation)(world.Get(player, affID))
	aff.CityID = 7
	v := newUIVillage(t, &world, 7, "Foo")

	s.toggleChromeTab(chromeTabLaws)
	if !s.PC.LawsOpen || s.PC.LawsVillage != v {
		t.Fatalf("laws tab: open=%v village=%v", s.PC.LawsOpen, s.PC.LawsVillage)
	}
	// Toggle off.
	s.toggleChromeTab(chromeTabLaws)
	if s.PC.LawsOpen {
		t.Fatal("laws toggle-off failed")
	}

	// A homeless player cannot open city laws; a note explains why.
	aff = (*components.Affiliation)(world.Get(player, affID))
	aff.CityID = 0
	s.toggleChromeTab(chromeTabLaws)
	if s.PC.LawsOpen {
		t.Fatal("homeless player must not open laws")
	}
	if len(s.PC.Log) == 0 {
		t.Fatal("expected an explanatory note")
	}
}

func TestCloseChromePanels(t *testing.T) {
	pc := &PlayerContext{}
	pc.DiploOpen = true
	pc.MarketOpen = true
	pc.LawsOpen = true
	pc.AmbitionsOpen = true
	pc.CharOpen = true
	pc.EventLogOpen = true
	closeChromePanels(pc)
	if pc.DiploOpen || pc.MarketOpen || pc.LawsOpen || pc.AmbitionsOpen || pc.CharOpen || pc.EventLogOpen {
		t.Fatalf("closeChromePanels left panels open: %+v", pc)
	}
}
