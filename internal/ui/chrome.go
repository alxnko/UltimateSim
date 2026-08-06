package ui

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec U5 + G3): game chrome. The bottom bar belongs to
// hud.go; this file adds the TOP-LEFT horizontal tab strip that toggles the
// strategy panels, and the always-visible progression hint strip (current
// rank + the concrete next step up the ladder) right under it.
//
// All input goes through the BeginUIFrame snapshot (leftClicked/consumeClickIn
// inside the kit widgets); no inpututil in Draw paths.

// Chrome layout constants.
const (
	chromeX     = 8  // strip top-left
	chromeY     = 8
	chromeHintH = 20 // progression hint line height
)

// Chrome tab indices (order of chromeTabs).
const (
	chromeTabDiplomacy = 0
	chromeTabDynasty   = 1
	chromeTabMarket    = 2
	chromeTabLaws      = 3
	chromeTabGoals     = 4
	chromeTabCharacter = 5
	chromeTabChronicle = 6
)

// chromeTabs is the fixed tab order of the strip.
var chromeTabs = []string{"Diplomacy", "Dynasty", "Market", "Laws", "Goals", "Character", "Chronicle"}

// chromeTabW returns the uniform tab width: the widest label plus padding.
func chromeTabW() int {
	w := 0
	for _, lb := range chromeTabs {
		if lw := MeasureText(lb) + 2*PadM; lw > w {
			w = lw
		}
	}
	return w
}

// ChromeOccludes reports whether a screen point lands on the chrome strip
// (tab bar + hint line) so world clicks there can be suppressed (UIOccludes).
func ChromeOccludes(x, y int) bool {
	return PointIn(x, y, chromeX, chromeY,
		chromeTabW()*len(chromeTabs), TabH+PadS+chromeHintH)
}

// chromeActiveTab maps the open-panel flags to the strip's active tab index,
// -1 when no chrome-toggled panel is open.
func chromeActiveTab(pc *PlayerContext) int {
	switch {
	case pc.DiploOpen:
		return chromeTabDiplomacy
	case pc.DynastyOpen:
		return chromeTabDynasty
	case pc.MarketOpen:
		return chromeTabMarket
	case pc.LawsOpen:
		return chromeTabLaws
	case pc.AmbitionsOpen:
		return chromeTabGoals
	case pc.CharOpen:
		return chromeTabCharacter
	case pc.EventLogOpen:
		return chromeTabChronicle
	}
	return -1
}

// closeChromePanels clears every chrome-toggled panel flag.
func closeChromePanels(pc *PlayerContext) {
	pc.DiploOpen = false
	pc.DynastyOpen = false
	pc.MarketOpen = false
	pc.LawsOpen = false
	pc.AmbitionsOpen = false
	pc.CharOpen = false
	pc.EventLogOpen = false
}

// toggleChromeTab toggles tab i's panel: an open panel closes; otherwise every
// other chrome panel closes and tab i's panel opens. Also the hotkey target
// (state_playing wiring calls it for the diplomacy/market keys).
func (s *StatePlaying) toggleChromeTab(i int) {
	pc := s.PC
	wasActive := chromeActiveTab(pc) == i
	closeChromePanels(pc)
	if wasActive {
		return
	}
	switch i {
	case chromeTabDiplomacy:
		pc.DiploOpen = true
	case chromeTabDynasty:
		pc.DynastyOpen = true
	case chromeTabMarket:
		pc.MarketOpen = true
	case chromeTabLaws:
		s.openOwnCityLaws()
	case chromeTabGoals:
		pc.AmbitionsOpen = true
	case chromeTabCharacter:
		pc.CharOpen = true
	case chromeTabChronicle:
		pc.EventLogOpen = true
	}
}

// openOwnCityLaws targets the laws panel at the player's own city (the Laws
// tab has no clicked village; the player's home settlement is the target).
func (s *StatePlaying) openOwnCityLaws() {
	pc := s.PC
	world := s.Status.TM.World
	player, ok := pc.PossessedEntity()
	if !ok {
		return
	}
	_, cityID, _ := systems.GetRank(world, player)
	if cityID == 0 {
		pc.PushNote("You have no home city whose laws you could read.", s.Status.TM.Ticks)
		return
	}
	village, found := findCityVillage(world, cityID)
	if !found {
		pc.PushNote("Your city cannot be found.", s.Status.TM.Ticks)
		return
	}
	pc.LawsVillage = village
	pc.LawsOpen = true
}

// findCityVillage locates the living village entity whose city ID
// (uint32(Identity.ID), the CityBinder convention) matches. The query is
// always iterated to exhaustion.
func findCityVillage(world *ecs.World, cityID uint32) (ecs.Entity, bool) {
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	var out ecs.Entity
	found := false
	q := world.Query(ecs.All(villageID, identID))
	for q.Next() {
		if found {
			continue
		}
		ident := (*components.Identity)(q.Get(identID))
		if uint32(ident.ID) == cityID {
			out = q.Entity()
			found = true
		}
	}
	return out, found
}

// cityIsCapital reports whether the settlement with the given city ID carries
// the CapitalComponent. Full query iteration.
func cityIsCapital(world *ecs.World, cityID uint32) bool {
	if cityID == 0 {
		return false
	}
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	isCap := false
	q := world.Query(ecs.All(villageID, identID))
	for q.Next() {
		ident := (*components.Identity)(q.Get(identID))
		if uint32(ident.ID) == cityID && q.Has(capitalID) {
			isCap = true
		}
	}
	return isCap
}

// progressionHint is the pure G3 ladder text: the concrete next step for a
// rank. cityName may be "" when the player has no home city yet.
func progressionHint(rank uint8, cityName string, canClaim, isCapital bool) string {
	switch rank {
	case systems.RankSovereign:
		return "Direct your realm: diplomacy, taxes, war"
	case systems.RankRuler:
		if isCapital {
			return "You rule the realm"
		}
		return "Grow legitimacy; your capital awaits"
	default:
		if cityName == "" {
			return "Find a city to call home"
		}
		if canClaim {
			return "Claim leadership of " + cityName + " - you are ready!"
		}
		return "Win friends to claim leadership of " + cityName
	}
}

// DrawChrome renders the top-left tab strip and the progression hint line.
// Call it in the Draw chain BEFORE the modals so panels stack above it.
func (s *StatePlaying) DrawChrome(screen *ebiten.Image) {
	pc := s.PC
	if pc.Warmup > 0 || pc.SelectOpen || pc.HeirOpen || pc.GameOverOpen {
		return
	}

	tw := chromeTabW()
	n := len(chromeTabs)
	stripW := tw * n

	// Detect the strip click BEFORE Tabs consumes it: Tabs cannot report a
	// click on the already-active tab, which the toggle-off case needs.
	clicked := -1
	if leftClicked() && PointIn(uiMX, uiMY, chromeX, chromeY, stripW, TabH) {
		clicked = tabHit(chromeX, tw, n, uiMX)
	}
	Tabs(screen, chromeX, chromeY, stripW, chromeTabs, chromeActiveTab(pc))

	// Chrome panels may be switched while one of them is open (the strip has
	// first claim on the click either way); other modals own the screen.
	otherModal := pc.DialogOpen || pc.TradeOpen || pc.PauseOpen ||
		pc.HeirOpen || pc.GameOverOpen || pc.SelectOpen
	if clicked >= 0 && !otherModal && pc.Mode == ModeNormal {
		s.toggleChromeTab(clicked)
	}

	s.drawProgressionHint(screen, chromeX, chromeY+TabH+PadS)
}

// drawProgressionHint renders the G3 ladder line: current rank plus the next
// concrete step, always visible under the tab strip.
func (s *StatePlaying) drawProgressionHint(screen *ebiten.Image, x, y int) {
	world := s.Status.TM.World
	player, ok := s.PC.PossessedEntity()
	if !ok {
		return
	}
	rank, cityID, _ := systems.GetRank(world, player)
	cityName := ""
	if cityID != 0 {
		cityName = CityName(world, cityID)
	}
	canClaim := false
	if rank == systems.RankCitizen {
		canClaim = systems.CanClaimLeadership(world, s.Status.HookGraph, player)
	}
	hint := progressionHint(rank, cityName, canClaim, cityIsCapital(world, cityID))
	line := systems.RankName(rank) + " - " + hint

	w := MeasureText(line) + 2*PadM
	DrawPanel(screen, x, y, w, chromeHintH)
	col := TextCol
	if canClaim {
		col = AccentCol
	}
	DrawText(screen, line, x+PadM, y+(chromeHintH-13)/2, col)
	HoverTip(screen, x, y, w, chromeHintH,
		"Your current rank and the concrete next step up the ladder.")
}

// tableHeaderTips attaches HoverTip explanations to a Table's header cells
// (spec U8). Call right after t.Draw with the same geometry; an empty string
// skips that column.
func tableHeaderTips(dst *ebiten.Image, t *Table, x, y, w int, tips []string) {
	for i := range t.Columns {
		if i >= len(tips) || tips[i] == "" {
			continue
		}
		cx := x + colOffset(t.Widths, i)
		HoverTip(dst, cx, y, t.colWidth(i, w), TableHeadH, tips[i])
	}
}

// rosterFilterThreshold: rosters longer than this get a filter box (spec U7).
const rosterFilterThreshold = 8

// rosterIndices returns the visible row indices of a roster: every row while
// the roster is small (no filter box shown), the query-filtered subset above
// rosterFilterThreshold.
func rosterIndices(names []string, query string) []int {
	if len(names) <= rosterFilterThreshold {
		return allIndices(len(names))
	}
	return filterIndices(names, query)
}

// allIndices returns [0..n).
func allIndices(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}
