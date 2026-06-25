package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// Shell Phase: shared visual theme — RimWorld-ish dark panels, gold accents —
// plus thin text helpers over ebiten/text/v2 wrapping the stdlib bitmap font.

var (
	PanelBG     = color.RGBA{25, 28, 36, 235}
	PanelBorder = color.RGBA{90, 95, 110, 255}
	TextCol     = color.RGBA{220, 220, 225, 255}
	TextDim     = color.RGBA{150, 155, 165, 255}
	AccentCol   = color.RGBA{230, 180, 60, 255}
	BarRed      = color.RGBA{200, 70, 60, 255}
	BarGreen    = color.RGBA{90, 175, 80, 255}
	BarBlue     = color.RGBA{70, 130, 200, 255}
	BarPurple   = color.RGBA{160, 90, 200, 255}
	BarGray     = color.RGBA{55, 60, 70, 255}
	ButtonBG    = color.RGBA{45, 50, 62, 255}
	ButtonHover = color.RGBA{70, 78, 95, 255}
)

// fontFace is the text/v2 face wrapping the 7x13 stdlib bitmap font.
var fontFace = text.NewGoXFace(basicfont.Face7x13)

const charWidth = 7 // basicfont.Face7x13 advance width

// DrawText draws a string with its top-left at (x, y).
func DrawText(dst *ebiten.Image, s string, x, y int, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, fontFace, op)
}

// MeasureText returns the pixel width of a single-line string.
func MeasureText(s string) int {
	return len(s) * charWidth
}
