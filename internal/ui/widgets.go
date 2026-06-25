package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Shell Phase: immediate-mode UI widgets. Hover/click state is derived fresh
// each frame from the cursor — no retained widget tree.

// PointIn reports whether (px,py) lies within the rectangle.
func PointIn(px, py, x, y, w, h int) bool {
	return px >= x && px < x+w && py >= y && py < y+h
}

// leftClicked reports a fresh left-button press this frame.
func leftClicked() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}

// DrawPanel draws a filled panel with a 1px border.
func DrawPanel(dst *ebiten.Image, x, y, w, h int) {
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), float64(h), PanelBG)
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), 1, PanelBorder)
	ebitenutil.DrawRect(dst, float64(x), float64(y+h-1), float64(w), 1, PanelBorder)
	ebitenutil.DrawRect(dst, float64(x), float64(y), 1, float64(h), PanelBorder)
	ebitenutil.DrawRect(dst, float64(x+w-1), float64(y), 1, float64(h), PanelBorder)
}

// Button draws a labelled button and returns true on a fresh click.
func Button(dst *ebiten.Image, label string, x, y, w, h int) bool {
	mx, my := ebiten.CursorPosition()
	hover := PointIn(mx, my, x, y, w, h)
	bg := ButtonBG
	if hover {
		bg = ButtonHover
	}
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), float64(h), bg)
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), 1, PanelBorder)
	tx := x + (w-MeasureText(label))/2
	ty := y + (h-13)/2
	DrawText(dst, label, tx, ty, TextCol)
	return hover && leftClicked()
}

// Bar draws a labelled progress bar; frac is clamped to 0..1.
func Bar(dst *ebiten.Image, x, y, w, h int, frac float32, fg color.RGBA, label string) {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w), float64(h), BarGray)
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(w)*float64(frac), float64(h), fg)
	if label != "" {
		tx := x + (w-MeasureText(label))/2
		DrawText(dst, label, tx, y+(h-13)/2, TextCol)
	}
}

// Checkbox draws a labelled checkbox and returns true on a fresh click.
func Checkbox(dst *ebiten.Image, label string, x, y int, checked bool) bool {
	box := 14
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(box), float64(box), BarGray)
	ebitenutil.DrawRect(dst, float64(x), float64(y), float64(box), 1, PanelBorder)
	if checked {
		ebitenutil.DrawRect(dst, float64(x+3), float64(y+3), float64(box-6), float64(box-6), AccentCol)
	}
	DrawText(dst, label, x+box+6, y+1, TextCol)
	mx, my := ebiten.CursorPosition()
	w := box + 8 + MeasureText(label)
	return PointIn(mx, my, x, y, w, box) && leftClicked()
}

// ContextMenu is a small click-driven popup list.
type ContextMenu struct {
	X, Y    int
	Items   []string
	Visible bool
	W       int
}

const ctxRowH = 20

// Open positions the menu and makes it visible, sizing width to its widest item.
func (m *ContextMenu) Open(x, y int, items []string) {
	m.X, m.Y = x, y
	m.Items = items
	m.Visible = true
	m.W = 90
	for _, it := range items {
		if w := MeasureText(it) + 20; w > m.W {
			m.W = w
		}
	}
}

// IndexAt maps a screen point to a menu row index (-1 if outside).
func (m *ContextMenu) IndexAt(mx, my int) int {
	if !PointIn(mx, my, m.X, m.Y, m.W, len(m.Items)*ctxRowH) {
		return -1
	}
	return (my - m.Y) / ctxRowH
}

// Update processes input. Returns (chosenIndex, closed). A click inside picks
// an item and closes; a click outside or Esc closes with index -1.
func (m *ContextMenu) Update() (int, bool) {
	if !m.Visible {
		return -1, false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		m.Visible = false
		return -1, true
	}
	if !leftClicked() {
		return -1, false
	}
	mx, my := ebiten.CursorPosition()
	idx := m.IndexAt(mx, my)
	m.Visible = false
	if idx >= 0 && idx < len(m.Items) {
		return idx, true
	}
	return -1, true
}

// Draw renders the menu popup.
func (m *ContextMenu) Draw(dst *ebiten.Image) {
	if !m.Visible {
		return
	}
	h := len(m.Items) * ctxRowH
	DrawPanel(dst, m.X, m.Y, m.W, h)
	mx, my := ebiten.CursorPosition()
	hover := m.IndexAt(mx, my)
	for i, it := range m.Items {
		ry := m.Y + i*ctxRowH
		if i == hover {
			ebitenutil.DrawRect(dst, float64(m.X+1), float64(ry), float64(m.W-2), ctxRowH, ButtonHover)
		}
		DrawText(dst, it, m.X+8, ry+3, TextCol)
	}
}

// ScrollList tracks vertical scroll offset for a list region.
type ScrollList struct {
	X, Y, W, H int
	Offset     int
	RowH       int
}

// HandleWheel scrolls the list when the cursor hovers it.
func (l *ScrollList) HandleWheel() {
	mx, my := ebiten.CursorPosition()
	if !PointIn(mx, my, l.X, l.Y, l.W, l.H) {
		return
	}
	_, dy := ebiten.Wheel()
	if dy != 0 {
		l.Offset -= int(dy) * 2
		if l.Offset < 0 {
			l.Offset = 0
		}
	}
}

// VisibleRange returns the [from,to) row indices visible for n rows, clamping
// Offset so the list cannot scroll past its end.
func (l *ScrollList) VisibleRange(n int) (from, to int) {
	if l.RowH <= 0 {
		l.RowH = 16
	}
	rows := l.H / l.RowH
	maxOffset := n - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if l.Offset > maxOffset {
		l.Offset = maxOffset
	}
	if l.Offset < 0 {
		l.Offset = 0
	}
	from = l.Offset
	to = l.Offset + rows
	if to > n {
		to = n
	}
	return from, to
}
