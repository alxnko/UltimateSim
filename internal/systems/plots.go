package systems

import (
	"errors"
	"math/rand/v2"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Grand Strategy Phase (spec P2.4): intrigue plots.
// A PlotComponent on the plotter accumulates Progress from plot Power every
// cycle. Power derives from the plotter's Intellect plus conspirators (allies
// with positive incoming hooks). Each cycle an undiscovered plot risks
// exposure; a Spymaster on the target's council raises the discovery chance.
// At Progress >= 100 the plot resolves through existing macro/micro channels:
// assassination deals lethal damage the DeathSystem processes (Vitals.Blood /
// Needs.Food), seizure transfers leadership through the ClaimLeadership rank
// path (forced coup when the popularity gate denies the claim).

const (
	// PlotTickRate throttles PlotSystem to every 200 ticks.
	PlotTickRate = 200

	// plotResolveThreshold: Progress needed to resolve a plot.
	plotResolveThreshold = 100
	// plotBasePower is the floor Power of any plot.
	plotBasePower = 10
	// plotPowerCap bounds Power against runaway conspiracies.
	plotPowerCap = 200
	// conspiratorHookMin: incoming hook value that counts an ally as conspirator.
	conspiratorHookMin = 2
	// conspiratorPowerBonus: Power added per conspirator.
	conspiratorPowerBonus = 5
	// plotBaseDiscoveryPct: baseline exposure chance (percent) per cycle.
	plotBaseDiscoveryPct = 10
	// plotMaxDiscoveryPct caps the exposure chance.
	plotMaxDiscoveryPct = 95
	// plotExposureLegitimacyLoss: legitimacy stripped from an exposed plotter.
	plotExposureLegitimacyLoss = 10
	// plotExposureHookPenalty: hook the target places on an exposed plotter.
	plotExposureHookPenalty = -20
	// usurperLegitimacy: seed legitimacy for a coup that forced the throne.
	usurperLegitimacy = 30
)

var (
	ErrPlotInvalidKind   = errors.New("plot rejected: unknown plot kind")
	ErrPlotSelf          = errors.New("plot rejected: cannot plot against oneself")
	ErrPlotNotAlive      = errors.New("plot rejected: plotter and target must be alive with an Identity")
	ErrPlotAlreadyActive = errors.New("plot rejected: plotter already runs a plot")
)

// StartPlot begins a plot by the plotter against the target. One plot per
// plotter at a time. Must be called outside any open query loop (structural
// mutation). StartTick is stamped by PlotSystem on its first cycle.
func StartPlot(world *ecs.World, plotter, target ecs.Entity, kind uint8) error {
	if kind != components.PlotSeizeRule && kind != components.PlotAssassinate {
		return ErrPlotInvalidKind
	}
	if plotter == target {
		return ErrPlotSelf
	}
	identID := ecs.ComponentID[components.Identity](world)
	if !world.Alive(plotter) || !world.Alive(target) ||
		!world.Has(plotter, identID) || !world.Has(target, identID) {
		return ErrPlotNotAlive
	}
	plotID := ecs.ComponentID[components.PlotComponent](world)
	if world.Has(plotter, plotID) {
		return ErrPlotAlreadyActive
	}

	targetIdentityID := (*components.Identity)(world.Get(target, identID)).ID

	world.Add(plotter, plotID)
	plot := (*components.PlotComponent)(world.Get(plotter, plotID))
	plot.TargetID = targetIdentityID
	plot.Kind = kind
	plot.StartTick = 0
	plot.Progress = 0
	plot.Power = 0
	plot.Exposed = false
	return nil
}

// plotResolution captures a matured plot for post-query execution.
type plotResolution struct {
	plotter   ecs.Entity
	target    ecs.Entity
	plotterID uint64
	targetID  uint64
	kind      uint8
}

// plotTargetInfo caches a potential victim for O(1) lookup by Identity.ID.
type plotTargetInfo struct {
	entity    ecs.Entity
	countryID uint32
}

// PlotSystem advances, exposes, and resolves plots every PlotTickRate ticks.
type PlotSystem struct {
	hooks *engine.SparseHookGraph
	tick  uint64
}

func NewPlotSystem(world *ecs.World, hooks *engine.SparseHookGraph) *PlotSystem {
	// Pre-register component IDs used by Update.
	ecs.ComponentID[components.PlotComponent](world)
	ecs.ComponentID[components.CouncilComponent](world)
	ecs.ComponentID[components.VitalsComponent](world)
	ecs.ComponentID[components.LegitimacyComponent](world)
	return &PlotSystem{hooks: hooks}
}

// IsExpensive throttles the system during fast-forward.
func (s *PlotSystem) IsExpensive() bool { return true }

func (s *PlotSystem) Update(world *ecs.World) {
	s.tick++
	if s.tick%PlotTickRate != 0 {
		return
	}

	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	plotID := ecs.ComponentID[components.PlotComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	councilID := ecs.ComponentID[components.CouncilComponent](world)

	// 1. Spymaster coverage per country (precomputed to avoid nested queries).
	spymasterPct := make(map[uint32]int)
	sq := world.Query(filter.All(capitalID, countryCompID, affID, councilID))
	for sq.Next() {
		aff := (*components.Affiliation)(sq.Get(affID))
		council := (*components.CouncilComponent)(sq.Get(councilID))
		if aff.CountryID != 0 && council.Spymaster != 0 {
			spymasterPct[aff.CountryID] = SpymasterDiscoveryBonus
		}
	}

	// 2. Victim lookup table keyed by Identity.ID.
	targets := make(map[uint64]plotTargetInfo)
	tq := world.Query(ecs.All(identID))
	for tq.Next() {
		ident := (*components.Identity)(tq.Get(identID))
		info := plotTargetInfo{entity: tq.Entity()}
		if tq.Has(affID) {
			info.countryID = (*components.Affiliation)(tq.Get(affID)).CountryID
		}
		targets[ident.ID] = info
	}

	// 3. Advance every plot; collect structural actions for after the loop.
	var toRemove []ecs.Entity
	var resolutions []plotResolution

	pq := world.Query(ecs.All(plotID, identID))
	for pq.Next() {
		plot := (*components.PlotComponent)(pq.Get(plotID))
		plotterIdent := (*components.Identity)(pq.Get(identID))

		// Exposed plots were penalized last cycle; unravel them now.
		if plot.Exposed {
			toRemove = append(toRemove, pq.Entity())
			continue
		}

		target, exists := targets[plot.TargetID]
		if !exists || !world.Alive(target.entity) {
			// The target died on its own: the plot is moot.
			toRemove = append(toRemove, pq.Entity())
			continue
		}

		if plot.StartTick == 0 {
			plot.StartTick = s.tick
		}

		// Power = base + Intellect/5 + 5 per conspirator (allies holding a
		// positive incoming hook on the plotter; counting is order-independent).
		power := plotBasePower
		if pq.Has(genomeID) {
			power += int((*components.GenomeComponent)(pq.Get(genomeID)).Intellect) / 5
		}
		if s.hooks != nil {
			conspirators := 0
			for _, val := range s.hooks.GetAllIncomingHooks(plotterIdent.ID) {
				if val >= conspiratorHookMin {
					conspirators++
				}
			}
			power += conspirators * conspiratorPowerBonus
		}
		if power > plotPowerCap {
			power = plotPowerCap
		}
		plot.Power = uint16(power)
		if int(plot.Progress)+power > 65535 {
			plot.Progress = 65535
		} else {
			plot.Progress += uint16(power)
		}

		// A plot that matures this cycle strikes before any spymaster can act.
		if plot.Progress >= plotResolveThreshold {
			resolutions = append(resolutions, plotResolution{
				plotter:   pq.Entity(),
				target:    target.entity,
				plotterID: plotterIdent.ID,
				targetID:  plot.TargetID,
				kind:      plot.Kind,
			})
			toRemove = append(toRemove, pq.Entity())
			continue
		}

		// Discovery roll: local PRNG seeded from tick + plotter identity keeps
		// determinism without draining the global RNG stream.
		var seed [32]byte
		seed[0] = byte(s.tick)
		seed[1] = byte(s.tick >> 8)
		seed[2] = byte(s.tick >> 16)
		seed[3] = byte(plotterIdent.ID)
		seed[4] = byte(plotterIdent.ID >> 8)
		seed[5] = byte(plotterIdent.ID >> 16)
		prng := rand.New(rand.NewChaCha8(seed))

		chance := plotBaseDiscoveryPct + spymasterPct[target.countryID]
		if chance > plotMaxDiscoveryPct {
			chance = plotMaxDiscoveryPct
		}
		if prng.IntN(100) < chance {
			plot.Exposed = true
			// Legitimacy penalty for a ruling plotter (non-structural).
			if pq.Has(legitID) {
				legit := (*components.LegitimacyComponent)(pq.Get(legitID))
				if legit.Score > plotExposureLegitimacyLoss {
					legit.Score -= plotExposureLegitimacyLoss
				} else {
					legit.Score = 0
				}
			}
			// The target learns who moved against them.
			if s.hooks != nil {
				s.hooks.AddHook(plot.TargetID, plotterIdent.ID, plotExposureHookPenalty)
			}
		}
	}

	// 4. Structural mutations strictly outside the query loops.
	for _, r := range resolutions {
		switch r.kind {
		case components.PlotAssassinate:
			s.resolveAssassination(world, r)
		case components.PlotSeizeRule:
			s.resolveSeizure(world, r)
		}
	}
	for _, e := range toRemove {
		if world.Alive(e) && world.Has(e, plotID) {
			world.Remove(e, plotID)
		}
	}
}

// resolveAssassination deals lethal damage through the existing death path:
// DeathSystem despawns entities with Vitals.Blood <= 0 or Needs.Food <= 0 and
// runs succession, corpse spawning, and hook cleanup itself.
func (s *PlotSystem) resolveAssassination(world *ecs.World, r plotResolution) {
	if !world.Alive(r.target) {
		return
	}
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	needsID := ecs.ComponentID[components.Needs](world)
	if world.Has(r.target, vitalsID) {
		vitals := (*components.VitalsComponent)(world.Get(r.target, vitalsID))
		vitals.Blood = 0
		vitals.Pain = 100
		return
	}
	if world.Has(r.target, needsID) {
		// No vitals to bleed: the poisoner's route.
		(*components.Needs)(world.Get(r.target, needsID)).Food = 0
	}
}

// resolveSeizure transfers leadership through the rank path. A popular plotter
// claims legitimately via ClaimLeadership; when the popularity gate denies the
// claim, the matured plot forces a coup with the same primitives (marker
// transfer + seeded legitimacy) at usurper legitimacy.
func (s *PlotSystem) resolveSeizure(world *ecs.World, r plotResolution) {
	if !world.Alive(r.plotter) {
		return
	}
	if err := ClaimLeadership(world, s.hooks, r.plotter); err == nil {
		return
	}

	affID := ecs.ComponentID[components.Affiliation](world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](world)

	if !world.Has(r.plotter, affID) {
		return
	}
	cityID := (*components.Affiliation)(world.Get(r.plotter, affID)).CityID
	if cityID == 0 {
		return // Homeless plotters have no throne to force.
	}

	// CityRuler runs a fully-consumed query; the world is unlocked afterwards.
	oldRuler, _, hadRuler := CityRuler(world, cityID)
	if hadRuler && oldRuler == r.plotter {
		return // Already on the throne.
	}
	if hadRuler && world.Has(oldRuler, adminMarkerID) {
		world.Remove(oldRuler, adminMarkerID)
	}
	if !world.Has(r.plotter, adminMarkerID) {
		world.Add(r.plotter, adminMarkerID)
	}
	if !world.Has(r.plotter, legitID) {
		world.Add(r.plotter, legitID)
		(*components.LegitimacyComponent)(world.Get(r.plotter, legitID)).Score = usurperLegitimacy
	}
}
