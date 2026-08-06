package ui

import (
	"fmt"
	"slices"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.6 UI + U7 + U8): the market panel. A sortable
// table of per-city prices for every good, the sovereign tax-rate control with
// the national treasury readout, and the trade-route ledger with the
// two-city "establish route" flow. All input via the BeginUIFrame snapshot.

const (
	marketW = 660
	marketH = 440

	marketTableW = 360 // left column: the price table
	taxRateStep  = 5   // percent per [-]/[+] click
)

// Persistent panel state (UI-only; never read by the simulation).
var (
	marketTable = Table{
		Columns: []string{"City", "Food", "Wood", "Stone", "Iron"},
		Widths:  []int{130, 55, 55, 55, 55},
		SortAsc: true,
	}
	marketFilter  SearchableList
	routePickList SearchableList
	routePicking  bool
	routeFromCity uint32 // first endpoint picked; 0 = none yet
)

// cityNamesFor resolves display names aligned with a MarketSnapshot slice.
// One full village query builds the lookup; unnamed cities fall back to
// "City N".
func cityNamesFor(world *ecs.World, snap []systems.CityPrices) []string {
	villageID := ecs.ComponentID[components.Village](world)
	identID := ecs.ComponentID[components.Identity](world)
	byID := make(map[uint32]string, len(snap))
	q := world.Query(ecs.All(villageID, identID))
	for q.Next() {
		ident := (*components.Identity)(q.Get(identID))
		if ident.Name != "" {
			byID[uint32(ident.ID)] = ident.Name
		}
	}
	out := make([]string, len(snap))
	for i, cp := range snap {
		if n, ok := byID[cp.CityID]; ok {
			out[i] = n
		} else {
			out[i] = fmt.Sprintf("City %d", cp.CityID)
		}
	}
	return out
}

// marketRow renders one price-table row.
func marketRow(name string, cp systems.CityPrices) []string {
	return []string{
		name,
		fmt.Sprintf("%.1f", cp.Food),
		fmt.Sprintf("%.1f", cp.Wood),
		fmt.Sprintf("%.1f", cp.Stone),
		fmt.Sprintf("%.1f", cp.Iron),
	}
}

// marketKey maps a sortable price column to its value.
func marketKey(cp systems.CityPrices, col int) float32 {
	switch col {
	case 1:
		return cp.Food
	case 2:
		return cp.Wood
	case 3:
		return cp.Stone
	case 4:
		return cp.Iron
	}
	return 0
}

// sortMarketIndices orders the visible row indices by the table's sort
// column, deterministically tie-broken by CityID ascending.
func sortMarketIndices(idx []int, snap []systems.CityPrices, names []string, col int, asc bool) {
	slices.SortStableFunc(idx, func(a, b int) int {
		c := 0
		if col == 0 {
			c = cmpString(names[a], names[b])
		} else {
			ka, kb := marketKey(snap[a], col), marketKey(snap[b], col)
			switch {
			case ka < kb:
				c = -1
			case ka > kb:
				c = 1
			}
		}
		if c == 0 {
			return cmpUint32(snap[a].CityID, snap[b].CityID)
		}
		if !asc {
			c = -c
		}
		return c
	})
}

// bumpTaxRate applies a signed step to a tax rate, clamped to the legal
// 0..MaxTaxRatePercent band (SetTaxRate re-clamps the top defensively).
func bumpTaxRate(cur uint8, delta int) uint8 {
	v := int(cur) + delta
	if v < 0 {
		v = 0
	}
	if v > int(components.MaxTaxRatePercent) {
		v = int(components.MaxTaxRatePercent)
	}
	return uint8(v)
}

// routeLabel renders one trade-route ledger line using the city-name lookup.
func routeLabel(nameByCity map[uint32]string, r systems.TradeRouteInfo) string {
	name := func(id uint32) string {
		if n, ok := nameByCity[id]; ok {
			return n
		}
		return fmt.Sprintf("City %d", id)
	}
	return fmt.Sprintf("%s <-> %s  x%d", name(r.FromCity), name(r.ToCity), r.Volume)
}

// DrawMarket renders the market modal. No-op while pc.MarketOpen is false.
func (s *StatePlaying) DrawMarket(screen *ebiten.Image) {
	pc := s.PC
	if !pc.MarketOpen {
		return
	}
	world := s.Status.TM.World
	tick := s.Status.TM.Ticks

	cx, cy, closed := ModalFrame(screen, "MARKET & TRADE", marketW, marketH)
	if closed {
		pc.MarketOpen = false
		routePicking = false
		routeFromCity = 0
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	_, my0 := modalOrigin(sw, sh, marketW, marketH)
	bottom := my0 + marketH - PadM
	cw := marketW - 2*PadM

	snap := systems.MarketSnapshot(world)
	names := cityNamesFor(world, snap)
	nameByCity := make(map[uint32]string, len(snap))
	for i, cp := range snap {
		nameByCity[cp.CityID] = names[i]
	}

	// --- Left column: the price table (U7 filter on big rosters) ---
	ly := cy
	if len(snap) == 0 {
		DrawText(screen, "No living markets anywhere.", cx, ly, TextDim)
	} else {
		if len(snap) > rosterFilterThreshold {
			marketFilter.Draw(screen, cx, ly, marketTableW, SearchBoxH, names)
			ly += SearchBoxH + PadS
		}
		visible := rosterIndices(names, marketFilter.Query)
		sortMarketIndices(visible, snap, names, marketTable.SortCol, marketTable.SortAsc)
		rows := make([][]string, len(visible))
		for i, vi := range visible {
			rows[i] = marketRow(names[vi], snap[vi])
		}
		marketTable.Draw(screen, cx, ly, marketTableW, bottom-ly, rows)
		tableHeaderTips(screen, &marketTable, cx, ly, marketTableW, []string{
			"Every living settlement with a market.",
			"Local price of food; traders buy where it is cheap and sell where it is dear.",
			"Local price of wood; construction demand pushes it up.",
			"Local price of stone; scarcity raises it.",
			"Local price of iron; the rarest good on most maps.",
		})
	}

	// --- Right column: taxes, treasury, trade routes ---
	rx := cx + marketTableW + PadL
	rw := cw - marketTableW - PadL
	ry := cy

	player, hasPlayer := pc.PossessedEntity()
	rank := systems.RankCitizen
	var countryID uint32
	if hasPlayer {
		rank, _, countryID = systems.GetRank(world, player)
	}

	DrawText(screen, "NATIONAL TAXES", rx, ry, AccentCol)
	ry += 18
	if countryID == 0 {
		DrawText(screen, "You serve no realm.", rx, ry, TextDim)
		ry += 18
	} else {
		rate := systems.GetTaxRate(world, countryID)
		rateLabel := fmt.Sprintf("Tax rate: %d%%", rate)
		DrawText(screen, rateLabel, rx, ry, TextCol)
		HoverTip(screen, rx, ry, MeasureText(rateLabel), 16, fmt.Sprintf(
			"National tax rate (0..%d%%). Villages pay this share of their market value into the sovereign treasury each collection.",
			components.MaxTaxRatePercent))
		if rank == systems.RankSovereign {
			if Button(screen, "-", rx+130, ry-2, 24, 16) {
				if err := systems.SetTaxRate(world, countryID, bumpTaxRate(rate, -taxRateStep)); err != nil {
					pc.PushNote("Tax decree failed: "+err.Error(), tick)
				}
			}
			if Button(screen, "+", rx+158, ry-2, 24, 16) {
				if err := systems.SetTaxRate(world, countryID, bumpTaxRate(rate, taxRateStep)); err != nil {
					pc.PushNote("Tax decree failed: "+err.Error(), tick)
				}
			}
		} else {
			DrawSmall(screen, "Only a Sovereign sets taxes.", rx+130, ry+2, TextDim)
		}
		ry += 20

		treasuryLabel := "Treasury: -"
		if capital, found := systems.FindCapitalOf(world, countryID); found {
			treasID := ecs.ComponentID[components.TreasuryComponent](world)
			if world.Has(capital, treasID) {
				t := (*components.TreasuryComponent)(world.Get(capital, treasID))
				treasuryLabel = fmt.Sprintf("Treasury: %.0f gold", t.Wealth)
			}
		}
		DrawText(screen, treasuryLabel, rx, ry, AccentCol)
		HoverTip(screen, rx, ry, MeasureText(treasuryLabel), 16,
			"Your capital's national treasury: taxes and trade cuts in, envoys and tributes out.")
		ry += 24
	}

	// --- Trade routes ---
	DrawText(screen, "TRADE ROUTES", rx, ry, AccentCol)
	HoverTip(screen, rx, ry, MeasureText("TRADE ROUTES"), 16,
		"Goods flow from the cheaper endpoint to the pricier one; both cities and their sovereigns take a cut of the profit.")
	ry += 18
	routes := systems.ListTradeRoutes(world)
	if len(routes) == 0 {
		DrawText(screen, "No routes established.", rx, ry, TextDim)
		ry += 16
	}
	const maxRouteLines = 6
	for i, r := range routes {
		if i >= maxRouteLines {
			DrawText(screen, fmt.Sprintf("...and %d more", len(routes)-maxRouteLines), rx, ry, TextDim)
			ry += 16
			break
		}
		DrawText(screen, routeLabel(nameByCity, r), rx, ry, TextCol)
		ry += 16
	}
	ry += 6

	// --- Establish-route flow: pick two cities from a searchable list ---
	if !routePicking {
		if Button(screen, "Establish route", rx, ry, 130, 20) {
			routePicking = true
			routeFromCity = 0
			routePickList.Query = ""
			routePickList.Focus()
		}
		return
	}

	prompt := "Pick the FIRST city:"
	if routeFromCity != 0 {
		prompt = "Link " + nameByCity[routeFromCity] + " with:"
	}
	DrawText(screen, prompt, rx, ry, TextCol)
	if Button(screen, "Cancel", rx+rw-64, ry-2, 60, 18) {
		routePicking = false
		routeFromCity = 0
		return
	}
	ry += 20
	pickH := bottom - ry
	picked := routePickList.Draw(screen, rx, ry, rw, pickH, names)
	if picked < 0 {
		picked = routePickList.EnterIndex
	}
	if picked >= 0 && picked < len(snap) {
		s.pickRouteCity(snap[picked].CityID, nameByCity, tick)
	}
}

// pickRouteCity advances the two-step route flow: the first pick arms the
// route, the second establishes it via systems.EstablishTradeRoute.
func (s *StatePlaying) pickRouteCity(cityID uint32, nameByCity map[uint32]string, tick uint64) {
	pc := s.PC
	world := s.Status.TM.World
	if routeFromCity == 0 {
		routeFromCity = cityID
		return
	}
	from := routeFromCity
	routePicking = false
	routeFromCity = 0
	if _, err := systems.EstablishTradeRoute(world, from, cityID); err != nil {
		pc.PushNote("No route: "+err.Error(), tick)
		return
	}
	pc.PushNote(fmt.Sprintf("Trade route established: %s <-> %s.",
		nameByCity[from], nameByCity[cityID]), tick)
}
