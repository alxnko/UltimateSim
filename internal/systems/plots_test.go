package systems

import (
	"errors"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.4): plot tests.
// Deterministic, headless coverage of StartPlot validation, assassination
// resolving through the DeathSystem path, seizure forcing a coup through the
// rank path, and spymaster-driven exposure penalties.

// spawnPlotter creates an NPC+Identity+Affiliation(+Genome) entity.
func spawnPlotter(world *ecs.World, identityID uint64, intellect uint8) ecs.Entity {
	e := spawnDynastyNPC(world, identityID, 1, 30)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	world.Add(e, genomeID)
	(*components.GenomeComponent)(world.Get(e, genomeID)).Intellect = intellect
	return e
}

// addConspirators places positive incoming hooks on the plotter so plot Power
// rises by conspiratorPowerBonus per ally.
func addConspirators(hooks *engine.SparseHookGraph, plotterID uint64, count int) {
	for i := 0; i < count; i++ {
		hooks.AddHook(uint64(1000+i), plotterID, conspiratorHookMin)
	}
}

// TestStartPlotValidation verifies kind, self, liveness, and single-plot rules.
func TestStartPlotValidation(t *testing.T) {
	world := ecs.NewWorld()

	plotter := spawnDynastyNPC(&world, 1, 1, 30)
	target := spawnDynastyNPC(&world, 2, 2, 30)

	if err := StartPlot(&world, plotter, target, 99); !errors.Is(err, ErrPlotInvalidKind) {
		t.Errorf("invalid kind: err = %v, want ErrPlotInvalidKind", err)
	}
	if err := StartPlot(&world, plotter, plotter, components.PlotAssassinate); !errors.Is(err, ErrPlotSelf) {
		t.Errorf("self plot: err = %v, want ErrPlotSelf", err)
	}

	if err := StartPlot(&world, plotter, target, components.PlotAssassinate); err != nil {
		t.Fatalf("valid plot rejected: %v", err)
	}
	plotID := ecs.ComponentID[components.PlotComponent](&world)
	plot := (*components.PlotComponent)(world.Get(plotter, plotID))
	if plot.TargetID != 2 || plot.Kind != components.PlotAssassinate {
		t.Errorf("plot = {TargetID:%d Kind:%d}, want {2 %d}", plot.TargetID, plot.Kind, components.PlotAssassinate)
	}

	// One plot at a time.
	if err := StartPlot(&world, plotter, target, components.PlotSeizeRule); !errors.Is(err, ErrPlotAlreadyActive) {
		t.Errorf("double plot: err = %v, want ErrPlotAlreadyActive", err)
	}

	// Dead target rejected.
	dead := spawnDynastyNPC(&world, 3, 3, 30)
	world.RemoveEntity(dead)
	fresh := spawnDynastyNPC(&world, 4, 4, 30)
	if err := StartPlot(&world, fresh, dead, components.PlotAssassinate); !errors.Is(err, ErrPlotNotAlive) {
		t.Errorf("dead target: err = %v, want ErrPlotNotAlive", err)
	}
}

// TestPlotAssassinateResolves runs a real world where a high-power plot
// matures in one cycle and kills the target through the existing death path
// (Vitals.Blood = 0, then DeathSystem despawns it).
func TestPlotAssassinateResolves(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	// Intellect 255 (+51) + 8 conspirators (+40) + base 10 = Power 101: one cycle.
	plotter := spawnPlotter(&world, 1, 255)
	addConspirators(hooks, 1, 8)

	target := spawnDynastyNPC(&world, 2, 2, 40)
	needsID := ecs.ComponentID[components.Needs](&world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
	world.Add(target, needsID)
	world.Add(target, vitalsID)
	tNeeds := (*components.Needs)(world.Get(target, needsID))
	tNeeds.Food = 100
	tVitals := (*components.VitalsComponent)(world.Get(target, vitalsID))
	tVitals.Blood = 100

	if err := StartPlot(&world, plotter, target, components.PlotAssassinate); err != nil {
		t.Fatalf("StartPlot failed: %v", err)
	}

	sys := NewPlotSystem(&world, hooks)
	for i := 0; i < PlotTickRate; i++ {
		sys.Update(&world)
	}

	// The plot resolved: lethal damage landed and the plot was consumed.
	tVitals = (*components.VitalsComponent)(world.Get(target, vitalsID))
	if tVitals.Blood != 0 {
		t.Fatalf("target Blood = %v, want 0 after assassination", tVitals.Blood)
	}
	plotID := ecs.ComponentID[components.PlotComponent](&world)
	if world.Has(plotter, plotID) {
		t.Errorf("plot component should be removed after resolution")
	}

	// The existing death path finishes the job.
	ds := NewDeathSystem(&world, hooks)
	ds.Update(&world)
	if world.Alive(target) {
		t.Errorf("target should be despawned by DeathSystem after Blood hit 0")
	}
}

// TestPlotSeizeForcesCoup runs a real world where the sitting ruler is far too
// popular for a legitimate claim, so the matured plot forces a coup: marker
// transfer plus usurper legitimacy.
func TestPlotSeizeForcesCoup(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	affID := ecs.ComponentID[components.Affiliation](&world)
	adminID := ecs.ComponentID[components.AdministrationMarker](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)

	// Sitting ruler of city 3 with overwhelming support (score 100).
	ruler := spawnDynastyNPC(&world, 10, 2, 50)
	(*components.Affiliation)(world.Get(ruler, affID)).CityID = 3
	world.Add(ruler, adminID)
	hooks.AddHook(200, 10, 100)

	// The plotter: same city, one-cycle plot power, weak popular support (40 < 101).
	plotter := spawnPlotter(&world, 1, 255)
	(*components.Affiliation)(world.Get(plotter, affID)).CityID = 3
	addConspirators(hooks, 1, 8)

	if CanClaimLeadership(&world, hooks, plotter) {
		t.Fatalf("precondition broken: plotter must NOT be able to claim legitimately")
	}

	if err := StartPlot(&world, plotter, ruler, components.PlotSeizeRule); err != nil {
		t.Fatalf("StartPlot failed: %v", err)
	}

	sys := NewPlotSystem(&world, hooks)
	for i := 0; i < PlotTickRate; i++ {
		sys.Update(&world)
	}

	if world.Has(ruler, adminID) {
		t.Errorf("displaced ruler should have lost the AdministrationMarker")
	}
	if !world.Has(plotter, adminID) {
		t.Fatalf("plotter should have gained the AdministrationMarker")
	}
	if !world.Has(plotter, legitID) {
		t.Fatalf("usurper should have a seeded LegitimacyComponent")
	}
	if score := (*components.LegitimacyComponent)(world.Get(plotter, legitID)).Score; score != usurperLegitimacy {
		t.Errorf("usurper legitimacy = %d, want %d", score, usurperLegitimacy)
	}
}

// TestPlotExposure runs a weak plot against a spymaster-guarded target: the
// deterministic discovery roll exposes it, penalties land (legitimacy + hook),
// and the plot unravels before maturing.
func TestPlotExposure(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	affID := ecs.ComponentID[components.Affiliation](&world)
	legitID := ecs.ComponentID[components.LegitimacyComponent](&world)
	needsID := ecs.ComponentID[components.Needs](&world)
	capitalID := ecs.ComponentID[components.CapitalComponent](&world)
	countryCompID := ecs.ComponentID[components.CountryComponent](&world)
	councilID := ecs.ComponentID[components.CouncilComponent](&world)

	// Country 5 capital with an appointed Spymaster (+25% discovery).
	capital := world.NewEntity(capitalID, countryCompID, affID, councilID)
	(*components.Affiliation)(world.Get(capital, affID)).CountryID = 5
	(*components.CouncilComponent)(world.Get(capital, councilID)).Spymaster = 42

	// The target lives in the guarded country.
	target := spawnDynastyNPC(&world, 2, 2, 40)
	(*components.Affiliation)(world.Get(target, affID)).CountryID = 5
	world.Add(target, needsID)
	(*components.Needs)(world.Get(target, needsID)).Food = 100

	// A weak lone plotter (no genome, no conspirators): Power 10, so the plot
	// needs 10 cycles and the 35% discovery roll fires long before maturity.
	plotter := spawnDynastyNPC(&world, 1, 1, 30)
	world.Add(plotter, legitID)
	(*components.LegitimacyComponent)(world.Get(plotter, legitID)).Score = 50

	if err := StartPlot(&world, plotter, target, components.PlotAssassinate); err != nil {
		t.Fatalf("StartPlot failed: %v", err)
	}

	sys := NewPlotSystem(&world, hooks)
	plotID := ecs.ComponentID[components.PlotComponent](&world)

	// Run until the exposed plot unravels (deterministic; cap at 20 cycles).
	for i := 0; i < 20*PlotTickRate && world.Has(plotter, plotID); i++ {
		sys.Update(&world)
	}

	if world.Has(plotter, plotID) {
		t.Fatalf("plot should have been removed after exposure")
	}
	// The target survived: the plot was exposed, not resolved.
	if food := (*components.Needs)(world.Get(target, needsID)).Food; food != 100 {
		t.Fatalf("target was assassinated (Food=%v); plot should have been exposed instead", food)
	}
	// Exposure penalties: target resents the plotter, legitimacy dented.
	if got := hooks.GetHook(2, 1); got != plotExposureHookPenalty {
		t.Errorf("target->plotter hook = %d, want %d", got, plotExposureHookPenalty)
	}
	if score := (*components.LegitimacyComponent)(world.Get(plotter, legitID)).Score; score != 50-plotExposureLegitimacyLoss {
		t.Errorf("plotter legitimacy = %d, want %d", score, 50-plotExposureLegitimacyLoss)
	}
}

// TestPlotDeterminism runs the same plot scenario twice and requires identical
// progress trajectories and outcomes.
func TestPlotDeterminism(t *testing.T) {
	run := func() (uint16, float32) {
		world := ecs.NewWorld()
		hooks := engine.NewSparseHookGraph()

		plotter := spawnPlotter(&world, 1, 100)
		addConspirators(hooks, 1, 2)

		target := spawnDynastyNPC(&world, 2, 2, 40)
		vitalsID := ecs.ComponentID[components.VitalsComponent](&world)
		world.Add(target, vitalsID)
		(*components.VitalsComponent)(world.Get(target, vitalsID)).Blood = 100

		if err := StartPlot(&world, plotter, target, components.PlotAssassinate); err != nil {
			t.Fatalf("StartPlot failed: %v", err)
		}

		sys := NewPlotSystem(&world, hooks)
		for i := 0; i < 3*PlotTickRate; i++ {
			sys.Update(&world)
		}

		var progress uint16
		plotID := ecs.ComponentID[components.PlotComponent](&world)
		if world.Has(plotter, plotID) {
			progress = (*components.PlotComponent)(world.Get(plotter, plotID)).Progress
		}
		blood := (*components.VitalsComponent)(world.Get(target, vitalsID)).Blood
		return progress, blood
	}

	p1, b1 := run()
	p2, b2 := run()
	if p1 != p2 || b1 != b2 {
		t.Fatalf("determinism broken: run1=(%d, %v) run2=(%d, %v)", p1, b1, p2, b2)
	}
}
