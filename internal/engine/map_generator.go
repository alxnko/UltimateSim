package engine

import (
	smath "math"
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/pkg/math"
)

// Phase 02.2: Procedural Generation Pipeline
// Worldgen v2 (spec P1.1-P1.4): planet-scale macro structure.
// The map is planned as 3-5 continents distributed across two "bands" separated
// by a guaranteed ocean channel (so at least two landmasses can never merge),
// with island chains in open ocean, lakes carved into continental interiors,
// mountain ranges from ridged noise, and latitude-driven climate belts.

const (
	// genDeepC: continent-support field value below which a tile is abyssal ocean.
	genDeepC float32 = 0.02
	// genLandRange: fixed raw-height span mapped onto land elevations [85..255].
	// A fixed scale keeps mountain presence independent of any global outlier.
	genLandRange float32 = 1.10
	// genDeepDrop: depth subtracted from abyssal ocean so detail noise can never
	// breach sea level in open water (prevents the "noise blob" speckle look).
	genDeepDrop float32 = 0.25
	// genDetailWeight: additive detail noise (coast wobble, ocean floor texture).
	genDetailWeight float32 = 0.12
	// genReliefWeight: multiplicative detail relief on continental support.
	genReliefWeight float32 = 0.30
	// genPlateau: base raw height of continental interiors; kept low so the
	// interior reads as rolling terrain and only ridges reach mountain bands.
	genPlateau float32 = 0.55
	// genRidgeWeight: ridged mountain-range contribution on continent interiors.
	genRidgeWeight float32 = 0.65
	// genSeaMargin: ocean fraction carved into continental fringes beyond the abyss.
	genSeaMargin float32 = 0.08
	// genIslandHeight: raw elevation of island-chain bumps (must clear genDeepDrop).
	genIslandHeight float32 = 0.65
	// genLakeElev / genLakeRimElev: carved lake water level and forced shore level.
	genLakeElev    uint8 = 70
	genLakeRimElev uint8 = 96

	genDetailFreq float32 = 1.0 / 96.0
	genRidgeFreq  float32 = 1.0 / 170.0
	genMoistFreq  float32 = 1.0 / 140.0
	genTempFreq   float32 = 1.0 / 180.0
)

// genBlob is an axis-aligned falloff bump used for continents, lobes, islands
// and lake sites. rx/ry are the support radii in tiles; shape blends the
// falloff profile from ellipse (0) to squircle (1) for silhouette variety.
type genBlob struct {
	x, y, rx, ry, shape float32
}

// planetPlan is the macro layout drawn from the seeded RNG before any
// per-tile work happens. All slices iterate in deterministic order.
type planetPlan struct {
	lobes    []genBlob   // continental support lobes (union forms the landmasses)
	cores    []genBlob   // one main blob per placed continent (lake anchors)
	islands  []genBlob   // small oceanic bumps forming island chains
	lakeSets [][]genBlob // lake site candidates, grouped per continent
}

// GenerateMap procedurally generates the terrain data for the given MapGrid.
// It iterates sequentially over the 1D contiguous array `grid.Tiles` to maximize CPU L1/L2 cache hits.
// It uses the provided seed to initialize distinct deterministic noise generators for each layer.
func GenerateMap(grid *MapGrid, seed [32]byte) {
	// Local PRNG drives macro planning first, then per-tile resources, in a fixed order.
	localRNG := rand.New(rand.NewChaCha8(seed))

	w, h := grid.Width, grid.Height
	n := w * h
	md := float32(min(w, h))

	plan := planPlanet(w, h, localRNG)

	// Distinct deterministic noise generators per layer, derived from the base seed.
	elevSeed := seed
	elevSeed[0] ^= 0x01
	detailNoise := math.NewPerlin(elevSeed)

	moistSeed := seed
	moistSeed[0] ^= 0x02
	moistNoise := math.NewPerlin(moistSeed)

	tempSeed := seed
	tempSeed[0] ^= 0x03
	tempNoise := math.NewPerlin(tempSeed)

	ridgeSeed := seed
	ridgeSeed[0] ^= 0x04
	ridgeNoise := math.NewPerlin(ridgeSeed)

	warpXSeed := seed
	warpXSeed[0] ^= 0x05
	warpXNoise := math.NewPerlin(warpXSeed)

	warpYSeed := seed
	warpYSeed[0] ^= 0x06
	warpYNoise := math.NewPerlin(warpYSeed)

	warpAmp := md * 0.055
	warpFreq := 2.5 / md

	// Pass 1: raw floating-point elevation field.
	raw := make([]float32, n)
	deepCount := 0
	rawMin := float32(smath.MaxFloat32)
	rawMax := float32(-smath.MaxFloat32)

	for y := 0; y < h; y++ {
		fy := float32(y)
		for x := 0; x < w; x++ {
			fx := float32(x)
			i := y*w + x

			// Domain warp bends coastlines into bays and peninsulas.
			wx := fx + warpAmp*fbm(warpXNoise, fx*warpFreq, fy*warpFreq, 3)
			wy := fy + warpAmp*fbm(warpYNoise, fx*warpFreq, fy*warpFreq, 3)

			// Continental support field: max falloff over all lobes, each
			// blending ellipse and squircle profiles for silhouette variety.
			var c float32
			for _, b := range plan.lobes {
				dx := (wx - b.x) / b.rx
				dy := (wy - b.y) / b.ry
				dx2 := dx * dx
				dy2 := dy * dy
				d2 := dx2 + dy2
				d4 := dx2*dx2 + dy2*dy2
				dd := b.shape*d4 + (1-b.shape)*d2
				if dd < 1 {
					f := 1 - dd
					f *= f
					if f > c {
						c = f
					}
				}
			}

			detail := fbm(detailNoise, fx*genDetailFreq, fy*genDetailFreq, 4)
			// sqrt(c) compresses the interior into rolling mid-elevations
			// (mountains come from ridges, not the continental plateau) while
			// steepening coastal gradients.
			base := genPlateau * float32(smath.Sqrt(float64(c)))
			v := base*(1+genReliefWeight*detail) + genDetailWeight*detail
			if c < genDeepC {
				// Abyssal floor sits strictly below any coastal fringe value.
				v -= genDeepDrop * (genDeepC - c) / genDeepC
				deepCount++
			} else if c > 0.30 {
				// Ridged noise forms mountain ranges on continent interiors.
				rn := fbm(ridgeNoise, fx*genRidgeFreq, fy*genRidgeFreq, 3)
				ridge := 1 - 2*absf(rn)
				if ridge > 0 {
					mask := (c - 0.30) / 0.70
					v += ridge * ridge * mask * genRidgeWeight
				}
			}

			raw[i] = v
			if v < rawMin {
				rawMin = v
			}
			if v > rawMax {
				rawMax = v
			}
		}
	}

	// Pass 2: island chain bumps (bounded-box additive pass over open ocean).
	for _, b := range plan.islands {
		x0 := max(0, int(b.x-b.rx))
		x1 := min(w-1, int(b.x+b.rx))
		y0 := max(0, int(b.y-b.ry))
		y1 := min(h-1, int(b.y+b.ry))
		for yy := y0; yy <= y1; yy++ {
			for xx := x0; xx <= x1; xx++ {
				dx := (float32(xx) - b.x) / b.rx
				dy := (float32(yy) - b.y) / b.ry
				d2 := dx*dx + dy*dy
				if d2 >= 1 {
					continue
				}
				f := 1 - d2
				i := yy*w + xx
				raw[i] += f * f * genIslandHeight
				if raw[i] > rawMax {
					rawMax = raw[i]
				}
			}
		}
	}

	// Pass 3: sea level via quantile so the ocean fraction is controlled per seed.
	// The abyss must stay fully submerged, so the target can never dip below it.
	oceanFrac := float32(deepCount)/float32(n) + genSeaMargin
	if oceanFrac < 0.30 {
		oceanFrac = 0.30
	}
	if oceanFrac > 0.55 {
		oceanFrac = 0.55
	}
	if minFrac := float32(deepCount)/float32(n) + 0.03; oceanFrac < minFrac {
		oceanFrac = minFrac
	}
	if oceanFrac > 0.97 {
		oceanFrac = 0.97
	}
	seaLevel := quantileThreshold(raw, rawMin, rawMax, oceanFrac)

	for i, v := range raw {
		var e float32
		if v < seaLevel {
			if seaLevel > rawMin {
				e = 84 * (v - rawMin) / (seaLevel - rawMin)
			}
		} else {
			t := (v - seaLevel) / genLandRange
			if t > 1 {
				t = 1
			}
			// t^0.75 narrows beach bands and widens highlands.
			s := float32(smath.Sqrt(float64(t)))
			e = 85 + 170*s*float32(smath.Sqrt(float64(s)))
		}
		if e < 0 {
			e = 0
		} else if e > 255 {
			e = 255
		}
		grid.Tiles[i].Elevation = uint8(e)
	}

	// Pass 4: carve lakes into continental interiors.
	carveLakes(grid, &plan)

	// Pass 5: climate, biomes and resources (sequential; RNG order is fixed).
	for y := 0; y < h; y++ {
		// Latitude: 1.0 at the equator (Height/2), 0.0 at the poles.
		distToEquator := float32(y) - float32(h)/2.0
		if distToEquator < 0 {
			distToEquator = -distToEquator
		}
		latBase := 1.0 - distToEquator/(float32(h)/2.0)
		latMoist := latitudeMoisture(latBase)

		for x := 0; x < w; x++ {
			i := y*w + x
			e := grid.Tiles[i].Elevation

			// Moisture: noise blended with latitude belts (wet equator,
			// dry subtropics, damp temperate zone, dry poles), plus an
			// orographic bonus so high ridges catch rain.
			mv := 0.5 + 0.8*fbm(moistNoise, float32(x)*genMoistFreq, float32(y)*genMoistFreq, 3)
			m := clamp01(mv)*0.65 + latMoist*0.35
			if e > 150 {
				m += float32(e-150) / 105.0 * 0.15
			}
			m = clamp01(m)
			grid.Tiles[i].Moisture = uint8(m * 255.0)

			// Temperature: latitude belts plus noise, mildly cooled by altitude.
			tn := clamp01(0.5 + 0.8*fbm(tempNoise, float32(x)*genTempFreq, float32(y)*genTempFreq, 2))
			t := latBase*0.72 + tn*0.28
			if e > 95 {
				t -= float32(e-95) / 160.0 * 0.12
			}
			t = clamp01(t)
			grid.Tiles[i].Temperature = uint8(t * 255.0)

			grid.Tiles[i].BiomeID = DetermineBiome(e, grid.Tiles[i].Moisture, grid.Tiles[i].Temperature)

			// Phase 02.4: Static Resource Depots
			// Initialize resources based on BiomeID to populate parallel array.
			// This preserves cache lines and handles deterministic RNG inside the loop.
			switch grid.Tiles[i].BiomeID {
			case BiomeTemperateDeciduousForest, BiomeTemperateRainForest, BiomeTropicalSeasonalForest, BiomeTropicalRainForest:
				// Base wood value (e.g., 50-150)
				grid.Resources[i].WoodValue = uint8(50 + localRNG.IntN(101))
				// Base food value for forests (e.g., 20-60)
				grid.Resources[i].FoodValue = uint8(20 + localRNG.IntN(41))
			case BiomeGrassland:
				// Higher base food value for grasslands (e.g., 40-100)
				grid.Resources[i].FoodValue = uint8(40 + localRNG.IntN(61))
			case BiomeMountain:
				// Base stone value (e.g., 100-255)
				grid.Resources[i].StoneValue = uint8(100 + localRNG.IntN(156))
				// Secondary roll for Iron
				if localRNG.IntN(100) < 30 { // 30% chance for Iron
					grid.Resources[i].IronValue = uint8(20 + localRNG.IntN(81)) // 20-100 Iron
				}
			}

			// Add to FoodCache if food is present
			if grid.Resources[i].FoodValue > 0 {
				grid.FoodCache = append(grid.FoodCache, i)
			}
		}
	}
}

// planPlanet draws the macro layout (continents, islands, lakes) from the
// seeded RNG. All randomness is consumed in a fixed order for determinism.
func planPlanet(w, h int, rng *rand.Rand) planetPlan {
	fw, fh := float32(w), float32(h)
	md := float32(min(w, h))
	var plan planetPlan

	// Bands split the map along one axis with a guaranteed ocean channel, so at
	// least two landmasses can never merge into one. Each band is divided into
	// one slot per continent, guaranteeing layout capacity deterministically.
	vertical := rng.IntN(2) == 0
	span, vSpan := fw, fh
	if !vertical {
		span, vSpan = fh, fw
	}
	split := span * (0.44 + rng.Float32()*0.12)
	channel := span * 0.04
	edge := span * 0.015
	bands := [2][2]float32{
		{edge, split - channel},
		{split + channel, span - edge},
	}

	k := 3 + rng.IntN(3) // 3-5 continents
	startBand := rng.IntN(2)
	var cnt [2]int
	for i := 0; i < k; i++ {
		cnt[(i+startBand)%2]++
	}

	// Random per-slot weights break the regular grid look.
	weights := make([]float32, k)
	var wSum [2]float32
	for i := 0; i < k; i++ {
		weights[i] = 0.75 + rng.Float32()*0.5
		wSum[(i+startBand)%2] += weights[i]
	}

	// Gaps between a continent's support and its cell edge. Same-band gaps are
	// small on purpose: near-touching continents form natural isthmuses while
	// the band channel still guarantees two disjoint landmasses.
	const uGap, vGap float32 = 12, 8
	vEdge := vSpan * 0.015

	var vCursor [2]float32
	vCursor[0] = vEdge
	vCursor[1] = vEdge
	for i := 0; i < k; i++ {
		band := (i + startBand) % 2

		u0, u1 := bands[band][0], bands[band][1]
		slotH := (vSpan - 2*vEdge) * weights[i] / wSum[band]
		v0 := vCursor[band]
		v1 := v0 + slotH
		vCursor[band] = v1

		ru := ((u1-u0)/2 - uGap) * (0.88 + rng.Float32()*0.12)
		rv := ((v1-v0)/2 - vGap) * (0.88 + rng.Float32()*0.12)
		if ru < 2 {
			ru = 2
		}
		if rv < 2 {
			rv = 2
		}
		u := uniformIn(rng, u0+ru+uGap, u1-ru-uGap)
		v := uniformIn(rng, v0+rv+vGap, v1-rv-vGap)
		shape := 0.55 + rng.Float32()*0.45

		x, y, rx, ry := u, v, ru, rv
		if !vertical {
			x, y = v, u
			rx, ry = rv, ru
		}
		core := genBlob{x, y, rx, ry, shape}
		plan.lobes = append(plan.lobes, core)
		plan.cores = append(plan.cores, core)

		// Lobes break the elliptical silhouette. Offset + radius stays within
		// the core support so band separation guarantees still hold.
		nl := 2 + rng.IntN(2)
		for j := 0; j < nl; j++ {
			ox := (rng.Float32()*2 - 1) * 0.30 * rx
			oy := (rng.Float32()*2 - 1) * 0.30 * ry
			plan.lobes = append(plan.lobes, genBlob{
				x + ox, y + oy,
				rx * (0.45 + rng.Float32()*0.25),
				ry * (0.45 + rng.Float32()*0.25),
				rng.Float32(),
			})
		}

		// Lake site candidates well inside the continent core.
		var sites []genBlob
		for j := 0; j < 10; j++ {
			ang := float64(rng.Float32()) * 2 * smath.Pi
			dr := 0.15 + rng.Float32()*0.35
			lr := md * 0.006 * (0.7 + rng.Float32()*0.8)
			if lr < 2 {
				lr = 2
			} else if lr > 12 {
				lr = 12
			}
			sites = append(sites, genBlob{
				x + float32(smath.Cos(ang))*dr*rx,
				y + float32(smath.Sin(ang))*dr*ry,
				lr, lr, 0,
			})
		}
		plan.lakeSets = append(plan.lakeSets, sites)
	}

	// Island chains in open ocean, kept clear of all continental support.
	if md >= 64 {
		var bestX, bestY, bestR, bestClear float32
		bestClear = -smath.MaxFloat32
		arcs := 2 + rng.IntN(3)
		for a := 0; a < arcs; a++ {
			r := md * (0.006 + rng.Float32()*0.006)
			var ax, ay float32
			found := false
			for attempt := 0; attempt < 160; attempt++ {
				ax = fw * (0.06 + rng.Float32()*0.88)
				ay = fh * (0.06 + rng.Float32()*0.88)
				clear := oceanClearance(plan.lobes, ax, ay)
				if clear > bestClear {
					bestClear, bestX, bestY, bestR = clear, ax, ay, r
				}
				if clear >= 24+3*r {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			theta := float64(rng.Float32()) * 2 * smath.Pi
			nb := 3 + rng.IntN(4)
			bx, by := ax, ay
			for b := 0; b < nb; b++ {
				br := r * (0.8 + rng.Float32()*0.5)
				if bx > br && by > br && bx < fw-br && by < fh-br &&
					oceanClearance(plan.lobes, bx, by) >= 16+2*br {
					plan.islands = append(plan.islands, genBlob{bx, by, br, br, 0})
				}
				step := r * (2.6 + rng.Float32())
				theta += (float64(rng.Float32()) - 0.5) * 0.7
				bx += step * float32(smath.Cos(theta))
				by += step * float32(smath.Sin(theta))
			}
		}
		// Fallback: guarantee at least one island if every arc was rejected.
		if len(plan.islands) == 0 && bestClear >= 8+bestR {
			plan.islands = append(plan.islands, genBlob{bestX, bestY, bestR, bestR, 0})
		}
	}

	return plan
}

// carveLakes turns candidate sites into enclosed inland water. If no natural
// candidate fits, one lake is forced at a continent core with a raised rim.
func carveLakes(grid *MapGrid, plan *planetPlan) {
	if min(grid.Width, grid.Height) < 64 {
		return
	}
	carved := false
	for _, sites := range plan.lakeSets {
		for _, s := range sites {
			if tryCarveLake(grid, s, false) {
				carved = true
				break
			}
		}
	}
	if !carved && len(plan.cores) > 0 {
		c := plan.cores[0]
		r := float32(min(grid.Width, grid.Height)) * 0.006
		if r < 3 {
			r = 3
		}
		tryCarveLake(grid, genBlob{c.x, c.y, r, r, 0}, true)
	}
}

// tryCarveLake sinks a disk of water below sea level. In normal mode the
// surrounding rim must already be solid land; in force mode the rim is raised.
func tryCarveLake(grid *MapGrid, s genBlob, force bool) bool {
	r := int(s.rx)
	cx, cy := int(s.x), int(s.y)
	if r < 2 {
		return false
	}
	if cx-r-2 < 0 || cy-r-2 < 0 || cx+r+2 >= grid.Width || cy+r+2 >= grid.Height {
		return false
	}
	rr := r * r
	rimR := (r + 2) * (r + 2)
	if !force {
		for dy := -r - 2; dy <= r+2; dy++ {
			for dx := -r - 2; dx <= r+2; dx++ {
				if dx*dx+dy*dy > rimR {
					continue
				}
				if grid.Tiles[(cy+dy)*grid.Width+cx+dx].Elevation < genLakeRimElev {
					return false
				}
			}
		}
	}
	for dy := -r - 2; dy <= r+2; dy++ {
		for dx := -r - 2; dx <= r+2; dx++ {
			d2 := dx*dx + dy*dy
			i := (cy+dy)*grid.Width + cx + dx
			if d2 <= rr {
				grid.Tiles[i].Elevation = genLakeElev
			} else if force && d2 <= rimR && grid.Tiles[i].Elevation < genLakeRimElev {
				grid.Tiles[i].Elevation = genLakeRimElev
			}
		}
	}
	return true
}

// oceanClearance returns the distance (in tiles, approximately) from a point
// to the nearest continental support edge. Negative values are inside support.
func oceanClearance(lobes []genBlob, x, y float32) float32 {
	clear := float32(smath.MaxFloat32)
	for _, b := range lobes {
		dx := (x - b.x) / b.rx
		dy := (y - b.y) / b.ry
		dn := float32(smath.Sqrt(float64(dx*dx + dy*dy)))
		rMin := b.rx
		if b.ry < rMin {
			rMin = b.ry
		}
		if c := (dn - 1) * rMin; c < clear {
			clear = c
		}
	}
	return clear
}

// quantileThreshold finds the approximate value below which `frac` of the
// samples fall, using a fixed histogram (O(n), deterministic).
func quantileThreshold(raw []float32, lo, hi, frac float32) float32 {
	const bins = 4096
	if hi <= lo {
		return lo
	}
	var hist [bins]int
	scale := float32(bins-1) / (hi - lo)
	for _, v := range raw {
		hist[int((v-lo)*scale)]++
	}
	target := int(frac * float32(len(raw)))
	cum := 0
	for b := 0; b < bins; b++ {
		cum += hist[b]
		if cum >= target {
			return lo + float32(b+1)/scale
		}
	}
	return hi
}

// fbm sums `octaves` of Perlin noise (lacunarity 2, gain 0.5), normalized to
// roughly [-1, 1].
func fbm(n *math.Perlin, x, y float32, octaves int) float32 {
	var sum, norm float32
	amp := float32(1)
	freq := float32(1)
	for o := 0; o < octaves; o++ {
		sum += n.Noise2D(x*freq, y*freq) * amp
		norm += amp
		amp *= 0.5
		freq *= 2
	}
	return sum / norm
}

// latitudeMoisture maps latitude (0 pole, 1 equator) to a moisture bias:
// wet equator, dry subtropics, damp temperate belt, dry poles.
func latitudeMoisture(lat float32) float32 {
	switch {
	case lat < 0.45:
		return lerpf(0.35, 0.60, lat/0.45)
	case lat < 0.70:
		return lerpf(0.60, 0.20, (lat-0.45)/0.25)
	default:
		return lerpf(0.20, 0.95, (lat-0.70)/0.30)
	}
}

// uniformIn returns a uniform sample in [lo, hi], or the midpoint when the
// interval is inverted (degenerate small-map case).
func uniformIn(rng *rand.Rand, lo, hi float32) float32 {
	if hi <= lo {
		return (lo + hi) / 2
	}
	return lo + rng.Float32()*(hi-lo)
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerpf(a, b, t float32) float32 {
	return a + (b-a)*t
}
