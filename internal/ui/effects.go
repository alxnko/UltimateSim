package ui

import (
	"fmt"
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Shell Phase: render-side combat/feedback effects. Purely visual — consumes
// InputBridge events after each tick; never touches the simulation.

type floater struct {
	WX, WY float32 // world position
	Text   string
	Clr    color.RGBA
	Life   int // frames remaining
}

type swing struct {
	WX, WY float32
	Life   int
}

// Effects holds short-lived visual feedback items.
type Effects struct {
	floaters []floater
	swings   []swing
}

// Consume converts bridge events into visual effects and notification text.
func (fx *Effects) Consume(events []systems.BridgeEvent, pc *PlayerContext, tick uint64) {
	for _, ev := range events {
		switch ev.Kind {
		case systems.EventDamage:
			fx.floaters = append(fx.floaters, floater{
				WX: ev.X, WY: ev.Y,
				Text: fmt.Sprintf("-%d", ev.Value),
				Clr:  color.RGBA{255, 80, 60, 255},
				Life: 45,
			})
		case systems.EventAttackSwing:
			fx.swings = append(fx.swings, swing{WX: ev.X, WY: ev.Y, Life: 12})
		case systems.EventExhausted:
			fx.floaters = append(fx.floaters, floater{
				WX: ev.X, WY: ev.Y,
				Text: "Exhausted!",
				Clr:  color.RGBA{200, 200, 80, 255},
				Life: 45,
			})
		case systems.EventOrderRefused:
			pc.PushNote("An order was refused — your authority weakens", tick)
		case systems.EventDeath:
			fx.floaters = append(fx.floaters, floater{
				WX: ev.X, WY: ev.Y,
				Text: "†",
				Clr:  color.RGBA{220, 220, 220, 255},
				Life: 60,
			})
		}
	}
}

// Draw renders all live effects and ages them out.
func (fx *Effects) Draw(screen *ebiten.Image, cam *Camera) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	keptF := fx.floaters[:0]
	for _, f := range fx.floaters {
		sx, sy := cam.WorldToScreen(f.WX, f.WY, sw, sh)
		rise := float64(45-f.Life) * 0.6
		DrawText(screen, f.Text, int(sx)-MeasureText(f.Text)/2, int(sy)-18-int(rise), f.Clr)
		f.Life--
		if f.Life > 0 {
			keptF = append(keptF, f)
		}
	}
	fx.floaters = keptF

	keptS := fx.swings[:0]
	for _, s := range fx.swings {
		sx, sy := cam.WorldToScreen(s.WX, s.WY, sw, sh)
		size := float64(14 - s.Life)
		ebitenutil.DrawRect(screen, sx-size/2, sy-size/2, size, 2, color.RGBA{255, 230, 150, 200})
		s.Life--
		if s.Life > 0 {
			keptS = append(keptS, s)
		}
	}
	fx.swings = keptS
}

// DrawHurtVignette flashes screen edges red when the player takes damage.
func DrawHurtVignette(screen *ebiten.Image, frames int) {
	if frames <= 0 {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	v := frames * 12
	if v > 120 {
		v = 120
	}
	a := uint8(v)
	clr := color.RGBA{180, 20, 20, a}
	ebitenutil.DrawRect(screen, 0, 0, float64(sw), 8, clr)
	ebitenutil.DrawRect(screen, 0, float64(sh-8), float64(sw), 8, clr)
	ebitenutil.DrawRect(screen, 0, 0, 8, float64(sh), clr)
	ebitenutil.DrawRect(screen, float64(sw-8), 0, 8, float64(sh), clr)
}
