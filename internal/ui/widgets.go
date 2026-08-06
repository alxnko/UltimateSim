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

// Input snapshot. Widgets render during Draw(), but Ebiten's "just pressed"
// state lives on the Update tick timeline — when FPS != TPS, Draw-time
// inpututil reads miss every click, which made every button in the game dead.
// BeginUIFrame captures the click during Update; drawn widgets consume it.
// A click no widget consumed is promoted to a world click on the NEXT tick
// (one 16ms frame of latency), so UI always has first claim.
// Keyboard input (typed runes, backspace, list navigation, Esc) and the wheel
// are captured on the same snapshot so kit widgets can consume them in Draw.
var (
	uiMX, uiMY        int  // cursor at the capturing Update tick
	uiClick           bool // one unconsumed left click (widgets eat this)
	worldMX, worldMY  int  // cursor where the surviving click happened
	worldClickPending bool // click that survived a full widget pass

	uiRunes     []rune  // characters typed this tick (text input)
	uiBackspace bool    // backspace pressed (with key repeat)
	uiUp        bool    // arrow-up pressed (with key repeat)
	uiDown      bool    // arrow-down pressed (with key repeat)
	uiEnter     bool    // enter / numpad-enter just pressed
	uiEsc       bool    // one unconsumed Esc press (widgets eat this)
	uiWheelY    float64 // wheel movement this tick (lists consume it)
	uiFrameID   uint64  // increments each BeginUIFrame; lets Draw-side
	// widgets self-update exactly once per tick even when FPS != TPS

	modalSeen    bool // a ModalFrame was drawn since the last BeginUIFrame
	modalWasSeen bool // ...and in the frame before that
)

// BeginUIFrame snapshots input state. Call once per Update tick before any
// widget-bearing Draw runs.
func BeginUIFrame() {
	modalWasSeen = modalSeen
	modalSeen = false
	if modalWasSeen {
		// A modal owned the last frame: clicks it left unclaimed die here
		// instead of being promoted to world clicks behind the modal.
		uiClick = false
	}
	worldClickPending = uiClick
	worldMX, worldMY = uiMX, uiMY
	uiMX, uiMY = ebiten.CursorPosition()
	uiClick = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	uiRunes = ebiten.AppendInputChars(uiRunes[:0])
	uiBackspace = repeatingKeyPressed(ebiten.KeyBackspace)
	uiUp = repeatingKeyPressed(ebiten.KeyArrowUp)
	uiDown = repeatingKeyPressed(ebiten.KeyArrowDown)
	uiEnter = inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
	uiEsc = inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	_, uiWheelY = ebiten.Wheel()
	uiFrameID++
	advanceHoverTip()
}

// repeatingKeyPressed reports a fresh press, then repeats after a hold
// (~0.4s delay, ~15Hz repeat at 60 TPS) — standard text-editing feel.
func repeatingKeyPressed(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	if d == 1 {
		return true
	}
	return d >= 24 && (d-24)%4 == 0
}

// TakeWorldClick returns (and consumes) a left click no UI widget claimed.
func TakeWorldClick() (int, int, bool) {
	if worldClickPending {
		worldClickPending = false
		return worldMX, worldMY, true
	}
	return 0, 0, false
}

// consumeClickIn eats the pending click if it landed inside the rect.
func consumeClickIn(x, y, w, h int) bool {
	if uiClick && PointIn(uiMX, uiMY, x, y, w, h) {
		uiClick = false
		return true
	}
	return false
}

// consumeEsc eats the pending Esc press. First consumer wins, so draw the
// topmost Esc-closable widget's frame before lower ones when stacking.
func consumeEsc() bool {
	if uiEsc {
		uiEsc = false
		return true
	}
	return false
}

// leftClicked reports a fresh, unconsumed left-button press this tick.
func leftClicked() bool {
	return uiClick
}

// ModalOpen reports whether a ModalFrame was drawn this frame or the last.
// World/HUD click paths should stand down while it is true.
func ModalOpen() bool {
	return modalSeen || modalWasSeen
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
	return consumeClickIn(x, y, w, h)
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
	w := box + 8 + MeasureText(label)
	return consumeClickIn(x, y, w, box)
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
	uiClick = false // the menu is topmost: it consumes the click either way
	idx := m.IndexAt(uiMX, uiMY)
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
