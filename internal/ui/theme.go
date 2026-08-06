package ui

import (
	"bytes"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// Shell Phase: shared visual theme — RimWorld-ish dark panels, gold accents —
// plus text helpers over ebiten/text/v2 rendering the embedded Go Regular TTF
// at crisp scalable sizes (title/heading/body/small tiers).

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

// Font size tiers in pixels.
const (
	TitleSize   = 22
	HeadingSize = 16
	BodySize    = 13
	SmallSize   = 11
)

var (
	fontOnce   sync.Once
	fontSource *text.GoTextFaceSource
)

// fontSrc lazily parses the embedded Go Regular TTF exactly once.
func fontSrc() *text.GoTextFaceSource {
	fontOnce.Do(func() {
		src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		if err != nil {
			// The TTF is embedded in the binary; failure to parse it is a
			// build defect, not a runtime condition.
			panic("ui: parse embedded goregular TTF: " + err.Error())
		}
		fontSource = src
	})
	return fontSource
}

// faceAt returns a face for an arbitrary pixel size. GoTextFace is a small
// value wrapper; glyph caching lives in the shared GoTextFaceSource.
func faceAt(size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: fontSrc(), Size: size}
}

// drawAt draws s with its top-left at integer pixel (x, y) using default
// LayoutOptions (left-aligned, first line's top at y).
func drawAt(dst *ebiten.Image, s string, x, y int, clr color.Color, size float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, faceAt(size), op)
}

// DrawTitle draws s at the title tier (~22px), top-left at (x, y).
func DrawTitle(dst *ebiten.Image, s string, x, y int, clr color.Color) {
	drawAt(dst, s, x, y, clr, TitleSize)
}

// DrawHeading draws s at the heading tier (~16px), top-left at (x, y).
func DrawHeading(dst *ebiten.Image, s string, x, y int, clr color.Color) {
	drawAt(dst, s, x, y, clr, HeadingSize)
}

// DrawText draws a string at body size with its top-left at (x, y).
func DrawText(dst *ebiten.Image, s string, x, y int, clr color.Color) {
	drawAt(dst, s, x, y, clr, BodySize)
}

// DrawSmall draws s at the small tier (~11px), top-left at (x, y).
func DrawSmall(dst *ebiten.Image, s string, x, y int, clr color.Color) {
	drawAt(dst, s, x, y, clr, SmallSize)
}

// MeasureText returns the pixel advance width of a single-line string at
// body size — the width DrawText will occupy.
func MeasureText(s string) int {
	return MeasureAt(s, BodySize)
}

// MeasureAt returns the pixel advance width of a single-line string at the
// given face size, rounded up so layouts never clip the final glyph.
func MeasureAt(s string, size float64) int {
	return int(math.Ceil(text.Advance(s, faceAt(size))))
}
