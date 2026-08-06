package ui

import (
	"fmt"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.2 UI + U7 + U8): the diplomacy panel. A modal
// table of every known foreign realm (opinion, war/alliance/truce status,
// war score) with the sovereign action row: improve relations, ally, break
// alliance, declare war, sue for peace. Non-sovereigns read the ledger but
// cannot act. All input via the BeginUIFrame snapshot / widget kit.

const (
	diploW = 640
	diploH = 420
)

// Persistent panel state (UI-only; never read by the simulation).
var (
	diploTable = Table{
		Columns: []string{"Realm", "Capital", "Opinion", "Status", "WarScore"},
		Widths:  []int{110, 150, 70, 90, 80},
		SortAsc: true,
	}
	diploFilter   SearchableList
	diploSelected uint32 // TargetCountry of the selected row; 0 = none
)

// realmName renders the display name of a country ID.
func realmName(countryID uint32) string {
	return fmt.Sprintf("Realm %d", countryID)
}

// countryCapitalCityName resolves the name of a country's capital settlement,
// "-" when the country has no living capital.
func countryCapitalCityName(world *ecs.World, countryID uint32) string {
	capital, ok := systems.FindCapitalOf(world, countryID)
	if !ok {
		return "-"
	}
	identID := ecs.ComponentID[components.Identity](world)
	if !world.Has(capital, identID) {
		return "-"
	}
	ident := (*components.Identity)(world.Get(capital, identID))
	if ident.Name == "" {
		return fmt.Sprintf("City %d", uint32(ident.ID))
	}
	return ident.Name
}

// diploRow is one prepared diplomacy-table row.
type diploRow struct {
	rel     components.CountryRelation
	realm   string
	capital string
}

// buildDiploRows resolves display names for a relations snapshot. The input
// comes from systems.GetRelations, which is already sorted by TargetCountry.
func buildDiploRows(world *ecs.World, rels []components.CountryRelation) []diploRow {
	rows := make([]diploRow, len(rels))
	for i, r := range rels {
		rows[i] = diploRow{
			rel:     r,
			realm:   realmName(r.TargetCountry),
			capital: countryCapitalCityName(world, r.TargetCountry),
		}
	}
	return rows
}

// diploStatus renders a relation's state as text (icons-as-text).
func diploStatus(r components.CountryRelation, tick uint64) string {
	switch {
	case r.AtWar:
		return "! AT WAR"
	case r.Alliance:
		return "+ Allied"
	case tick < r.TruceUntil:
		return "~ Truce"
	default:
		return "  Peace"
	}
}

// diploStatusRank orders statuses for the sortable Status column:
// wars first, then truces, peace, alliances.
func diploStatusRank(r components.CountryRelation, tick uint64) int {
	switch {
	case r.AtWar:
		return 0
	case tick < r.TruceUntil:
		return 1
	case r.Alliance:
		return 3
	default:
		return 2
	}
}

// diploCells renders one table row. The selected realm gets a "> " marker.
func diploCells(row diploRow, tick uint64, selected uint32) []string {
	realm := row.realm
	if selected != 0 && row.rel.TargetCountry == selected {
		realm = "> " + realm
	}
	war := "-"
	if row.rel.AtWar {
		war = fmt.Sprintf("%+d", row.rel.WarScore)
	}
	return []string{
		realm,
		row.capital,
		fmt.Sprintf("%+d", row.rel.Opinion),
		diploStatus(row.rel, tick),
		war,
	}
}

// sortDiploIndices orders the visible row indices by the table's sort column,
// deterministically tie-broken by TargetCountry ascending.
func sortDiploIndices(idx []int, rows []diploRow, tick uint64, col int, asc bool) {
	slices.SortStableFunc(idx, func(a, b int) int {
		ra, rb := rows[a].rel, rows[b].rel
		c := 0
		switch col {
		case 0: // Realm: numeric country ID (names are "Realm N")
			c = cmpUint32(ra.TargetCountry, rb.TargetCountry)
		case 1:
			c = cmpString(rows[a].capital, rows[b].capital)
		case 2:
			c = cmpInt(int(ra.Opinion), int(rb.Opinion))
		case 3:
			c = cmpInt(diploStatusRank(ra, tick), diploStatusRank(rb, tick))
		case 4:
			c = cmpInt(int(ra.WarScore), int(rb.WarScore))
		}
		if c == 0 {
			return cmpUint32(ra.TargetCountry, rb.TargetCountry)
		}
		if !asc {
			c = -c
		}
		return c
	})
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpUint32(a, b uint32) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpString(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// DrawDiplomacy renders the diplomacy modal. No-op while pc.DiploOpen is false.
func (s *StatePlaying) DrawDiplomacy(screen *ebiten.Image) {
	pc := s.PC
	if !pc.DiploOpen {
		return
	}
	world := s.Status.TM.World
	tick := s.Status.TM.Ticks

	cx, cy, closed := ModalFrame(screen, "DIPLOMACY", diploW, diploH)
	if closed {
		pc.DiploOpen = false
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	_, my0 := modalOrigin(sw, sh, diploW, diploH)
	bottom := my0 + diploH - PadM
	cw := diploW - 2*PadM

	player, ok := pc.PossessedEntity()
	if !ok {
		DrawText(screen, "No body, no politics.", cx, cy, TextDim)
		return
	}
	rank, _, countryID := systems.GetRank(world, player)
	if countryID == 0 {
		DrawText(screen, "You belong to no realm; the great game plays on without you.", cx, cy, TextDim)
		return
	}

	DrawText(screen, fmt.Sprintf("Your realm: %s   Capital: %s",
		realmName(countryID), countryCapitalCityName(world, countryID)), cx, cy, TextCol)
	cy += 20

	rels := systems.GetRelations(world, countryID)
	if len(rels) == 0 {
		DrawText(screen, "No foreign realms known yet; diplomacy awakens as countries meet.", cx, cy, TextDim)
		return
	}
	rows := buildDiploRows(world, rels)

	// U7: filter box on big rosters.
	names := make([]string, len(rows))
	for i := range rows {
		names[i] = rows[i].realm
	}
	if len(rows) > rosterFilterThreshold {
		diploFilter.Draw(screen, cx, cy, cw, SearchBoxH, names)
		cy += SearchBoxH + PadS
	}
	visible := rosterIndices(names, diploFilter.Query)
	sortDiploIndices(visible, rows, tick, diploTable.SortCol, diploTable.SortAsc)

	cells := make([][]string, len(visible))
	for i, vi := range visible {
		cells[i] = diploCells(rows[vi], tick, diploSelected)
	}

	tableY := cy
	tableH := bottom - 56 - tableY
	if clicked := diploTable.Draw(screen, cx, tableY, cw, tableH, cells); clicked >= 0 {
		diploSelected = rows[visible[clicked]].rel.TargetCountry
	}
	tableHeaderTips(screen, &diploTable, cx, tableY, cw, []string{
		"A foreign country, led from its capital.",
		"The settlement that rules that realm.",
		"How warmly they regard you (-100..+100); high opinion enables alliances, low breeds war.",
		"Peace, Truce (war blocked until it expires), Allied, or AT WAR.",
		"Positive = your side is winning; at +/-100 the war settles and the loser pays tribute.",
	})

	// Selection line + action row.
	selY := bottom - 50
	var selRow *diploRow
	for i := range rows {
		if diploSelected != 0 && rows[i].rel.TargetCountry == diploSelected {
			selRow = &rows[i]
		}
	}
	if selRow == nil {
		DrawText(screen, "Select a realm to conduct diplomacy.", cx, selY, TextDim)
	} else {
		DrawText(screen, fmt.Sprintf("Selected: %s  (opinion %+d, %s)",
			selRow.realm, selRow.rel.Opinion, diploStatus(selRow.rel, tick)), cx, selY, AccentCol)
	}

	byY := bottom - 28
	if rank != systems.RankSovereign {
		DrawText(screen, "Only a Sovereign conducts diplomacy.", cx, byY, TextDim)
		return
	}
	if selRow == nil {
		return
	}
	target := selRow.rel.TargetCountry
	s.drawDiploActions(screen, cx, byY, countryID, target, tick)
}

// drawDiploActions renders the sovereign action buttons for the selected
// realm and pushes the outcome (or the returned error) as a notification.
func (s *StatePlaying) drawDiploActions(screen *ebiten.Image, bx, by int, countryID, target uint32, tick uint64) {
	pc := s.PC
	world := s.Status.TM.World

	improveLabel := fmt.Sprintf("Improve (%.0fg)", systems.ImproveRelationsCost)
	improveW := MeasureText(improveLabel) + 16
	if Button(screen, improveLabel, bx, by, improveW, 20) {
		if err := systems.ImproveRelations(world, countryID, target, tick); err != nil {
			pc.PushNote("Envoys rebuffed: "+err.Error(), tick)
		} else {
			pc.PushNote(fmt.Sprintf("Envoys sent to %s (+%d opinion).",
				realmName(target), systems.ImproveRelationsOpinionGain), tick)
		}
	}
	HoverTip(screen, bx, by, improveW, 20, fmt.Sprintf(
		"Spend %.0f gold from your treasury to raise mutual opinion by %d (cooldown applies).",
		systems.ImproveRelationsCost, systems.ImproveRelationsOpinionGain))
	bx += improveW + 6

	if Button(screen, "Ally", bx, by, 56, 20) {
		if err := systems.FormAlliance(world, countryID, target); err != nil {
			pc.PushNote("Alliance refused: "+err.Error(), tick)
		} else {
			pc.PushNote("Alliance sealed with "+realmName(target)+".", tick)
		}
	}
	HoverTip(screen, bx, by, 56, 20, fmt.Sprintf(
		"They accept at opinion %d or above (never while at war).", systems.AllianceOfferMinOpinion))
	bx += 62

	if Button(screen, "Break Alliance", bx, by, 108, 20) {
		if err := systems.BreakAlliance(world, countryID, target); err != nil {
			pc.PushNote("Cannot break alliance: "+err.Error(), tick)
		} else {
			pc.PushNote("Alliance with "+realmName(target)+" dissolved.", tick)
		}
	}
	HoverTip(screen, bx, by, 108, 20, fmt.Sprintf(
		"Dissolves the pact and costs %d mutual opinion.", systems.BreakAllianceOpinionPenalty))
	bx += 114

	if Button(screen, "Declare War", bx, by, 96, 20) {
		if err := systems.DeclareWarAction(world, countryID, target, tick); err != nil {
			pc.PushNote("War averted: "+err.Error(), tick)
		} else {
			pc.PushNote("WAR! Your realm marches on "+realmName(target)+".", tick)
		}
	}
	HoverTip(screen, bx, by, 96, 20,
		"Opens a war through the macro war model; truces block declarations.")
	bx += 102

	if Button(screen, "Sue for Peace", bx, by, 104, 20) {
		if err := systems.SueForPeace(world, countryID, target, tick); err != nil {
			pc.PushNote("Peace refused: "+err.Error(), tick)
		} else {
			pc.PushNote("Peace with "+realmName(target)+"; a truce holds.", tick)
		}
	}
	HoverTip(screen, bx, by, 104, 20,
		"They accept only while the war favors them; you pay the tribute, both sides get a truce.")
}
