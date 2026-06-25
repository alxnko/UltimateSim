package render

// Shell Phase: Procedural Pixel Sprites
// All sprite generation happens into std-lib *image.RGBA via pure, deterministic
// functions (zero RNG, headless-testable). Conversion to *ebiten.Image happens
// ONLY inside the SpriteCache getters, which are called from the running game.

import (
	"image"
	"image/color"
	"math/bits"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ALXNKO/UltimateSim/internal/components"
)

// jobPalette maps JobID -> clothing color (15 entries, medieval dyes).
var jobPalette = [15]color.RGBA{
	components.JobNone:        {128, 128, 128, 255}, // jobless gray
	components.JobFarmer:      {58, 132, 56, 255},   // field green
	components.JobLumberjack:  {110, 78, 38, 255},   // wood brown
	components.JobArtisan:     {142, 88, 160, 255},  // dyed purple
	components.JobGuard:       {70, 100, 140, 255},  // steel blue
	components.JobPreacher:    {235, 235, 230, 255}, // white robes
	components.JobCaster:      {90, 60, 170, 255},   // arcane violet
	components.JobBandit:      {120, 30, 30, 255},   // dark red
	components.JobSailor:      {40, 90, 150, 255},   // sea blue
	components.JobCaptain:     {30, 60, 110, 255},   // navy coat
	components.JobPenalLabor:  {150, 140, 100, 255}, // drab tan
	components.JobMercenary:   {60, 60, 65, 255},    // dark iron
	components.JobBuilder:     {210, 130, 40, 255},  // builder orange
	components.JobGravedigger: {70, 60, 50, 255},    // grave soil
	components.JobDoctor:      {200, 215, 200, 255}, // pale mint
}

// skinTones indexed by genome.Beauty % 3 (nil genome -> tone 0).
var skinTones = [3]color.RGBA{
	{224, 172, 124, 255},
	{184, 126, 86, 255},
	{120, 80, 50, 255},
}

// hairColors indexed by bits.OnesCount32(genome.Dominant) % 4.
var hairColors = [4]color.RGBA{
	{30, 25, 20, 255},   // black
	{92, 58, 28, 255},   // brown
	{200, 168, 80, 255}, // blonde
	{150, 60, 30, 255},  // red
}

var eyeColor = color.RGBA{20, 20, 25, 255}

// Static medieval palette shared by GenStaticRGBA kinds.
var (
	colWallBrown   = color.RGBA{124, 82, 46, 255}
	colWallDark    = color.RGBA{86, 56, 30, 255}
	colRoofRed     = color.RGBA{128, 36, 32, 255}
	colStoneGray   = color.RGBA{142, 142, 148, 255}
	colStoneDark   = color.RGBA{96, 96, 102, 255}
	colTan         = color.RGBA{196, 164, 110, 255}
	colTanDark     = color.RGBA{150, 120, 76, 255}
	colGold        = color.RGBA{226, 184, 56, 255}
	colGoldDark    = color.RGBA{170, 130, 30, 255}
	colSoil        = color.RGBA{110, 76, 44, 255}
	colSoilDark    = color.RGBA{82, 54, 30, 255}
	colSprout      = color.RGBA{84, 160, 60, 255}
	colWindowLit   = color.RGBA{240, 200, 70, 255}
	colSailWhite   = color.RGBA{236, 232, 220, 255}
	colPale        = color.RGBA{210, 208, 200, 255}
	colBannerBlue  = color.RGBA{50, 80, 150, 255}
	colFlagRed     = color.RGBA{190, 40, 36, 255}
	colAnvilOrange = color.RGBA{214, 120, 36, 255}
	colToolGray    = color.RGBA{120, 124, 130, 255}
	colGhostOK     = color.RGBA{0, 128, 0, 128}   // premultiplied 50% alpha green
	colGhostBad    = color.RGBA{128, 0, 0, 128}   // premultiplied 50% alpha red
	colMissing     = color.RGBA{255, 0, 255, 255} // magenta placeholder for unknown kinds
)

// fillRect fills the inclusive pixel rectangle [x0,x1] x [y0,y1].
// image.RGBA.SetRGBA bounds-checks internally, so clipping is implicit.
func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// GenCharRGBA generates a 16x16 character sprite. Deterministic from inputs
// ONLY (no RNG): clothing from jobPalette[jobID], skin from Beauty, hair from
// the Dominant gene popcount, with a 2-frame walk bob on the legs.
// A nil genome falls back to skin tone 0 / hair color 0.
func GenCharRGBA(id uint64, genome *components.GenomeComponent, jobID uint8, frame int) *image.RGBA {
	_ = id // id participates in cache keys only; pixels derive from genome/job/frame.
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))

	clothes := jobPalette[components.JobNone]
	if int(jobID) < len(jobPalette) {
		clothes = jobPalette[jobID]
	}
	skin := skinTones[0]
	hair := hairColors[0]
	if genome != nil {
		skin = skinTones[int(genome.Beauty)%len(skinTones)]
		hair = hairColors[bits.OnesCount32(genome.Dominant)%len(hairColors)]
	}

	// Clothing: 6x8 rect rows 7-14 cols 5-10. Torso rows 7-12 are static;
	// leg rows 13-14 are drawn per-frame below for the walk bob.
	fillRect(img, 5, 7, 10, 12, clothes)
	if frame&1 == 0 {
		// Frame 0: both legs at rest on rows 13-14.
		fillRect(img, 5, 13, 7, 14, clothes)
		fillRect(img, 8, 13, 10, 14, clothes)
	} else {
		// Frame 1: walk bob, legs shift 1px alternating (left leg up, right leg down).
		fillRect(img, 5, 12, 7, 13, clothes)
		fillRect(img, 8, 14, 10, 15, clothes)
	}

	// Skin head: 6x5 rows 2-6 cols 5-10.
	fillRect(img, 5, 2, 10, 6, skin)

	// Hair: 6x2 rows 1-2 cols 5-10 (covers the scalp row of the head).
	fillRect(img, 5, 1, 10, 2, hair)

	// Eyes: 2 dark pixels on row 4.
	img.SetRGBA(6, 4, eyeColor)
	img.SetRGBA(9, 4, eyeColor)

	return img
}

// GenStaticRGBA generates a static world sprite by kind. Sizes: "village",
// "village_rich" and "capital" are 24x24; all other kinds are 16x16.
// Unknown kinds return a magenta 16x16 placeholder (never nil).
func GenStaticRGBA(kind string) *image.RGBA {
	switch kind {
	case "village", "village_rich", "capital":
		img := image.NewRGBA(image.Rect(0, 0, 24, 24))
		switch kind {
		case "village":
			genSettlement(img, colWallBrown, colWallDark, colBannerBlue)
		case "village_rich":
			genSettlement(img, colStoneGray, colStoneDark, colGold)
		case "capital":
			genCapital(img)
		}
		return img
	}

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	switch kind {
	case "house":
		genHouse(img)
	case "workshop":
		genWorkshop(img)
	case "storehouse":
		genStorehouse(img)
	case "shrine":
		genShrine(img)
	case "farm":
		genFarm(img)
	case "tavern":
		genTavern(img)
	case "site":
		genSite(img)
	case "ruin":
		genRuin(img)
	case "ship":
		genShip(img)
	case "caravan":
		genCaravan(img)
	case "item":
		genItem(img)
	case "coin":
		genCoin(img)
	case "corpse":
		genCorpse(img)
	case "ledger":
		genLedger(img)
	case "workbench":
		genWorkbench(img)
	case "ghost_ok":
		genGhostBox(img, colGhostOK)
	case "ghost_bad":
		genGhostBox(img, colGhostBad)
	default:
		fillRect(img, 0, 0, 15, 15, colMissing)
	}
	return img
}

// genHouse: brown walls with a dark red stacked triangle roof.
func genHouse(img *image.RGBA) {
	fillRect(img, 7, 3, 8, 3, colRoofRed)
	fillRect(img, 6, 4, 9, 4, colRoofRed)
	fillRect(img, 5, 5, 10, 5, colRoofRed)
	fillRect(img, 4, 6, 11, 6, colRoofRed)
	fillRect(img, 3, 7, 12, 7, colRoofRed)
	fillRect(img, 3, 8, 12, 14, colWallBrown)
	fillRect(img, 7, 11, 8, 14, colWallDark) // door
}

// genWorkshop: gray walls with a flat roof and an orange anvil block.
func genWorkshop(img *image.RGBA) {
	fillRect(img, 2, 5, 13, 6, colStoneDark)  // flat roof
	fillRect(img, 3, 7, 12, 14, colStoneGray) // walls
	fillRect(img, 5, 10, 10, 10, colAnvilOrange)
	fillRect(img, 7, 11, 8, 12, colAnvilOrange) // anvil block
}

// genStorehouse: wide tan barn with dark crossed front beams.
func genStorehouse(img *image.RGBA) {
	fillRect(img, 0, 4, 15, 5, colTanDark) // roof
	fillRect(img, 1, 6, 14, 14, colTan)    // wide barn body
	fillRect(img, 1, 6, 1, 14, colWallDark)
	fillRect(img, 14, 6, 14, 14, colWallDark)
	for i := 0; i <= 6; i++ { // cross beams
		img.SetRGBA(3+i, 7+i, colWallDark)
		img.SetRGBA(12-i, 7+i, colWallDark)
	}
}

// genShrine: light gray obelisk with a gold tip.
func genShrine(img *image.RGBA) {
	fillRect(img, 5, 13, 10, 14, colStoneGray) // base
	fillRect(img, 6, 12, 9, 12, colStoneGray)  // step
	fillRect(img, 7, 3, 8, 12, colStoneGray)   // obelisk column
	fillRect(img, 7, 1, 8, 2, colGold)         // gold tip
}

// genFarm: tilled brown rows with green sprouts.
func genFarm(img *image.RGBA) {
	fillRect(img, 1, 3, 14, 14, colSoil)
	for _, y := range [4]int{4, 7, 10, 13} { // dark furrows
		fillRect(img, 1, y, 14, y, colSoilDark)
	}
	for _, y := range [4]int{3, 6, 9, 12} { // sprouts
		for _, x := range [4]int{3, 6, 9, 12} {
			img.SetRGBA(x, y, colSprout)
		}
	}
}

// genTavern: brown building with yellow lit windows.
func genTavern(img *image.RGBA) {
	fillRect(img, 1, 4, 14, 5, colWallDark) // dark roof
	fillRect(img, 2, 6, 13, 14, colWallBrown)
	fillRect(img, 4, 9, 5, 10, colWindowLit)   // lit window left
	fillRect(img, 10, 9, 11, 10, colWindowLit) // lit window right
	fillRect(img, 7, 10, 8, 14, colWallDark)   // door
}

// genSite: brown scaffold lattice on transparent background.
func genSite(img *image.RGBA) {
	fillRect(img, 3, 4, 3, 14, colWallBrown)   // left post
	fillRect(img, 12, 4, 12, 14, colWallBrown) // right post
	fillRect(img, 3, 5, 12, 5, colWallBrown)   // beams
	fillRect(img, 3, 9, 12, 9, colWallBrown)
	fillRect(img, 3, 13, 12, 13, colWallBrown)
	for i := 0; i <= 2; i++ { // diagonal braces
		img.SetRGBA(4+i, 6+i, colWallDark)
		img.SetRGBA(11-i, 10+i, colWallDark)
	}
}

// genSettlement: 24x24 hall with a banner; wall/door/banner colors vary
// between "village" (brown + blue) and "village_rich" (stone gray + gold).
func genSettlement(img *image.RGBA, wall, dark, banner color.RGBA) {
	fillRect(img, 10, 6, 13, 6, colRoofRed) // stacked roof
	fillRect(img, 8, 7, 15, 7, colRoofRed)
	fillRect(img, 6, 8, 17, 8, colRoofRed)
	fillRect(img, 4, 9, 19, 9, colRoofRed)
	fillRect(img, 3, 10, 20, 10, colRoofRed)
	fillRect(img, 3, 11, 20, 22, wall)  // hall walls
	fillRect(img, 10, 17, 13, 22, dark) // door
	fillRect(img, 6, 14, 7, 15, dark)   // windows
	fillRect(img, 16, 14, 17, 15, dark)
	fillRect(img, 21, 2, 21, 10, colWallDark) // banner pole
	fillRect(img, 19, 2, 20, 5, banner)       // banner cloth
}

// genCapital: 24x24 stone keep with a crenellated top row and a red flag.
func genCapital(img *image.RGBA) {
	fillRect(img, 4, 6, 19, 22, colStoneGray) // keep body
	for x := 4; x <= 19; x++ {                // crenellation top row
		if ((x-4)/2)%2 == 0 {
			img.SetRGBA(x, 5, colStoneGray)
		}
	}
	fillRect(img, 11, 1, 11, 5, colWallDark)    // flag pole
	fillRect(img, 12, 1, 14, 2, colFlagRed)     // red flag
	fillRect(img, 10, 16, 13, 22, colStoneDark) // gate
	fillRect(img, 7, 10, 8, 11, colStoneDark)   // windows
	fillRect(img, 15, 10, 16, 11, colStoneDark)
}

// genRuin: broken gray rubble and wall stubs.
func genRuin(img *image.RGBA) {
	fillRect(img, 1, 14, 14, 14, colStoneDark) // rubble base
	fillRect(img, 2, 13, 5, 13, colStoneGray)
	fillRect(img, 8, 13, 12, 13, colStoneGray)
	fillRect(img, 3, 8, 5, 12, colStoneGray) // left wall stub
	img.SetRGBA(3, 7, colStoneGray)
	fillRect(img, 10, 10, 12, 12, colStoneGray) // right wall stub
	img.SetRGBA(12, 9, colStoneGray)
	img.SetRGBA(7, 12, colStoneGray) // scattered debris
	img.SetRGBA(6, 13, colStoneDark)
}

// genShip: brown hull with a white sail.
func genShip(img *image.RGBA) {
	fillRect(img, 2, 11, 13, 12, colWallBrown) // hull
	fillRect(img, 4, 13, 11, 13, colWallBrown)
	fillRect(img, 5, 14, 10, 14, colWallDark) // keel
	fillRect(img, 7, 3, 7, 10, colWallDark)   // mast
	fillRect(img, 8, 4, 13, 9, colSailWhite)  // sail
}

// genCaravan: brown cart with wheels and a tan canopy.
func genCaravan(img *image.RGBA) {
	fillRect(img, 4, 5, 11, 7, colTan)        // canopy
	fillRect(img, 3, 8, 12, 11, colWallBrown) // cart body
	fillRect(img, 4, 12, 5, 13, colWallDark)  // wheels
	fillRect(img, 10, 12, 11, 13, colWallDark)
}

// genItem: gold sparkle diamond.
func genItem(img *image.RGBA) {
	fillRect(img, 7, 4, 8, 4, colGold)
	fillRect(img, 6, 5, 9, 5, colGold)
	fillRect(img, 5, 6, 10, 6, colGold)
	fillRect(img, 4, 7, 11, 8, colGold)
	fillRect(img, 5, 9, 10, 9, colGold)
	fillRect(img, 6, 10, 9, 10, colGold)
	fillRect(img, 7, 11, 8, 11, colGold)
	img.SetRGBA(6, 6, colSailWhite) // sparkle
	img.SetRGBA(9, 9, colSailWhite)
}

// genCoin: gold circle-ish disc with a darker stamped center.
func genCoin(img *image.RGBA) {
	fillRect(img, 6, 4, 9, 4, colGold)
	fillRect(img, 5, 5, 10, 5, colGold)
	fillRect(img, 4, 6, 11, 9, colGold)
	fillRect(img, 5, 10, 10, 10, colGold)
	fillRect(img, 6, 11, 9, 11, colGold)
	fillRect(img, 6, 6, 9, 9, colGoldDark) // stamped center
	img.SetRGBA(6, 5, colSailWhite)        // highlight
}

// genCorpse: gray-white figure lying horizontally.
func genCorpse(img *image.RGBA) {
	fillRect(img, 1, 9, 9, 11, colPale)       // body
	fillRect(img, 10, 8, 13, 11, colPale)     // head
	fillRect(img, 3, 12, 7, 12, colStoneDark) // shadow arm
}

// genLedger: brown book with white pages.
func genLedger(img *image.RGBA) {
	fillRect(img, 3, 4, 12, 12, colWallDark)  // cover
	fillRect(img, 5, 5, 11, 11, colSailWhite) // pages
	fillRect(img, 6, 6, 10, 6, colStoneDark)  // text lines
	fillRect(img, 6, 8, 10, 8, colStoneDark)
	fillRect(img, 6, 10, 10, 10, colStoneDark)
}

// genWorkbench: brown table with a gray tool on top.
func genWorkbench(img *image.RGBA) {
	fillRect(img, 2, 7, 13, 8, colWallBrown) // tabletop
	fillRect(img, 3, 9, 4, 13, colWallDark)  // legs
	fillRect(img, 11, 9, 12, 13, colWallDark)
	fillRect(img, 5, 5, 7, 6, colToolGray) // tool
}

// genGhostBox: translucent 50% alpha box outline (placement ghost).
func genGhostBox(img *image.RGBA, c color.RGBA) {
	fillRect(img, 1, 1, 14, 1, c)
	fillRect(img, 1, 14, 14, 14, c)
	fillRect(img, 1, 2, 1, 13, c)
	fillRect(img, 14, 2, 14, 13, c)
}

// charKey mixes id + job + frame parity for the character sprite cache.
type charKey struct {
	id    uint64
	jobID uint8
	frame uint8
}

// SpriteCache lazily converts generated *image.RGBA sprites into GPU-side
// *ebiten.Image textures and memoizes them. Must only be used from the
// running game (ebiten context); tests exercise the Gen*RGBA functions.
type SpriteCache struct {
	chars   map[charKey]*ebiten.Image
	statics map[string]*ebiten.Image
}

// NewSpriteCache allocates an empty sprite cache.
func NewSpriteCache() *SpriteCache {
	return &SpriteCache{
		chars:   make(map[charKey]*ebiten.Image),
		statics: make(map[string]*ebiten.Image),
	}
}

// Char returns the cached character texture for id+job+frame, generating it
// on first use. Frame is normalized to its parity since the walk cycle has
// exactly 2 frames, keeping the cache bounded.
func (s *SpriteCache) Char(id uint64, genome *components.GenomeComponent, jobID uint8, frame int) *ebiten.Image {
	key := charKey{id: id, jobID: jobID, frame: uint8(frame & 1)}
	if img, ok := s.chars[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(GenCharRGBA(id, genome, jobID, frame))
	s.chars[key] = img
	return img
}

// Static returns the cached static texture for kind, generating it on first use.
func (s *SpriteCache) Static(kind string) *ebiten.Image {
	if img, ok := s.statics[kind]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(GenStaticRGBA(kind))
	s.statics[kind] = img
	return img
}
