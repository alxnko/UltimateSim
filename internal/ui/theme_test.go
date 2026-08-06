package ui_test

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestMeasureTextRealAdvance(t *testing.T) {
	if got := ui.MeasureText(""); got != 0 {
		t.Errorf("MeasureText(\"\") = %d, want 0", got)
	}
	hello := ui.MeasureText("Hello")
	if hello <= 0 {
		t.Fatalf("MeasureText(\"Hello\") = %d, want > 0", hello)
	}
	longer := ui.MeasureText("Hello, sovereign world")
	if longer <= hello {
		t.Errorf("longer string measured %d, not wider than %d", longer, hello)
	}
	// Proportional font: 'W' must be wider than 'i' — proves we are not on
	// the old fixed-advance bitmap path.
	if wi, ii := ui.MeasureText("WWWW"), ui.MeasureText("iiii"); wi <= ii {
		t.Errorf("MeasureText(\"WWWW\") = %d not wider than \"iiii\" = %d; advance looks fixed-width", wi, ii)
	}
}

func TestMeasureAtScalesWithSize(t *testing.T) {
	const s = "Boundless Sovereigns"
	small := ui.MeasureAt(s, ui.SmallSize)
	body := ui.MeasureAt(s, ui.BodySize)
	title := ui.MeasureAt(s, ui.TitleSize)
	if !(small < body && body < title) {
		t.Errorf("widths not monotonic in size: small=%d body=%d title=%d", small, body, title)
	}
	if got := ui.MeasureAt(s, ui.BodySize); got != ui.MeasureText(s) {
		t.Errorf("MeasureAt(body) = %d != MeasureText = %d", got, ui.MeasureText(s))
	}
}

func TestMeasureTextDeterministic(t *testing.T) {
	const s = "Gold: 1234"
	first := ui.MeasureText(s)
	for i := 0; i < 100; i++ {
		if got := ui.MeasureText(s); got != first {
			t.Fatalf("MeasureText varied: %d then %d", first, got)
		}
	}
}

func TestDrawTiersSmoke(t *testing.T) {
	dst := ebiten.NewImage(320, 120)
	ui.DrawTitle(dst, "Title 22px", 4, 4, ui.AccentCol)
	ui.DrawHeading(dst, "Heading 16px", 4, 32, ui.TextCol)
	ui.DrawText(dst, "Body 13px", 4, 54, ui.TextCol)
	ui.DrawSmall(dst, "Small 11px", 4, 72, ui.TextDim)
}
