package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Widget kit v2: the reusable pieces every strategy panel is built from —
// Tabs, SearchableList, Table, Tooltip, ModalFrame. Immediate-mode like the
// rest of the package: clicks come from the BeginUIFrame snapshot, hover
// visuals read the live cursor.

// Spacing grid (design system): new panels use these instead of ad-hoc
// offsets so chrome stays consistent.
const (
	PadS = 4
	PadM = 8
	PadL = 16

	TabH        = 22 // tab strip height
	SearchBoxH  = 20 // SearchableList filter box height
	TableHeadH  = 20 // Table header row height
	ModalTitleH = 24 // ModalFrame title bar height
	KitRowH     = 16 // default list/table row height
)

var (
	rowStripe = color.RGBA{255, 255, 255, 10}
	modalDim  = color.RGBA{0, 0, 0, 110}
)

// Tabs draws a horizontal tab strip filling width w and returns the newly
// clicked tab index, or active when no tab was clicked this frame.
func Tabs(dst *ebiten.Image, x, y, w int, labels []string, active int) int {
	n := len(labels)
	if n == 0 {
		return active
	}
	tw := w / n
	if uiClick && PointIn(uiMX, uiMY, x, y, tw*n, TabH) {
		if i := tabHit(x, tw, n, uiMX); i >= 0 {
			uiClick = false
			active = i
		}
	}
	mx, my := ebiten.CursorPosition()
	for i, lb := range labels {
		tx := x + i*tw
		bg := ButtonBG
		if i == active {
			bg = PanelBG
		} else if PointIn(mx, my, tx, y, tw, TabH) {
			bg = ButtonHover
		}
		ebitenutil.DrawRect(dst, float64(tx), float64(y), float64(tw), float64(TabH), bg)
		ebitenutil.DrawRect(dst, float64(tx), float64(y), float64(tw), 1, PanelBorder)
		ebitenutil.DrawRect(dst, float64(tx), float64(y), 1, float64(TabH), PanelBorder)
		if i == active {
			ebitenutil.DrawRect(dst, float64(tx), float64(y+TabH-2), float64(tw), 2, AccentCol)
		}
		col := TextDim
		if i == active {
			col = TextCol
		}
		DrawText(dst, lb, tx+(tw-MeasureText(lb))/2, y+(TabH-13)/2, col)
	}
	ebitenutil.DrawRect(dst, float64(x+tw*n-1), float64(y), 1, float64(TabH), PanelBorder)
	return active
}

// tabHit maps a cursor x to a tab index for a strip at x with n tabs of
// width tw each; -1 when outside the strip.
func tabHit(x, tw, n, px int) int {
	if tw <= 0 || px < x || px >= x+tw*n {
		return -1
	}
	return (px - x) / tw
}

// kitFocus is the one SearchableList holding keyboard focus. Clicking a
// list's filter box (or calling Focus) claims it; clicking elsewhere drops it.
var kitFocus *SearchableList

// SearchableList is a filterable, keyboard-navigable list. While Focused,
// typed characters edit Query (case-insensitive substring filter), up/down
// move the selection and Enter confirms it via EnterIndex. Every index it
// reports refers to the caller's original items slice.
type SearchableList struct {
	Query      string
	Focused    bool
	List       ScrollList
	Sel        int // selection within the filtered view
	EnterIndex int // original-items index confirmed with Enter this frame; -1 otherwise

	navUp, navDown, navEnter bool
	lastFrame                uint64
}

// Focus grants s the keyboard focus (only one SearchableList holds it).
func (s *SearchableList) Focus() {
	s.Focused = true
	kitFocus = s
}

// Update captures this tick's typed characters and navigation keys from the
// BeginUIFrame snapshot. Calling it explicitly is optional: Draw self-updates
// once per frame when the caller skips it.
func (s *SearchableList) Update() {
	s.lastFrame = uiFrameID
	s.EnterIndex = -1
	if !s.Focused {
		return
	}
	s.applyTyping(uiRunes, uiBackspace)
	s.navUp = s.navUp || uiUp
	s.navDown = s.navDown || uiDown
	s.navEnter = s.navEnter || uiEnter
}

// applyTyping edits Query with typed runes and backspace; any change resets
// the selection and scroll to the top of the (new) filtered view.
func (s *SearchableList) applyTyping(runes []rune, backspace bool) {
	changed := false
	for _, r := range runes {
		if r >= ' ' && r != 0x7f {
			s.Query += string(r)
			changed = true
		}
	}
	if backspace && s.Query != "" {
		rs := []rune(s.Query)
		s.Query = string(rs[:len(rs)-1])
		changed = true
	}
	if changed {
		s.Sel = 0
		s.List.Offset = 0
	}
}

// applyNav applies pending up/down/enter to the filtered view: clamps the
// selection, resolves Enter to an original-items index, and scrolls the
// selection into view.
func (s *SearchableList) applyNav(filtered []int) {
	navEnter := s.navEnter
	navUp, navDown := s.navUp, s.navDown
	s.navUp, s.navDown, s.navEnter = false, false, false
	n := len(filtered)
	if n == 0 {
		s.Sel = 0
		return
	}
	if s.Sel >= n {
		s.Sel = n - 1
	}
	if s.Sel < 0 {
		s.Sel = 0
	}
	if navUp && s.Sel > 0 {
		s.Sel--
	}
	if navDown && s.Sel < n-1 {
		s.Sel++
	}
	if navEnter {
		s.EnterIndex = filtered[s.Sel]
	}
	if s.List.RowH <= 0 {
		s.List.RowH = KitRowH
	}
	if rows := s.List.H / s.List.RowH; rows > 0 {
		if s.Sel < s.List.Offset {
			s.List.Offset = s.Sel
		}
		if s.Sel >= s.List.Offset+rows {
			s.List.Offset = s.Sel - rows + 1
		}
	}
}

// filterIndices returns the indices of items containing query
// (case-insensitive substring), preserving order. Empty query matches all.
func filterIndices(items []string, query string) []int {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]int, 0, len(items))
	for i, it := range items {
		if q == "" || strings.Contains(strings.ToLower(it), q) {
			out = append(out, i)
		}
	}
	return out
}

// Draw renders the filter box and the filtered list inside (x,y,w,h) and
// returns the original-items index of a clicked row, or -1. Check EnterIndex
// after Draw for a keyboard confirmation.
func (s *SearchableList) Draw(dst *ebiten.Image, x, y, w, h int, items []string) int {
	if s.lastFrame != uiFrameID {
		s.Update()
	}
	if kitFocus != nil && kitFocus != s {
		s.Focused = false
	}
	if uiClick {
		if PointIn(uiMX, uiMY, x, y, w, SearchBoxH) {
			uiClick = false
			s.Focus()
		} else {
			s.Focused = false
			if kitFocus == s {
				kitFocus = nil
			}
		}
	}

	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), float64(SearchBoxH), BarGray)
	border := PanelBorder
	if s.Focused {
		border = AccentCol
	}
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), 1, border)
	ebitenutil.DrawRect(dst, float64(x), float64(y+SearchBoxH-1), float64(w), 1, border)
	ebitenutil.DrawRect(dst, float64(x), float64(y), 1, float64(SearchBoxH), border)
	ebitenutil.DrawRect(dst, float64(x+w-1), float64(y), 1, float64(SearchBoxH), border)
	ty := y + (SearchBoxH-13)/2
	if s.Query == "" && !s.Focused {
		DrawText(dst, "type to filter...", x+6, ty, TextDim)
	} else {
		q := s.Query
		if s.Focused {
			q += "_"
		}
		DrawText(dst, truncateToWidth(q, w-12), x+6, ty, TextCol)
	}

	filtered := filterIndices(items, s.Query)
	s.applyNav(filtered)

	ly := y + SearchBoxH + PadS
	lh := h - SearchBoxH - PadS
	s.List.X, s.List.Y, s.List.W, s.List.H = x, ly, w, lh
	if s.List.RowH <= 0 {
		s.List.RowH = KitRowH
	}
	scrollBySnapshotWheel(&s.List)
	clicked := -1
	from, to := s.List.VisibleRange(len(filtered))
	mx, my := ebiten.CursorPosition()
	for vi := from; vi < to; vi++ {
		ry := ly + (vi-from)*s.List.RowH
		if vi == s.Sel {
			ebitenutil.DrawRect(dst, float64(x), float64(ry), float64(w), float64(s.List.RowH), ButtonBG)
			ebitenutil.DrawRect(dst, float64(x), float64(ry), 2, float64(s.List.RowH), AccentCol)
		} else if PointIn(mx, my, x, ry, w, s.List.RowH) {
			ebitenutil.DrawRect(dst, float64(x), float64(ry), float64(w), float64(s.List.RowH), ButtonHover)
		}
		DrawText(dst, truncateToWidth(items[filtered[vi]], w-12), x+6, ry+(s.List.RowH-13)/2, TextCol)
		if consumeClickIn(x, ry, w, s.List.RowH) {
			clicked = filtered[vi]
			s.Sel = vi
			s.Focus()
		}
	}
	if len(filtered) == 0 {
		DrawText(dst, "no matches", x+6, ly+PadS, TextDim)
	}
	return clicked
}

// scrollBySnapshotWheel scrolls l with the wheel movement captured at
// BeginUIFrame, consuming it so stacked lists don't double-scroll.
func scrollBySnapshotWheel(l *ScrollList) {
	if uiWheelY == 0 || !PointIn(uiMX, uiMY, l.X, l.Y, l.W, l.H) {
		return
	}
	l.Offset -= int(uiWheelY) * 2
	if l.Offset < 0 {
		l.Offset = 0
	}
	uiWheelY = 0
}

// truncateToWidth shortens s with a ".." suffix so it fits maxPx.
func truncateToWidth(s string, maxPx int) string {
	if MeasureText(s) <= maxPx {
		return s
	}
	rs := []rune(s)
	for len(rs) > 0 && MeasureText(string(rs)+"..") > maxPx {
		rs = rs[:len(rs)-1]
	}
	return string(rs) + ".."
}

// Table renders sortable columns over row data. Sorting the data is the
// CALLER's job: Table only reports SortCol/SortAsc (toggled by header
// clicks) and returns the clicked row index into rows exactly as passed.
// The zero value sorts column 0; set SortAsc explicitly for an initial
// ascending order.
type Table struct {
	Columns []string
	Widths  []int // pixel width per column; the last column stretches to fill
	SortCol int
	SortAsc bool
	List    ScrollList
}

// headerClick applies a header click: the current sort column flips
// direction, a new column starts ascending.
func (t *Table) headerClick(col int) {
	if col == t.SortCol {
		t.SortAsc = !t.SortAsc
	} else {
		t.SortCol = col
		t.SortAsc = true
	}
}

// colOffset returns the x offset of column i within the table.
func colOffset(widths []int, i int) int {
	off := 0
	for c := 0; c < i && c < len(widths); c++ {
		off += widths[c]
	}
	return off
}

// colWidth returns column i's width; the last column stretches to fill w.
func (t *Table) colWidth(i, w int) int {
	if i >= len(t.Widths) {
		return 60
	}
	if i == len(t.Columns)-1 {
		if rest := w - colOffset(t.Widths, i); rest > t.Widths[i] {
			return rest
		}
	}
	return t.Widths[i]
}

// Draw renders the table inside (x,y,w,h) and returns the clicked row index,
// or -1. Header clicks toggle SortCol/SortAsc for the caller to apply.
func (t *Table) Draw(dst *ebiten.Image, x, y, w, h int, rows [][]string) int {
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), float64(TableHeadH), ButtonBG)
	mx, my := ebiten.CursorPosition()
	for i, col := range t.Columns {
		cx := x + colOffset(t.Widths, i)
		cw := t.colWidth(i, w)
		if PointIn(mx, my, cx, y, cw, TableHeadH) {
			ebitenutil.DrawRect(dst, float64(cx), float64(y), float64(cw), float64(TableHeadH), ButtonHover)
		}
		label := col
		if i == t.SortCol {
			if t.SortAsc {
				label += " ^"
			} else {
				label += " v"
			}
		}
		DrawText(dst, truncateToWidth(label, cw-8), cx+PadS, y+(TableHeadH-13)/2, AccentCol)
		if i > 0 {
			ebitenutil.DrawRect(dst, float64(cx), float64(y), 1, float64(TableHeadH), PanelBorder)
		}
		if consumeClickIn(cx, y, cw, TableHeadH) {
			t.headerClick(i)
		}
	}
	ly := y + TableHeadH
	t.List.X, t.List.Y, t.List.W, t.List.H = x, ly, w, h-TableHeadH
	if t.List.RowH <= 0 {
		t.List.RowH = KitRowH
	}
	scrollBySnapshotWheel(&t.List)
	clicked := -1
	from, to := t.List.VisibleRange(len(rows))
	for ri := from; ri < to; ri++ {
		ry := ly + (ri-from)*t.List.RowH
		if ri%2 == 1 {
			ebitenutil.DrawRect(dst, float64(x), float64(ry), float64(w), float64(t.List.RowH), rowStripe)
		}
		if PointIn(mx, my, x, ry, w, t.List.RowH) {
			ebitenutil.DrawRect(dst, float64(x), float64(ry), float64(w), float64(t.List.RowH), ButtonHover)
		}
		for ci := range t.Columns {
			if ci >= len(rows[ri]) {
				break
			}
			cx := x + colOffset(t.Widths, ci)
			cw := t.colWidth(ci, w)
			DrawText(dst, truncateToWidth(rows[ri][ci], cw-8), cx+PadS, ry+(t.List.RowH-13)/2, TextCol)
		}
		if consumeClickIn(x, ry, w, t.List.RowH) {
			clicked = ri
		}
	}
	return clicked
}

// Tooltips: widgets queue tips during Draw; FlushTooltips renders the queue
// last so tips always sit above every panel.

type queuedTip struct {
	x, y int
	text string
}

var tipQueue []queuedTip

// Tip queues a tooltip with its top-left near (x,y). The dst parameter is
// accepted for call-site symmetry with other widgets; drawing happens in
// FlushTooltips.
func Tip(_ *ebiten.Image, x, y int, text string) {
	if text == "" {
		return
	}
	tipQueue = append(tipQueue, queuedTip{x: x, y: y, text: text})
}

// FlushTooltips draws every queued tooltip and clears the queue. Call once,
// last, in the top-level Draw.
func FlushTooltips(dst *ebiten.Image) {
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	for _, q := range tipQueue {
		lines := strings.Split(q.text, "\n")
		tw := 0
		for _, ln := range lines {
			if lw := MeasureText(ln); lw > tw {
				tw = lw
			}
		}
		w := tw + 2*PadM
		h := len(lines)*14 + 2*PadS
		tx, ty := tipClamp(q.x, q.y, w, h, sw, sh)
		DrawPanel(dst, tx, ty, w, h)
		for i, ln := range lines {
			DrawText(dst, ln, tx+PadM, ty+PadS+i*14, TextCol)
		}
	}
	tipQueue = tipQueue[:0]
}

// tipClamp keeps a w×h box inside a sw×sh screen.
func tipClamp(x, y, w, h, sw, sh int) (int, int) {
	if x+w > sw {
		x = sw - w
	}
	if y+h > sh {
		y = sh - h
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// Hover-delayed tooltips: HoverTip records the hovered rect during Draw;
// BeginUIFrame advances the timer on the Update timeline so the delay is
// tick-accurate regardless of FPS.

const tipDelayTicks = 24 // ~0.4s at 60 TPS

var (
	hoverTipRect  [4]int // rect the cursor has been resting in
	hoverTipTicks int    // ticks spent inside hoverTipRect
	hoverTipCand  [4]int // rect hovered during the current draw pass
	hoverTipSeen  bool
)

// HoverTip queues text near the cursor once it has hovered (x,y,w,h) for
// tipDelayTicks. Call during Draw wherever the hover target is drawn.
func HoverTip(dst *ebiten.Image, x, y, w, h int, text string) {
	mx, my := ebiten.CursorPosition()
	if !PointIn(mx, my, x, y, w, h) {
		return
	}
	r := [4]int{x, y, w, h}
	hoverTipCand = r
	hoverTipSeen = true
	if r == hoverTipRect && hoverTipTicks >= tipDelayTicks {
		Tip(dst, mx+14, my+18, text)
	}
}

// advanceHoverTip steps the hover-delay timer; BeginUIFrame calls it once per
// Update tick.
func advanceHoverTip() {
	if hoverTipSeen && hoverTipCand == hoverTipRect {
		hoverTipTicks++
	} else {
		hoverTipRect = hoverTipCand
		hoverTipTicks = 0
	}
	hoverTipSeen = false
	hoverTipCand = [4]int{}
}

// ModalFrame draws a dimmed backdrop and a centered panel with a title bar
// and close X. It returns the content origin (inside the panel, below the
// title bar, inset by PadM) and closed=true when the X or Esc was pressed.
// Clicks outside the panel are consumed, and BeginUIFrame drops any click a
// modal frame left unclaimed, so nothing leaks to the world while a modal is
// up. Draw the topmost modal's frame first when stacking so it gets Esc.
func ModalFrame(dst *ebiten.Image, title string, w, h int) (int, int, bool) {
	modalSeen = true
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	x, y := modalOrigin(sw, sh, w, h)
	ebitenutil.DrawRect(dst, 0, 0, float64(sw), float64(sh), modalDim)
	if uiClick && !PointIn(uiMX, uiMY, x, y, w, h) {
		uiClick = false
	}
	DrawPanel(dst, x, y, w, h)
	ebitenutil.DrawRect(dst, float64(x+1), float64(y+1), float64(w-2), float64(ModalTitleH-1), ButtonBG)
	ebitenutil.DrawRect(dst, float64(x+1), float64(y+ModalTitleH), float64(w-2), 1, PanelBorder)
	DrawText(dst, title, x+PadM, y+(ModalTitleH-13)/2, AccentCol)

	bs := ModalTitleH - 6
	bx, by := x+w-bs-3, y+3
	mx, my := ebiten.CursorPosition()
	if PointIn(mx, my, bx, by, bs, bs) {
		ebitenutil.DrawRect(dst, float64(bx), float64(by), float64(bs), float64(bs), BarRed)
	}
	DrawText(dst, "X", bx+(bs-MeasureText("X"))/2, by+(bs-13)/2, TextCol)
	closed := consumeClickIn(bx, by, bs, bs) || consumeEsc()
	return x + PadM, y + ModalTitleH + PadM, closed
}

// modalOrigin centers a w×h panel on a sw×sh screen, clamped to
// non-negative coordinates for oversized panels.
func modalOrigin(sw, sh, w, h int) (int, int) {
	x := (sw - w) / 2
	y := (sh - h) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}
