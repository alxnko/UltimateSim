package ui

import (
	"fmt"
	"image/color"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: strategic lenses — macro overlays satisfying the Law of
// Macro/Micro Parity. Active automatically when zoomed below LensZoomThreshold.

const LensZoomThreshold = 0.5

const (
	LensPolitical uint8 = 0
	LensWealth    uint8 = 1
	LensCrime     uint8 = 2
	LensCulture   uint8 = 3
	lensCount           = 4
)

// LensName returns the display label of a lens.
func LensName(l uint8) string {
	switch l {
	case LensPolitical:
		return "Political"
	case LensWealth:
		return "Wealth"
	case LensCrime:
		return "Crime"
	case LensCulture:
		return "Culture"
	}
	return "Unknown"
}

// LensColor maps a stable ID to a distinct hue. Deterministic.
func LensColor(seedID uint32) color.RGBA {
	if seedID == 0 {
		return color.RGBA{90, 90, 90, 255}
	}
	// Cheap integer hash -> palette index.
	h := seedID
	h ^= h >> 16
	h *= 0x45d9f3b
	h ^= h >> 16
	palette := []color.RGBA{
		{220, 80, 80, 255}, {80, 160, 220, 255}, {110, 200, 110, 255},
		{220, 180, 70, 255}, {180, 110, 220, 255}, {90, 210, 200, 255},
		{230, 130, 60, 255}, {200, 100, 160, 255}, {140, 160, 90, 255},
		{100, 120, 230, 255}, {210, 210, 120, 255}, {160, 220, 240, 255},
	}
	return palette[h%uint32(len(palette))]
}

// heatColor maps 0..1 to a blue->red heat ramp.
func heatColor(frac float32) color.RGBA {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	return color.RGBA{uint8(60 + 195*frac), 70, uint8(220 - 180*frac), 255}
}

// lensCity is the aggregated macro view of one settlement.
type lensCity struct {
	X, Y      float32
	CityID    uint32
	CountryID uint32
	Name      string
	Wealth    uint32 // total storage value
	Crime     int    // crime markers among citizens
	Language  uint16 // ruler/dominant language (first citizen sample)
	RadiusSq  float32
	IsCapital bool
	Pop       uint32
}

// lensData caches per-city aggregates, rebuilt at most every 60 ticks.
type lensData struct {
	cities    []lensCity
	lastBuild uint64
}

// Color picks the city's tint under the given lens.
func (c *lensCity) Color(lens uint8, maxWealth uint32, maxCrime int) color.RGBA {
	switch lens {
	case LensPolitical:
		if c.CountryID != 0 {
			return LensColor(c.CountryID)
		}
		return LensColor(c.CityID)
	case LensWealth:
		if maxWealth == 0 {
			maxWealth = 1
		}
		return heatColor(float32(c.Wealth) / float32(maxWealth))
	case LensCrime:
		if maxCrime == 0 {
			maxCrime = 1
		}
		return heatColor(float32(c.Crime) / float32(maxCrime))
	case LensCulture:
		return LensColor(uint32(c.Language) + 7)
	}
	return color.RGBA{200, 200, 200, 255}
}

// rebuild aggregates villages + citizens into lens cities.
func (ld *lensData) rebuild(world *ecs.World, tick uint64) {
	if tick != 0 && tick-ld.lastBuild < 60 && len(ld.cities) > 0 {
		return
	}
	ld.lastBuild = tick
	ld.cities = ld.cities[:0]

	villageID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	storID := ecs.ComponentID[components.StorageComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	capID := ecs.ComponentID[components.CapitalComponent](world)
	jurID := ecs.ComponentID[components.JurisdictionComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)

	cityIdx := make(map[uint32]int)

	q := world.Query(ecs.All(villageID, posID, affID))
	for q.Next() {
		pos := (*components.Position)(q.Get(posID))
		aff := (*components.Affiliation)(q.Get(affID))
		lc := lensCity{X: pos.X, Y: pos.Y, CityID: aff.CityID, CountryID: aff.CountryID, RadiusSq: 400}
		if q.Has(identID) {
			lc.Name = (*components.Identity)(q.Get(identID)).Name
		}
		if lc.Name == "" {
			lc.Name = fmt.Sprintf("City %d", aff.CityID)
		}
		if q.Has(storID) {
			st := (*components.StorageComponent)(q.Get(storID))
			lc.Wealth = st.Wood + st.Stone + st.Iron*3 + st.Food
		}
		if q.Has(capID) {
			lc.IsCapital = true
		}
		if q.Has(jurID) {
			lc.RadiusSq = (*components.JurisdictionComponent)(q.Get(jurID)).RadiusSquared
		}
		if q.Has(popID) {
			lc.Pop = (*components.PopulationComponent)(q.Get(popID)).Count
		}
		cityIdx[aff.CityID] = len(ld.cities)
		ld.cities = append(ld.cities, lc)
	}

	// Aggregate citizen-level data: crime markers + sample language.
	npcID := ecs.ComponentID[components.NPC](world)
	crimeID := ecs.ComponentID[components.CrimeMarker](world)
	cultID := ecs.ComponentID[components.CultureComponent](world)

	q2 := world.Query(ecs.All(npcID, affID))
	for q2.Next() {
		aff := (*components.Affiliation)(q2.Get(affID))
		idx, ok := cityIdx[aff.CityID]
		if !ok {
			continue
		}
		if q2.Has(crimeID) {
			ld.cities[idx].Crime++
		}
		if ld.cities[idx].Language == 0 && q2.Has(cultID) {
			ld.cities[idx].Language = (*components.CultureComponent)(q2.Get(cultID)).LanguageID
		}
	}
}

// maxima computes normalization bounds for heat lenses.
func (ld *lensData) maxima() (maxWealth uint32, maxCrime int) {
	for i := range ld.cities {
		if ld.cities[i].Wealth > maxWealth {
			maxWealth = ld.cities[i].Wealth
		}
		if ld.cities[i].Crime > maxCrime {
			maxCrime = ld.cities[i].Crime
		}
	}
	return
}
