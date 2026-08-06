package ui

import (
	"fmt"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.3/P2.4/P2.5 UI + U7 + U8): the dynasty panel.
// A modal with the living family roster (name, age, job, marital links; the
// player's row marked), the marriage flow (pick an eligible partner from the
// player's city -> systems.Marry), the player's active plot with an abandon
// action (plots start from the NPC context menu via StartPlotAgainst), and
// the sovereign's council with per-seat appointment (systems.Appoint).
// Non-sovereigns read the council; only a Sovereign appoints. All input goes
// through the BeginUIFrame snapshot / widget kit — no inpututil in Draw.

const (
	dynastyW      = 640
	dynastyH      = 440
	dynastyTableW = 330 // left column: the family roster
)

// Persistent panel state (UI-only; never read by the simulation).
var (
	dynastyTable = Table{
		Columns: []string{"Name", "Age", "Job", "Wed", "Spouse"},
		Widths:  []int{110, 34, 70, 34, 80},
		SortAsc: true,
	}
	dynastyFilter   SearchableList
	marryPickList   SearchableList
	marryPicking    bool
	councilPickList SearchableList
	councilPickSeat uint8 // components.Seat* being appointed; 0 = none
)

// resetDynastyFlows cancels any armed pick flow (panel close, Esc).
func resetDynastyFlows() {
	marryPicking = false
	councilPickSeat = 0
}

func cmpUint64(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// dynastyRow is one prepared roster-table row.
type dynastyRow struct {
	m      systems.DynastyMember
	name   string // display name ("Citizen N" fallback)
	job    string
	spouse string // spouse display name, "-" when unmarried
	player bool
}

// npcDisplayName renders an Identity name with the numeric fallback.
func npcDisplayName(name string, id uint64) string {
	if name == "" {
		return fmt.Sprintf("Citizen %d", id)
	}
	return name
}

// buildDynastyRows resolves display strings for a ListDynasty snapshot.
// Spouses married in from another rooted family are resolved world-wide.
func buildDynastyRows(world *ecs.World, members []systems.DynastyMember, playerID uint64) []dynastyRow {
	rows := make([]dynastyRow, len(members))
	for i, m := range members {
		r := dynastyRow{
			m:      m,
			name:   npcDisplayName(m.Name, m.ID),
			job:    JobName(m.JobID),
			spouse: "-",
			player: playerID != 0 && m.ID == playerID,
		}
		if m.Married && m.SpouseID != 0 {
			r.spouse = npcNameByID(world, m.SpouseID)
		}
		rows[i] = r
	}
	return rows
}

// dynastyCells renders one roster row; the player's row carries a "> " marker.
func dynastyCells(r dynastyRow) []string {
	name := r.name
	if r.player {
		name = "> " + name
	}
	wed := "-"
	if r.m.Married {
		wed = "yes"
	}
	return []string{name, fmt.Sprintf("%d", r.m.Age), r.job, wed, r.spouse}
}

// boolRank orders false before true for sortable boolean columns.
func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}

// sortDynastyIndices orders the visible row indices by the table's sort
// column, deterministically tie-broken by Identity.ID ascending.
func sortDynastyIndices(idx []int, rows []dynastyRow, col int, asc bool) {
	slices.SortStableFunc(idx, func(a, b int) int {
		ra, rb := &rows[a], &rows[b]
		c := 0
		switch col {
		case 0:
			c = cmpString(ra.name, rb.name)
		case 1:
			c = cmpInt(int(ra.m.Age), int(rb.m.Age))
		case 2:
			c = cmpString(ra.job, rb.job)
		case 3:
			c = cmpInt(boolRank(ra.m.Married), boolRank(rb.m.Married))
		case 4:
			c = cmpString(ra.spouse, rb.spouse)
		}
		if c == 0 {
			return cmpUint64(ra.m.ID, rb.m.ID)
		}
		if !asc {
			c = -c
		}
		return c
	})
}

// npcOption is one pickable NPC in a SearchableList flow.
type npcOption struct {
	Entity ecs.Entity
	ID     uint64
	Name   string
	Age    uint16
}

// sortNPCOptions orders options by Identity.ID ascending (deterministic UI).
func sortNPCOptions(out []npcOption) {
	slices.SortFunc(out, func(a, b npcOption) int { return cmpUint64(a.ID, b.ID) })
}

// npcOptionLabels renders "Name (age)" list labels aligned with options.
func npcOptionLabels(opts []npcOption) []string {
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = fmt.Sprintf("%s (%d)", o.Name, o.Age)
	}
	return labels
}

// listMarriageCandidates scans the player's city for eligible partners:
// living adult NPCs, unmarried, outside the player's family, never the player
// (systems.Marry re-validates; this is the UI pre-filter). Sorted by
// Identity.ID. Full query iteration keeps the world unlocked.
func listMarriageCandidates(world *ecs.World, player ecs.Entity) []npcOption {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	dynID := ecs.ComponentID[components.DynastyComponent](world)

	if !world.Alive(player) || !world.Has(player, affID) || !world.Has(player, identID) {
		return nil
	}
	pAff := (*components.Affiliation)(world.Get(player, affID))
	playerCity, playerFamily := pAff.CityID, pAff.FamilyID
	playerID := (*components.Identity)(world.Get(player, identID)).ID
	if playerCity == 0 {
		return nil // The wilds hold no wedding guests.
	}

	var out []npcOption
	q := world.Query(ecs.All(npcID, identID, affID))
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(affID))
		if aff.CityID != playerCity {
			continue
		}
		if playerFamily != 0 && aff.FamilyID == playerFamily {
			continue // No marrying your own house.
		}
		ident := (*components.Identity)(q.Get(identID))
		if ident.ID == playerID || ident.Age < systems.MarriageAdultAge {
			continue
		}
		if q.Has(dynID) && (*components.DynastyComponent)(q.Get(dynID)).Married {
			continue
		}
		out = append(out, npcOption{
			Entity: q.Entity(), ID: ident.ID,
			Name: npcDisplayName(ident.Name, ident.ID), Age: ident.Age,
		})
	}
	sortNPCOptions(out)
	return out
}

// canProposeMarriage reports whether the possessed player may propose to the
// target NPC: both alive adult unmarried NPCs, distinct, and the target is
// not of the player's (rooted) family. Context-menu gating helper.
func canProposeMarriage(world *ecs.World, player, target ecs.Entity) bool {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	dynID := ecs.ComponentID[components.DynastyComponent](world)

	if player == target || !world.Alive(player) || !world.Alive(target) {
		return false
	}
	for _, e := range [2]ecs.Entity{player, target} {
		if !world.Has(e, identID) {
			return false
		}
		if (*components.Identity)(world.Get(e, identID)).Age < systems.MarriageAdultAge {
			return false
		}
		if world.Has(e, dynID) && (*components.DynastyComponent)(world.Get(e, dynID)).Married {
			return false
		}
	}
	if !world.Has(target, npcID) {
		return false
	}
	if world.Has(player, affID) && world.Has(target, affID) {
		pf := (*components.Affiliation)(world.Get(player, affID)).FamilyID
		tf := (*components.Affiliation)(world.Get(target, affID)).FamilyID
		if pf != 0 && pf == tf {
			return false
		}
	}
	return true
}

// proposeMarriageTo weds the possessed player to the chosen partner through
// systems.Marry and reports the outcome (or the error) as a notification.
func (s *StatePlaying) proposeMarriageTo(partner ecs.Entity) {
	pc := s.PC
	world := s.Status.TM.World
	tick := s.Status.TM.Ticks
	player, ok := pc.PossessedEntity()
	if !ok {
		return
	}
	name := entityDisplayName(world, partner)
	if err := systems.Marry(world, s.Status.HookGraph, player, partner); err != nil {
		pc.PushNote("The proposal fails: "+err.Error(), tick)
		return
	}
	pc.PushNote("You are wed to "+name+"!", tick)
}

// entityDisplayName resolves a living entity's Identity display name.
func entityDisplayName(world *ecs.World, e ecs.Entity) string {
	identID := ecs.ComponentID[components.Identity](world)
	if !world.Alive(e) || !world.Has(e, identID) {
		return "someone"
	}
	ident := (*components.Identity)(world.Get(e, identID))
	return npcDisplayName(ident.Name, ident.ID)
}

// StartPlotAgainst starts a plot by the possessed player against the target
// via systems.StartPlot and reports the outcome (or the error) as a
// notification. Shared entry point for the dynasty panel and the context-menu
// "Plot Against" verbs — the menu call site is one line.
func StartPlotAgainst(s *StatePlaying, target ecs.Entity, kind uint8) {
	pc := s.PC
	world := s.Status.TM.World
	tick := s.Status.TM.Ticks
	player, ok := pc.PossessedEntity()
	if !ok {
		return
	}
	name := entityDisplayName(world, target)
	if err := systems.StartPlot(world, player, target, kind); err != nil {
		pc.PushNote("The plot dies stillborn: "+err.Error(), tick)
		return
	}
	verb := "seize power from"
	if kind == components.PlotAssassinate {
		verb = "assassinate"
	}
	pc.PushNote("You begin plotting to "+verb+" "+name+".", tick)
}

// abandonPlot removes the player's active plot. Structural removal strictly
// outside any query loop (UI runs between ticks; no queries are open here).
func (s *StatePlaying) abandonPlot(player ecs.Entity) {
	world := s.Status.TM.World
	plotID := ecs.ComponentID[components.PlotComponent](world)
	if world.Alive(player) && world.Has(player, plotID) {
		world.Remove(player, plotID)
		s.PC.PushNote("You quietly bury the plot.", s.Status.TM.Ticks)
	}
}

// plotKindName names a PlotComponent.Kind for display.
func plotKindName(kind uint8) string {
	switch kind {
	case components.PlotSeizeRule:
		return "Seize rule"
	case components.PlotAssassinate:
		return "Assassinate"
	}
	return "Scheme"
}

// seatView is one council seat prepared for display.
type seatView struct {
	Seat     uint8
	Title    string
	Holder   string // holder display name, "vacant" when empty
	HolderID uint64
}

// seatTitle names a council seat constant.
func seatTitle(seat uint8) string {
	switch seat {
	case components.SeatSteward:
		return "Steward"
	case components.SeatMarshal:
		return "Marshal"
	case components.SeatDiplomat:
		return "Diplomat"
	case components.SeatSpymaster:
		return "Spymaster"
	}
	return "?"
}

// seatTip explains a seat's national bonus (spec U8).
func seatTip(seat uint8) string {
	switch seat {
	case components.SeatSteward:
		return fmt.Sprintf("The Steward grows the national treasury by %d%% each council cycle.",
			systems.StewardIncomePct)
	case components.SeatMarshal:
		return fmt.Sprintf("The Marshal adds %d war score to your realm's wars.",
			systems.MarshalWarScoreBonus)
	case components.SeatDiplomat:
		return fmt.Sprintf("The Diplomat passively lifts foreign opinion by %d per diplomacy cycle.",
			systems.DiplomatOpinionDrift)
	case components.SeatSpymaster:
		return fmt.Sprintf("The Spymaster adds %d%% to plot-discovery rolls protecting your realm.",
			systems.SpymasterDiscoveryBonus)
	}
	return ""
}

// councilSeatViews resolves the country's four council seats with holder
// names. A missing capital or council yields all-vacant seats.
func councilSeatViews(world *ecs.World, countryID uint32) []seatView {
	var council components.CouncilComponent
	if capital, ok := systems.FindCapitalOf(world, countryID); ok {
		councilID := ecs.ComponentID[components.CouncilComponent](world)
		if world.Has(capital, councilID) {
			council = *(*components.CouncilComponent)(world.Get(capital, councilID))
		}
	}
	holders := [...]uint64{council.Steward, council.Marshal, council.Diplomat, council.Spymaster}
	out := make([]seatView, 0, len(holders))
	for i, holder := range holders {
		seat := components.SeatSteward + uint8(i)
		v := seatView{Seat: seat, Title: seatTitle(seat), Holder: "vacant", HolderID: holder}
		if holder != 0 {
			v.Holder = npcNameByID(world, holder)
		}
		out = append(out, v)
	}
	return out
}

// listCouncilCandidates lists the capital city's living adult citizens — the
// pool a sovereign appoints from (adulthood floor shared with marriage).
// Sorted by Identity.ID. Full query iteration keeps the world unlocked.
func listCouncilCandidates(world *ecs.World, countryID uint32) []npcOption {
	capital, ok := systems.FindCapitalOf(world, countryID)
	if !ok {
		return nil
	}
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	if !world.Has(capital, affID) {
		return nil
	}
	capCity := (*components.Affiliation)(world.Get(capital, affID)).CityID
	if capCity == 0 {
		return nil
	}

	var out []npcOption
	q := world.Query(ecs.All(npcID, identID, affID))
	for q.Next() {
		aff := (*components.Affiliation)(q.Get(affID))
		if aff.CityID != capCity {
			continue
		}
		ident := (*components.Identity)(q.Get(identID))
		if ident.Age < systems.MarriageAdultAge {
			continue
		}
		out = append(out, npcOption{
			Entity: q.Entity(), ID: ident.ID,
			Name: npcDisplayName(ident.Name, ident.ID), Age: ident.Age,
		})
	}
	sortNPCOptions(out)
	return out
}

// appointCouncilor seats the NPC on the given council seat of the player's
// country, re-checking the sovereign gate, and reports the outcome (or the
// error) as a notification.
func (s *StatePlaying) appointCouncilor(countryID uint32, seat uint8, npcID uint64, name string) {
	pc := s.PC
	world := s.Status.TM.World
	tick := s.Status.TM.Ticks
	player, ok := pc.PossessedEntity()
	if !ok {
		return
	}
	if rank, _, _ := systems.GetRank(world, player); rank != systems.RankSovereign {
		pc.PushNote("Only a Sovereign seats the council.", tick)
		return
	}
	if err := systems.Appoint(world, countryID, seat, npcID); err != nil {
		pc.PushNote("Appointment refused: "+err.Error(), tick)
		return
	}
	pc.PushNote(name+" now serves as "+seatTitle(seat)+".", tick)
}

// DrawDynasty renders the dynasty modal. No-op while pc.DynastyOpen is false.
func (s *StatePlaying) DrawDynasty(screen *ebiten.Image) {
	pc := s.PC
	if !pc.DynastyOpen {
		return
	}
	world := s.Status.TM.World

	cx, cy, closed := ModalFrame(screen, "DYNASTY & INTRIGUE", dynastyW, dynastyH)
	if closed {
		pc.DynastyOpen = false
		resetDynastyFlows()
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	_, my0 := modalOrigin(sw, sh, dynastyW, dynastyH)
	bottom := my0 + dynastyH - PadM
	cw := dynastyW - 2*PadM

	player, ok := pc.PossessedEntity()
	if !ok {
		DrawText(screen, "No body, no bloodline.", cx, cy, TextDim)
		return
	}

	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	dynID := ecs.ComponentID[components.DynastyComponent](world)

	var playerID uint64
	var playerAge uint16
	if world.Has(player, identID) {
		ident := (*components.Identity)(world.Get(player, identID))
		playerID, playerAge = ident.ID, ident.Age
	}
	var familyID uint32
	if world.Has(player, affID) {
		familyID = (*components.Affiliation)(world.Get(player, affID)).FamilyID
	}
	playerMarried := false
	var spouseID uint64
	if world.Has(player, dynID) {
		d := (*components.DynastyComponent)(world.Get(player, dynID))
		playerMarried, spouseID = d.Married, d.SpouseID
	}

	// --- Left column: the family roster. ---
	s.drawDynastyRoster(screen, cx, cy, bottom, familyID, playerID)

	// --- Right column: marriage, plots, council. ---
	rx := cx + dynastyTableW + PadL
	rw := cw - dynastyTableW - PadL

	// An armed pick flow owns the whole right column.
	if marryPicking {
		s.drawMarryPick(screen, rx, cy, rw, bottom, player)
		return
	}
	if councilPickSeat != 0 {
		s.drawCouncilPick(screen, rx, cy, rw, bottom, player)
		return
	}

	ry := cy
	DrawText(screen, "MARRIAGE", rx, ry, AccentCol)
	HoverTip(screen, rx, ry, MeasureText("MARRIAGE"), 16,
		"Marriage links two unmarried adults: spouse ties both ways, a mutual\naffection bond, and a rootless spouse joins the rooted family.\nChildren carry your line — and your heirs.")
	ry += 18
	switch {
	case playerMarried:
		DrawText(screen, "Wed to "+npcNameByID(world, spouseID)+".", rx, ry, TextCol)
		ry += 24
	case playerAge < systems.MarriageAdultAge:
		DrawText(screen, "You are too young to wed.", rx, ry, TextDim)
		ry += 24
	default:
		if Button(screen, "Propose marriage", rx, ry, 140, 20) {
			resetDynastyFlows()
			marryPicking = true
			marryPickList.Query = ""
			marryPickList.Focus()
		}
		ry += 26
	}

	DrawText(screen, "PLOTS", rx, ry, AccentCol)
	HoverTip(screen, rx, ry, MeasureText("PLOTS"), 16,
		"A plot gains Progress from your intellect and conspirators each\ncycle; at 100 it strikes. Every cycle risks exposure — a Spymaster\non the target's council sees further. Start one from an NPC's\nright-click menu (Plot Against).")
	ry += 18
	plotID := ecs.ComponentID[components.PlotComponent](world)
	if world.Has(player, plotID) {
		plot := (*components.PlotComponent)(world.Get(player, plotID))
		line := fmt.Sprintf("%s: %s  %d/100", plotKindName(plot.Kind),
			npcNameByID(world, plot.TargetID), plot.Progress)
		if plot.Exposed {
			line += "  EXPOSED!"
		}
		DrawText(screen, line, rx, ry, TextCol)
		ry += 18
		if Button(screen, "Abandon", rx, ry, 76, 18) {
			s.abandonPlot(player)
		}
		ry += 26
	} else {
		DrawText(screen, "No plot afoot.", rx, ry, TextDim)
		ry += 24
	}

	rank, _, countryID := systems.GetRank(world, player)
	DrawText(screen, "COUNCIL", rx, ry, AccentCol)
	HoverTip(screen, rx, ry, MeasureText("COUNCIL"), 16,
		"The Sovereign seats four councilors from the capital's citizens;\neach filled seat grants a periodic national bonus. Seats vacate\nwhen their holder dies or emigrates.")
	ry += 18
	if countryID == 0 {
		DrawText(screen, "You serve no realm.", rx, ry, TextDim)
		return
	}
	isSov := rank == systems.RankSovereign
	for _, sv := range councilSeatViews(world, countryID) {
		line := sv.Title + ": " + sv.Holder
		DrawText(screen, line, rx, ry, TextCol)
		HoverTip(screen, rx, ry, MeasureText(line), 16, seatTip(sv.Seat))
		if isSov {
			if Button(screen, "Appoint...", rx+rw-78, ry-2, 74, 18) {
				resetDynastyFlows()
				councilPickSeat = sv.Seat
				councilPickList.Query = ""
				councilPickList.Focus()
			}
		}
		ry += 20
	}
	if !isSov {
		DrawSmall(screen, "Only a Sovereign appoints the council.", rx, ry+2, TextDim)
	}
}

// drawDynastyRoster renders the family-roster table in the left column
// (U7 filter box on big rosters).
func (s *StatePlaying) drawDynastyRoster(screen *ebiten.Image, x, y, bottom int, familyID uint32, playerID uint64) {
	world := s.Status.TM.World
	DrawText(screen, "YOUR HOUSE", x, y, AccentCol)
	HoverTip(screen, x, y, MeasureText("YOUR HOUSE"), 16,
		"Every living member of your family, eldest first — the heir line.\nThe > marks you.")
	y += 18

	members := systems.ListDynasty(world, familyID)
	if len(members) == 0 {
		DrawText(screen, "You have no living kin.", x, y, TextDim)
		return
	}
	rows := buildDynastyRows(world, members, playerID)
	names := make([]string, len(rows))
	for i := range rows {
		names[i] = rows[i].name
	}
	if len(rows) > rosterFilterThreshold {
		dynastyFilter.Draw(screen, x, y, dynastyTableW, SearchBoxH, names)
		y += SearchBoxH + PadS
	}
	visible := rosterIndices(names, dynastyFilter.Query)
	sortDynastyIndices(visible, rows, dynastyTable.SortCol, dynastyTable.SortAsc)
	cells := make([][]string, len(visible))
	for i, vi := range visible {
		cells[i] = dynastyCells(rows[vi])
	}
	dynastyTable.Draw(screen, x, y, dynastyTableW, bottom-y, cells)
	tableHeaderTips(screen, &dynastyTable, x, y, dynastyTableW, []string{
		"A living member of your family; > marks you.",
		"Years lived; kin are listed eldest-first.",
		"Their trade.",
		"Whether they are married.",
		"Who they are wed to.",
	})
}

// drawMarryPick renders the marriage-candidate pick flow in the right column.
func (s *StatePlaying) drawMarryPick(screen *ebiten.Image, rx, ry, rw, bottom int, player ecs.Entity) {
	world := s.Status.TM.World
	DrawText(screen, "Choose your betrothed:", rx, ry, TextCol)
	if Button(screen, "Cancel", rx+rw-64, ry-2, 60, 18) {
		resetDynastyFlows()
		return
	}
	ry += 20
	cands := listMarriageCandidates(world, player)
	if len(cands) == 0 {
		DrawText(screen, "No eligible match lives in your city.", rx, ry, TextDim)
		return
	}
	picked := marryPickList.Draw(screen, rx, ry, rw, bottom-ry, npcOptionLabels(cands))
	if picked < 0 {
		picked = marryPickList.EnterIndex
	}
	if picked >= 0 && picked < len(cands) {
		resetDynastyFlows()
		s.proposeMarriageTo(cands[picked].Entity)
	}
}

// drawCouncilPick renders the councilor pick flow in the right column.
func (s *StatePlaying) drawCouncilPick(screen *ebiten.Image, rx, ry, rw, bottom int, player ecs.Entity) {
	world := s.Status.TM.World
	seat := councilPickSeat
	_, _, countryID := systems.GetRank(world, player)
	DrawText(screen, "Choose the new "+seatTitle(seat)+":", rx, ry, TextCol)
	if Button(screen, "Cancel", rx+rw-64, ry-2, 60, 18) {
		resetDynastyFlows()
		return
	}
	ry += 20
	cands := listCouncilCandidates(world, countryID)
	if len(cands) == 0 {
		DrawText(screen, "No adult citizen lives in your capital.", rx, ry, TextDim)
		return
	}
	picked := councilPickList.Draw(screen, rx, ry, rw, bottom-ry, npcOptionLabels(cands))
	if picked < 0 {
		picked = councilPickList.EnterIndex
	}
	if picked >= 0 && picked < len(cands) {
		resetDynastyFlows()
		s.appointCouncilor(countryID, seat, cands[picked].ID, cands[picked].Name)
	}
}
