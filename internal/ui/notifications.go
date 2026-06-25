package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Shell Phase: transient on-screen notifications stacked top-right under the
// minimap. The event log (player_context.Log) keeps the long history.

const (
	noteLifetime = 600 // ticks a notification stays visible
	noteMax      = 8
)

// Note is one timestamped message.
type Note struct {
	Text string
	Tick uint64
}

// Notifier holds recent transient messages.
type Notifier struct {
	items []Note
}

// Push appends a message, trimming history to the cap.
func (n *Notifier) Push(text string, tick uint64) {
	n.items = append(n.items, Note{Text: text, Tick: tick})
	if len(n.items) > 64 {
		n.items = n.items[len(n.items)-64:]
	}
}

// Active returns currently-visible notes, newest first, capped at noteMax.
func (n *Notifier) Active(tick uint64) []Note {
	var out []Note
	for i := len(n.items) - 1; i >= 0 && len(out) < noteMax; i-- {
		if tick-n.items[i].Tick <= noteLifetime {
			out = append(out, n.items[i])
		}
	}
	return out
}

// Draw renders the active notifications below the minimap.
func (n *Notifier) Draw(dst *ebiten.Image, tick uint64, sw int) {
	active := n.Active(tick)
	y := MinimapSize + 20
	x := sw - MinimapSize - 8
	for _, note := range active {
		DrawText(dst, note.Text, x, y, TextCol)
		y += 16
	}
}
