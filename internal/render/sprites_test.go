package render

// Shell Phase: Procedural Pixel Sprites E2E tests.
// Headless and deterministic: only the pure Gen*RGBA functions are exercised
// (no ebiten graphics context required). Run with:
//   go test ./internal/render/ -run TestSprites -count=2

import (
	"bytes"
	"image"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
)

// staticKindSizes lists all 20 supported static kinds and their expected
// square pixel size.
var staticKindSizes = map[string]int{
	"house":        16,
	"workshop":     16,
	"storehouse":   16,
	"shrine":       16,
	"farm":         16,
	"tavern":       16,
	"site":         16,
	"village":      24,
	"village_rich": 24,
	"capital":      24,
	"ruin":         16,
	"ship":         16,
	"caravan":      16,
	"item":         16,
	"coin":         16,
	"corpse":       16,
	"ledger":       16,
	"workbench":    16,
	"ghost_ok":     16,
	"ghost_bad":    16,
}

func testGenome() *components.GenomeComponent {
	return &components.GenomeComponent{
		Strength:  5,
		Beauty:    7, // 7 % 3 = tone 1
		Health:    5,
		Intellect: 5,
		Dominant:  0b1011, // popcount 3 -> hair color 3
	}
}

// TestSpritesCharDeterminism verifies two identical calls produce identical
// pixels over all 256 px (16x16) of the character sprite.
func TestSpritesCharDeterminism(t *testing.T) {
	a := GenCharRGBA(42, testGenome(), components.JobFarmer, 0)
	b := GenCharRGBA(42, testGenome(), components.JobFarmer, 0)
	if a == nil || b == nil {
		t.Fatal("GenCharRGBA returned nil")
	}
	if a.Bounds() != image.Rect(0, 0, 16, 16) {
		t.Fatalf("char bounds = %v, want 16x16", a.Bounds())
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("GenCharRGBA is not deterministic: identical inputs produced different pixels")
	}
}

// TestSpritesCharNilGenome verifies nil genome falls back to defaults without
// panicking and still renders visible (non-transparent) pixels.
func TestSpritesCharNilGenome(t *testing.T) {
	a := GenCharRGBA(1, nil, components.JobNone, 0)
	if a == nil {
		t.Fatal("GenCharRGBA(nil genome) returned nil")
	}
	if a.Bounds() != image.Rect(0, 0, 16, 16) {
		t.Fatalf("char bounds = %v, want 16x16", a.Bounds())
	}
	opaque := 0
	for i := 3; i < len(a.Pix); i += 4 {
		if a.Pix[i] != 0 {
			opaque++
		}
	}
	if opaque == 0 {
		t.Fatal("nil-genome char sprite is fully transparent")
	}
	b := GenCharRGBA(1, nil, components.JobNone, 0)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("GenCharRGBA(nil genome) is not deterministic")
	}
}

// TestSpritesCharJobPalette verifies clothing differs between two jobs.
func TestSpritesCharJobPalette(t *testing.T) {
	farmer := GenCharRGBA(42, testGenome(), components.JobFarmer, 0)
	guard := GenCharRGBA(42, testGenome(), components.JobGuard, 0)
	if bytes.Equal(farmer.Pix, guard.Pix) {
		t.Fatal("farmer and guard sprites are identical; job palette not applied")
	}
}

// TestSpritesCharFrameBob verifies the frame-1 walk bob shifts leg pixels.
func TestSpritesCharFrameBob(t *testing.T) {
	idle := GenCharRGBA(42, testGenome(), components.JobFarmer, 0)
	walk := GenCharRGBA(42, testGenome(), components.JobFarmer, 1)
	if bytes.Equal(idle.Pix, walk.Pix) {
		t.Fatal("frame 0 and frame 1 sprites are identical; walk bob not applied")
	}
}

// TestSpritesStaticKinds verifies all 20 kinds are non-nil, sized correctly,
// and deterministic across two calls.
func TestSpritesStaticKinds(t *testing.T) {
	if len(staticKindSizes) != 20 {
		t.Fatalf("expected 20 static kinds in test table, got %d", len(staticKindSizes))
	}
	for kind, size := range staticKindSizes {
		a := GenStaticRGBA(kind)
		if a == nil {
			t.Fatalf("GenStaticRGBA(%q) returned nil", kind)
		}
		if a.Bounds() != image.Rect(0, 0, size, size) {
			t.Fatalf("GenStaticRGBA(%q) bounds = %v, want %dx%d", kind, a.Bounds(), size, size)
		}
		b := GenStaticRGBA(kind)
		if !bytes.Equal(a.Pix, b.Pix) {
			t.Fatalf("GenStaticRGBA(%q) is not deterministic", kind)
		}
		opaque := 0
		for i := 3; i < len(a.Pix); i += 4 {
			if a.Pix[i] != 0 {
				opaque++
			}
		}
		if opaque == 0 {
			t.Fatalf("GenStaticRGBA(%q) is fully transparent", kind)
		}
	}
}

// TestSpritesStaticDistinctness verifies sprites are visually distinct
// (house vs workshop must differ pixel-for-pixel).
func TestSpritesStaticDistinctness(t *testing.T) {
	house := GenStaticRGBA("house")
	workshop := GenStaticRGBA("workshop")
	if bytes.Equal(house.Pix, workshop.Pix) {
		t.Fatal("house and workshop sprites are identical")
	}
	ok := GenStaticRGBA("ghost_ok")
	bad := GenStaticRGBA("ghost_bad")
	if bytes.Equal(ok.Pix, bad.Pix) {
		t.Fatal("ghost_ok and ghost_bad sprites are identical")
	}
}
