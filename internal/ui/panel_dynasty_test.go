package ui

import (
	"strings"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/mlange-42/arche/ecs"
)

// Logic-only tests for the dynasty panel (panel_dynasty.go): roster building,
// marriage-candidate eligibility, council views, sovereign gating, and the
// shared plot entry point. Everything runs headless on real ecs.Worlds; the
// rendering paths are exercised in the real game (panels_kit_test.go
// convention).

// resetDynastyPanelState zeroes the package-level dynasty UI state so
// -count=2 runs and test order stay independent.
func resetDynastyPanelState() {
	dynastyTable.SortCol, dynastyTable.SortAsc = 0, true
	dynastyFilter = SearchableList{}
	marryPickList = SearchableList{}
	councilPickList = SearchableList{}
	resetDynastyFlows()
}

// spawnUINPC creates an NPC + Identity + Affiliation entity for panel tests.
func spawnUINPC(t *testing.T, world *ecs.World, id uint64, name string,
	familyID, cityID, countryID uint32, age uint16) ecs.Entity {
	t.Helper()
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	e := world.NewEntity(npcID, identID, affID)
	ident := (*components.Identity)(world.Get(e, identID))
	ident.ID = id
	ident.Name = name
	ident.Age = age
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.FamilyID = familyID
	aff.CityID = cityID
	aff.CountryID = countryID
	return e
}

// markUIMarried stamps a Married DynastyComponent on an NPC (test rig).
func markUIMarried(world *ecs.World, e ecs.Entity) {
	dynID := ecs.ComponentID[components.DynastyComponent](world)
	if !world.Has(e, dynID) {
		world.Add(e, dynID)
	}
	(*components.DynastyComponent)(world.Get(e, dynID)).Married = true
}

func TestBuildDynastyRowsAndCells(t *testing.T) {
	resetDynastyPanelState()
	world := ecs.NewWorld()

	a := spawnUINPC(t, &world, 1, "Alden", 7, 5, 0, 40)
	b := spawnUINPC(t, &world, 2, "Bess", 9, 5, 0, 35) // Rooted family 9: stays distinct
	spawnUINPC(t, &world, 3, "Cub", 7, 5, 0, 10)

	jobID := ecs.ComponentID[components.JobComponent](&world)
	world.Add(a, jobID)
	(*components.JobComponent)(world.Get(a, jobID)).JobID = components.JobFarmer

	if err := systems.Marry(&world, nil, a, b); err != nil {
		t.Fatalf("Marry: %v", err)
	}

	rows := buildDynastyRows(&world, systems.ListDynasty(&world, 7), 1)
	if len(rows) != 2 {
		t.Fatalf("family-7 roster has %d rows, want 2 (spouse keeps her house)", len(rows))
	}
	// Eldest first: Alden (40) then Cub (10).
	if rows[0].m.ID != 1 || rows[1].m.ID != 3 {
		t.Fatalf("roster order = [%d %d], want [1 3]", rows[0].m.ID, rows[1].m.ID)
	}
	if !rows[0].player || rows[1].player {
		t.Errorf("player marking wrong: %v %v", rows[0].player, rows[1].player)
	}
	if rows[0].spouse != "Bess" {
		t.Errorf("spouse from another rooted family = %q, want Bess", rows[0].spouse)
	}

	cells := dynastyCells(rows[0])
	want := []string{"> Alden", "40", "Farmer", "yes", "Bess"}
	for i := range want {
		if cells[i] != want[i] {
			t.Errorf("player cells[%d] = %q, want %q", i, cells[i], want[i])
		}
	}
	cells = dynastyCells(rows[1])
	want = []string{"Cub", "10", "Unemployed", "-", "-"}
	for i := range want {
		if cells[i] != want[i] {
			t.Errorf("child cells[%d] = %q, want %q", i, cells[i], want[i])
		}
	}
}

func TestSortDynastyIndices(t *testing.T) {
	rows := []dynastyRow{
		{m: systems.DynastyMember{ID: 1, Age: 40, Married: true}, name: "b", job: "Farmer", spouse: "x"},
		{m: systems.DynastyMember{ID: 2, Age: 40}, name: "a", job: "Guard", spouse: "-"},
		{m: systems.DynastyMember{ID: 3, Age: 12}, name: "c", job: "Unemployed", spouse: "-"},
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
	sortDynastyIndices(idx, rows, 1, true) // age ascending
	check("age asc", idx, []int{2, 0, 1})  // tie 40/40 broken by ID 1 < 2

	idx = allIndices(3)
	sortDynastyIndices(idx, rows, 1, false) // age descending
	check("age desc", idx, []int{0, 1, 2})  // tie-break stays ascending by ID

	idx = allIndices(3)
	sortDynastyIndices(idx, rows, 0, true) // name ascending
	check("name asc", idx, []int{1, 0, 2})

	idx = allIndices(3)
	sortDynastyIndices(idx, rows, 3, true)    // married: unmarried first
	check("married asc", idx, []int{1, 2, 0}) // tie broken by ID 2 < 3
}

func TestListMarriageCandidates(t *testing.T) {
	resetDynastyPanelState()
	world := ecs.NewWorld()

	player := spawnUINPC(t, &world, 1, "You", 7, 5, 0, 30)
	spawnUINPC(t, &world, 4, "Match", 9, 5, 0, 25)     // Eligible
	spawnUINPC(t, &world, 5, "Kin", 7, 5, 0, 30)       // Same family: out
	spawnUINPC(t, &world, 6, "Child", 9, 5, 0, 10)     // Underage: out
	wed := spawnUINPC(t, &world, 7, "Wed", 9, 5, 0, 30) // Married: out
	markUIMarried(&world, wed)
	spawnUINPC(t, &world, 8, "Far", 9, 6, 0, 30)      // Other city: out
	spawnUINPC(t, &world, 9, "Rootless", 0, 5, 0, 20) // Rootless: eligible

	cands := listMarriageCandidates(&world, player)
	if len(cands) != 2 || cands[0].ID != 4 || cands[1].ID != 9 {
		t.Fatalf("candidates = %+v, want IDs [4 9]", cands)
	}
	labels := npcOptionLabels(cands)
	if labels[0] != "Match (25)" || labels[1] != "Rootless (20)" {
		t.Errorf("labels = %v", labels)
	}

	// A homeless player has nobody to court.
	affID := ecs.ComponentID[components.Affiliation](&world)
	(*components.Affiliation)(world.Get(player, affID)).CityID = 0
	if got := listMarriageCandidates(&world, player); got != nil {
		t.Errorf("homeless candidates = %v, want nil", got)
	}
}

func TestCanProposeMarriage(t *testing.T) {
	world := ecs.NewWorld()
	player := spawnUINPC(t, &world, 1, "You", 7, 5, 0, 30)
	match := spawnUINPC(t, &world, 4, "Match", 9, 5, 0, 25)
	kin := spawnUINPC(t, &world, 5, "Kin", 7, 5, 0, 30)
	child := spawnUINPC(t, &world, 6, "Child", 9, 5, 0, 10)
	wed := spawnUINPC(t, &world, 7, "Wed", 9, 5, 0, 30)
	markUIMarried(&world, wed)

	if !canProposeMarriage(&world, player, match) {
		t.Error("eligible match rejected")
	}
	if canProposeMarriage(&world, player, kin) {
		t.Error("own-family target accepted")
	}
	if canProposeMarriage(&world, player, child) {
		t.Error("underage target accepted")
	}
	if canProposeMarriage(&world, player, wed) {
		t.Error("married target accepted")
	}
	if canProposeMarriage(&world, player, player) {
		t.Error("self-proposal accepted")
	}
	markUIMarried(&world, player)
	if canProposeMarriage(&world, player, match) {
		t.Error("married player may not propose")
	}
}

func TestCouncilSeatViewsAndCandidates(t *testing.T) {
	resetDynastyPanelState()
	world := ecs.NewWorld()
	newUICapital(t, &world, 3, 8, "Cap", 100)
	spawnUINPC(t, &world, 21, "Sten", 0, 8, 3, 30) // Capital adult: candidate
	spawnUINPC(t, &world, 22, "Tot", 0, 8, 3, 9)   // Capital child: out
	spawnUINPC(t, &world, 23, "Away", 0, 9, 3, 30) // Other city: out

	views := councilSeatViews(&world, 3)
	if len(views) != 4 {
		t.Fatalf("seat views = %d, want 4", len(views))
	}
	wantTitles := []string{"Steward", "Marshal", "Diplomat", "Spymaster"}
	for i, v := range views {
		if v.Title != wantTitles[i] || v.Holder != "vacant" || v.HolderID != 0 {
			t.Errorf("seat %d = %+v, want vacant %s", i, v, wantTitles[i])
		}
		if seatTip(v.Seat) == "" {
			t.Errorf("seat %d has no tooltip", i)
		}
	}

	if err := systems.Appoint(&world, 3, components.SeatSteward, 21); err != nil {
		t.Fatalf("Appoint: %v", err)
	}
	views = councilSeatViews(&world, 3)
	if views[0].Holder != "Sten" || views[0].HolderID != 21 {
		t.Errorf("steward view = %+v, want Sten/21", views[0])
	}

	cands := listCouncilCandidates(&world, 3)
	if len(cands) != 1 || cands[0].ID != 21 {
		t.Fatalf("council candidates = %+v, want ID [21]", cands)
	}
	// A country without a capital has neither seats' holders nor candidates.
	if got := listCouncilCandidates(&world, 99); got != nil {
		t.Errorf("no-capital candidates = %v, want nil", got)
	}
	for _, v := range councilSeatViews(&world, 99) {
		if v.Holder != "vacant" {
			t.Errorf("no-capital seat %+v, want vacant", v)
		}
	}
}

func TestAppointCouncilorSovereignGating(t *testing.T) {
	resetDynastyPanelState()
	world := ecs.NewWorld()
	s := newUIStatePlaying(&world)
	newUICapital(t, &world, 3, 8, "Cap", 100)
	player := spawnUINPC(t, &world, 1, "You", 0, 8, 3, 30)
	world.Add(player, ecs.ComponentID[components.Possessed](&world))
	spawnUINPC(t, &world, 21, "Sten", 0, 8, 3, 30)

	// A plain citizen is refused.
	s.appointCouncilor(3, components.SeatSteward, 21, "Sten")
	if got := councilSeatViews(&world, 3)[0].HolderID; got != 0 {
		t.Fatalf("citizen appointment seated %d, want vacant", got)
	}
	if len(s.PC.Log) == 0 || !strings.Contains(s.PC.Log[len(s.PC.Log)-1].Text, "Sovereign") {
		t.Fatalf("expected a sovereign-gate note, log = %+v", s.PC.Log)
	}

	// Crowned in the capital city, the player is Sovereign and may appoint.
	world.Add(player, ecs.ComponentID[components.AdministrationMarker](&world))
	s.appointCouncilor(3, components.SeatSteward, 21, "Sten")
	if got := councilSeatViews(&world, 3)[0].HolderID; got != 21 {
		t.Fatalf("sovereign appointment seated %d, want 21", got)
	}
}

func TestStartPlotAgainstAndAbandon(t *testing.T) {
	resetDynastyPanelState()
	world := ecs.NewWorld()
	s := newUIStatePlaying(&world)
	player := spawnUINPC(t, &world, 1, "You", 0, 5, 0, 30)
	world.Add(player, ecs.ComponentID[components.Possessed](&world))
	target := spawnUINPC(t, &world, 40, "Rival", 0, 5, 0, 30)
	other := spawnUINPC(t, &world, 41, "Other", 0, 5, 0, 30)

	StartPlotAgainst(s, target, components.PlotAssassinate)
	plotID := ecs.ComponentID[components.PlotComponent](&world)
	if !world.Has(player, plotID) {
		t.Fatalf("StartPlotAgainst must attach a PlotComponent")
	}
	plot := (*components.PlotComponent)(world.Get(player, plotID))
	if plot.TargetID != 40 || plot.Kind != components.PlotAssassinate {
		t.Errorf("plot = %+v, want TargetID 40 kind PlotAssassinate", *plot)
	}
	if len(s.PC.Log) == 0 || !strings.Contains(s.PC.Log[len(s.PC.Log)-1].Text, "assassinate") {
		t.Errorf("expected an assassination note, log = %+v", s.PC.Log)
	}

	// One plot per plotter: a second attempt is refused with a note.
	StartPlotAgainst(s, other, components.PlotSeizeRule)
	plot = (*components.PlotComponent)(world.Get(player, plotID))
	if plot.TargetID != 40 {
		t.Errorf("second plot overwrote the first: %+v", *plot)
	}
	if !strings.Contains(s.PC.Log[len(s.PC.Log)-1].Text, "stillborn") {
		t.Errorf("expected a rejection note, log = %+v", s.PC.Log)
	}

	s.abandonPlot(player)
	if world.Has(player, plotID) {
		t.Fatalf("abandonPlot must remove the PlotComponent")
	}

	if got := plotKindName(components.PlotSeizeRule); got != "Seize rule" {
		t.Errorf("plotKindName(seize) = %q", got)
	}
	if got := plotKindName(components.PlotAssassinate); got != "Assassinate" {
		t.Errorf("plotKindName(assassinate) = %q", got)
	}
}

func TestProposeMarriageTo(t *testing.T) {
	resetDynastyPanelState()
	world := ecs.NewWorld()
	s := newUIStatePlaying(&world)
	player := spawnUINPC(t, &world, 1, "You", 7, 5, 0, 30)
	world.Add(player, ecs.ComponentID[components.Possessed](&world))
	partner := spawnUINPC(t, &world, 4, "Match", 9, 5, 0, 25)
	child := spawnUINPC(t, &world, 6, "Child", 9, 5, 0, 10)

	// Marrying a child fails through systems.Marry with an explanatory note.
	s.proposeMarriageTo(child)
	dynID := ecs.ComponentID[components.DynastyComponent](&world)
	if world.Has(player, dynID) && (*components.DynastyComponent)(world.Get(player, dynID)).Married {
		t.Fatalf("underage proposal must not wed the player")
	}
	if len(s.PC.Log) == 0 || !strings.Contains(s.PC.Log[len(s.PC.Log)-1].Text, "proposal fails") {
		t.Fatalf("expected a failure note, log = %+v", s.PC.Log)
	}

	// The real match weds both ways.
	s.proposeMarriageTo(partner)
	pd := (*components.DynastyComponent)(world.Get(player, dynID))
	if !pd.Married || pd.SpouseID != 4 {
		t.Errorf("player dynasty = %+v, want Married with SpouseID 4", *pd)
	}
	qd := (*components.DynastyComponent)(world.Get(partner, dynID))
	if !qd.Married || qd.SpouseID != 1 {
		t.Errorf("partner dynasty = %+v, want Married with SpouseID 1", *qd)
	}
	if !strings.Contains(s.PC.Log[len(s.PC.Log)-1].Text, "wed to Match") {
		t.Errorf("expected a wedding note, log = %+v", s.PC.Log)
	}
}
