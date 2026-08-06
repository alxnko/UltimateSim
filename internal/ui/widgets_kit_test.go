package ui

import (
	"testing"
)

// resetUIInput zeroes every package-level input snapshot so tests are
// independent and -count=2 runs stay deterministic.
func resetUIInput() {
	uiMX, uiMY = 0, 0
	uiClick = false
	worldMX, worldMY = 0, 0
	worldClickPending = false
	uiRunes = nil
	uiBackspace, uiUp, uiDown, uiEnter, uiEsc = false, false, false, false, false
	uiWheelY = 0
	uiFrameID = 0
	modalSeen, modalWasSeen = false, false
	kitFocus = nil
	tipQueue = nil
	hoverTipRect, hoverTipCand = [4]int{}, [4]int{}
	hoverTipTicks, hoverTipSeen = 0, false
}

func TestFilterIndices(t *testing.T) {
	items := []string{"Aldric the Bold", "Berta Miller", "aldwin", "Cook"}
	cases := []struct {
		query string
		want  []int
	}{
		{"", []int{0, 1, 2, 3}},
		{"ald", []int{0, 2}},
		{"ALD", []int{0, 2}},
		{"miller", []int{1}},
		{"  cook  ", []int{3}},
		{"zzz", []int{}},
	}
	for _, c := range cases {
		got := filterIndices(items, c.query)
		if len(got) != len(c.want) {
			t.Fatalf("query %q: got %v want %v", c.query, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("query %q: got %v want %v", c.query, got, c.want)
			}
		}
	}
}

func TestSearchableListApplyTyping(t *testing.T) {
	s := &SearchableList{}
	s.applyTyping([]rune("Ab"), false)
	if s.Query != "Ab" {
		t.Fatalf("Query = %q, want Ab", s.Query)
	}
	s.applyTyping(nil, true)
	if s.Query != "A" {
		t.Fatalf("after backspace Query = %q, want A", s.Query)
	}
	// control runes are ignored; printable ones append
	s.applyTyping([]rune{'\t', 'c'}, false)
	if s.Query != "Ac" {
		t.Fatalf("control-rune filter: Query = %q, want Ac", s.Query)
	}
	// backspace on empty query is a no-op
	s.Query = ""
	s.applyTyping(nil, true)
	if s.Query != "" {
		t.Fatalf("backspace on empty: Query = %q", s.Query)
	}
}

func TestSearchableListTypingResetsScroll(t *testing.T) {
	s := &SearchableList{Sel: 3}
	s.List.Offset = 2
	s.applyTyping([]rune{'x'}, false)
	if s.Sel != 0 || s.List.Offset != 0 {
		t.Fatalf("typing must reset view: Sel=%d Offset=%d", s.Sel, s.List.Offset)
	}
}

func TestSearchableListUpdateFromSnapshot(t *testing.T) {
	resetUIInput()
	defer resetUIInput()

	s := &SearchableList{Focused: true}
	uiRunes = []rune("hi")
	s.Update()
	if s.Query != "hi" {
		t.Fatalf("Update did not capture runes: %q", s.Query)
	}
	if s.EnterIndex != -1 {
		t.Fatalf("EnterIndex not reset: %d", s.EnterIndex)
	}

	uiRunes = nil
	uiBackspace = true
	s.Update()
	if s.Query != "h" {
		t.Fatalf("Update did not capture backspace: %q", s.Query)
	}

	// unfocused lists ignore typing and nav
	uiBackspace = false
	uiRunes = []rune("zz")
	uiDown = true
	u := &SearchableList{Focused: false}
	u.Update()
	if u.Query != "" || u.navDown {
		t.Fatalf("unfocused list captured input: %q navDown=%v", u.Query, u.navDown)
	}
}

func TestSearchableListNav(t *testing.T) {
	s := &SearchableList{EnterIndex: -1}
	s.List.H = 32
	s.List.RowH = 16 // 2 visible rows
	filtered := []int{2, 5, 9}

	s.navDown = true
	s.applyNav(filtered)
	if s.Sel != 1 {
		t.Fatalf("down: Sel=%d want 1", s.Sel)
	}
	s.navDown = true
	s.applyNav(filtered)
	if s.Sel != 2 {
		t.Fatalf("down: Sel=%d want 2", s.Sel)
	}
	if s.List.Offset != 1 {
		t.Fatalf("scroll must follow selection: Offset=%d want 1", s.List.Offset)
	}
	// clamped at the end
	s.navDown = true
	s.applyNav(filtered)
	if s.Sel != 2 {
		t.Fatalf("down at end: Sel=%d want 2", s.Sel)
	}

	s.navUp = true
	s.applyNav(filtered)
	s.navUp = true
	s.applyNav(filtered)
	if s.Sel != 0 {
		t.Fatalf("up: Sel=%d want 0", s.Sel)
	}
	if s.List.Offset != 0 {
		t.Fatalf("scroll must follow selection up: Offset=%d want 0", s.List.Offset)
	}
	// clamped at the top
	s.navUp = true
	s.applyNav(filtered)
	if s.Sel != 0 {
		t.Fatalf("up at top: Sel=%d want 0", s.Sel)
	}

	// Enter resolves to the ORIGINAL item index
	s.Sel = 1
	s.navEnter = true
	s.applyNav(filtered)
	if s.EnterIndex != 5 {
		t.Fatalf("EnterIndex=%d want 5", s.EnterIndex)
	}

	// selection clamps when the filter shrinks
	s.Sel = 2
	s.applyNav([]int{7})
	if s.Sel != 0 {
		t.Fatalf("shrunk filter: Sel=%d want 0", s.Sel)
	}

	// empty filter is safe and consumes pending nav
	s.EnterIndex = -1
	s.navEnter = true
	s.applyNav(nil)
	if s.EnterIndex != -1 || s.navEnter {
		t.Fatalf("empty filter: EnterIndex=%d navEnter=%v", s.EnterIndex, s.navEnter)
	}
}

func TestTableHeaderClick(t *testing.T) {
	tb := &Table{Columns: []string{"Name", "Age"}, SortAsc: true}
	tb.headerClick(0)
	if tb.SortCol != 0 || tb.SortAsc {
		t.Fatalf("same column must flip: col=%d asc=%v", tb.SortCol, tb.SortAsc)
	}
	tb.headerClick(1)
	if tb.SortCol != 1 || !tb.SortAsc {
		t.Fatalf("new column must sort ascending: col=%d asc=%v", tb.SortCol, tb.SortAsc)
	}
	tb.headerClick(1)
	if tb.SortCol != 1 || tb.SortAsc {
		t.Fatalf("second click must flip: col=%d asc=%v", tb.SortCol, tb.SortAsc)
	}
}

func TestTableColumnGeometry(t *testing.T) {
	widths := []int{40, 60, 80}
	if got := colOffset(widths, 0); got != 0 {
		t.Fatalf("colOffset(0)=%d", got)
	}
	if got := colOffset(widths, 2); got != 100 {
		t.Fatalf("colOffset(2)=%d want 100", got)
	}
	// out-of-range column index is safe
	if got := colOffset(widths, 5); got != 180 {
		t.Fatalf("colOffset(5)=%d want 180", got)
	}

	tb := &Table{Columns: []string{"a", "b", "c"}, Widths: widths}
	if got := tb.colWidth(0, 300); got != 40 {
		t.Fatalf("colWidth(0)=%d want 40", got)
	}
	// last column stretches to fill the table width
	if got := tb.colWidth(2, 300); got != 200 {
		t.Fatalf("stretched colWidth(2)=%d want 200", got)
	}
	// but never shrinks below its declared width
	if got := tb.colWidth(2, 150); got != 80 {
		t.Fatalf("min colWidth(2)=%d want 80", got)
	}
}

func TestScrollListClamp(t *testing.T) {
	l := &ScrollList{H: 64, RowH: 16, Offset: 99}
	from, to := l.VisibleRange(10)
	if l.Offset != 6 || from != 6 || to != 10 {
		t.Fatalf("over-scroll: Offset=%d from=%d to=%d", l.Offset, from, to)
	}
	l.Offset = -3
	from, to = l.VisibleRange(10)
	if l.Offset != 0 || from != 0 || to != 4 {
		t.Fatalf("under-scroll: Offset=%d from=%d to=%d", l.Offset, from, to)
	}
	// fewer rows than fit: no scrolling possible
	l.Offset = 5
	from, to = l.VisibleRange(2)
	if l.Offset != 0 || from != 0 || to != 2 {
		t.Fatalf("short list: Offset=%d from=%d to=%d", l.Offset, from, to)
	}
}

func TestTabHit(t *testing.T) {
	// strip at x=100, 4 tabs of width 50
	cases := []struct {
		px   int
		want int
	}{
		{99, -1}, {100, 0}, {149, 0}, {150, 1}, {299, 3}, {300, -1},
	}
	for _, c := range cases {
		if got := tabHit(100, 50, 4, c.px); got != c.want {
			t.Fatalf("tabHit(px=%d)=%d want %d", c.px, got, c.want)
		}
	}
	if got := tabHit(0, 0, 4, 0); got != -1 {
		t.Fatalf("zero-width tabs must miss, got %d", got)
	}
}

func TestModalOrigin(t *testing.T) {
	x, y := modalOrigin(800, 600, 400, 300)
	if x != 200 || y != 150 {
		t.Fatalf("centered: (%d,%d) want (200,150)", x, y)
	}
	x, y = modalOrigin(100, 100, 400, 300)
	if x != 0 || y != 0 {
		t.Fatalf("oversized must clamp: (%d,%d)", x, y)
	}
}

func TestTipQueueAndClamp(t *testing.T) {
	resetUIInput()
	defer resetUIInput()

	Tip(nil, 10, 10, "")
	if len(tipQueue) != 0 {
		t.Fatalf("empty tip must not queue")
	}
	Tip(nil, 10, 10, "hello")
	Tip(nil, 20, 20, "line1\nline2")
	if len(tipQueue) != 2 {
		t.Fatalf("queue len=%d want 2", len(tipQueue))
	}

	x, y := tipClamp(790, 590, 60, 20, 800, 600)
	if x != 740 || y != 580 {
		t.Fatalf("clamp bottom-right: (%d,%d)", x, y)
	}
	x, y = tipClamp(-5, -5, 60, 20, 800, 600)
	if x != 0 || y != 0 {
		t.Fatalf("clamp top-left: (%d,%d)", x, y)
	}
}

func TestHoverTipDelay(t *testing.T) {
	resetUIInput()
	defer resetUIInput()

	r := [4]int{10, 10, 50, 20}
	// first sighting arms the rect without counting
	hoverTipCand, hoverTipSeen = r, true
	advanceHoverTip()
	if hoverTipRect != r || hoverTipTicks != 0 {
		t.Fatalf("arm: rect=%v ticks=%d", hoverTipRect, hoverTipTicks)
	}
	// steady hover accumulates ticks
	for i := 0; i < tipDelayTicks; i++ {
		hoverTipCand, hoverTipSeen = r, true
		advanceHoverTip()
	}
	if hoverTipTicks < tipDelayTicks {
		t.Fatalf("ticks=%d want >= %d", hoverTipTicks, tipDelayTicks)
	}
	// moving to another rect resets the timer
	hoverTipCand, hoverTipSeen = [4]int{99, 99, 10, 10}, true
	advanceHoverTip()
	if hoverTipTicks != 0 {
		t.Fatalf("rect change must reset ticks, got %d", hoverTipTicks)
	}
	// leaving every hover target disarms
	advanceHoverTip()
	if hoverTipRect != ([4]int{}) || hoverTipTicks != 0 {
		t.Fatalf("disarm: rect=%v ticks=%d", hoverTipRect, hoverTipTicks)
	}
}

func TestConsumeClickSnapshot(t *testing.T) {
	resetUIInput()
	defer resetUIInput()

	uiClick = true
	uiMX, uiMY = 5, 5
	if !consumeClickIn(0, 0, 10, 10) {
		t.Fatalf("click inside rect must consume")
	}
	if consumeClickIn(0, 0, 10, 10) {
		t.Fatalf("click must only be consumable once")
	}

	uiClick = true
	uiMX, uiMY = 50, 5
	if consumeClickIn(0, 0, 10, 10) {
		t.Fatalf("click outside rect must not consume")
	}
	if !uiClick {
		t.Fatalf("unconsumed click must survive for later widgets")
	}
}

func TestConsumeEsc(t *testing.T) {
	resetUIInput()
	defer resetUIInput()

	uiEsc = true
	if !consumeEsc() {
		t.Fatalf("pending Esc must be consumable")
	}
	if consumeEsc() {
		t.Fatalf("Esc must only be consumable once")
	}
}

func TestModalDropsUnclaimedClicks(t *testing.T) {
	resetUIInput()
	defer resetUIInput()

	// BeginUIFrame touches live Ebiten input; skip if unavailable headless.
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("headless input unavailable: %v", r)
		}
	}()

	// no modal: an unconsumed click is promoted to a world click
	uiClick = true
	uiMX, uiMY = 7, 9
	BeginUIFrame()
	if x, y, ok := TakeWorldClick(); !ok || x != 7 || y != 9 {
		t.Fatalf("world promotion failed: (%d,%d,%v)", x, y, ok)
	}

	// modal drawn last frame: the unconsumed click dies instead
	uiClick = true
	modalSeen = true
	BeginUIFrame()
	if _, _, ok := TakeWorldClick(); ok {
		t.Fatalf("modal frame must drop unclaimed clicks")
	}
	if !ModalOpen() {
		t.Fatalf("ModalOpen must report the modal seen last frame")
	}
}
