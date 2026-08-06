package ui

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (P5 L2/L3): logic-only overlay tests — ghost tile
// snapping, overhead visibility thresholds, bar math, labels, and the
// player-construction toast tracker against a real arche world. No ebiten
// surface is created, so these run headless and deterministically.

func TestGhostTileSnap(t *testing.T) {
	cases := []struct {
		wx, wy   float32
		tx, ty   int
	}{
		{0, 0, 0, 0},
		{5.7, 5.2, 5, 5},
		{5.999, 5.0, 5, 5},
		{12.0, 3.0, 12, 3},
		// Floor semantics: negatives snap left/up, not toward zero.
		{-0.3, 2.9, -1, 2},
		{-1.5, -0.1, -2, -1},
	}
	for _, tc := range cases {
		gx, gy := GhostTile(tc.wx, tc.wy)
		if gx != tc.tx || gy != tc.ty {
			t.Errorf("GhostTile(%v, %v) = (%d, %d), want (%d, %d)",
				tc.wx, tc.wy, gx, gy, tc.tx, tc.ty)
		}
	}
}

func TestOverheadsVisible(t *testing.T) {
	// Mirrors LensZoomThreshold: strategic view below, action view at/above.
	if OverheadZoomThreshold != LensZoomThreshold {
		t.Fatalf("OverheadZoomThreshold = %v, want LensZoomThreshold %v",
			OverheadZoomThreshold, LensZoomThreshold)
	}
	if OverheadsVisible(0.49) {
		t.Errorf("OverheadsVisible(0.49) = true, want false (strategic view)")
	}
	if !OverheadsVisible(0.5) {
		t.Errorf("OverheadsVisible(0.5) = false, want true (action view)")
	}
	if !OverheadsVisible(1.5) {
		t.Errorf("OverheadsVisible(1.5) = false, want true")
	}
}

func TestNeedsHealthBar(t *testing.T) {
	if !NeedsHealthBar(89.9) {
		t.Errorf("NeedsHealthBar(89.9) = false, want true")
	}
	if NeedsHealthBar(90) {
		t.Errorf("NeedsHealthBar(90) = true, want false (threshold is exclusive)")
	}
	if NeedsHealthBar(100) {
		t.Errorf("NeedsHealthBar(100) = true, want false")
	}
	if !NeedsHealthBar(0) {
		t.Errorf("NeedsHealthBar(0) = false, want true")
	}
}

func TestSiteProgressFrac(t *testing.T) {
	if got := SiteProgressFrac(0, 0); got != 0 {
		t.Errorf("SiteProgressFrac(0, 0) = %v, want 0 (no div-by-zero)", got)
	}
	if got := SiteProgressFrac(50, 100); got != 0.5 {
		t.Errorf("SiteProgressFrac(50, 100) = %v, want 0.5", got)
	}
	if got := SiteProgressFrac(150, 100); got != 1 {
		t.Errorf("SiteProgressFrac(150, 100) = %v, want 1 (clamped)", got)
	}
	if got := SiteProgressFrac(100, 100); got != 1 {
		t.Errorf("SiteProgressFrac(100, 100) = %v, want 1", got)
	}
}

func TestStructureInitial(t *testing.T) {
	cases := []struct {
		t    uint32
		want string
	}{
		{uint32(components.StructureHouse), "H"},
		{uint32(components.StructureWorkshop), "W"},
		{uint32(components.StructureFarm), "F"},
		{uint32(components.StructureTavern), "T"},
		{999, "S"}, // unknown -> "Structure" -> "S"
	}
	for _, tc := range cases {
		if got := StructureInitial(tc.t); got != tc.want {
			t.Errorf("StructureInitial(%d) = %q, want %q", tc.t, got, tc.want)
		}
	}
}

func TestBuildCostLabel(t *testing.T) {
	if got := BuildCostLabel(components.StructureHouse); got != "House 50W/50S" {
		t.Errorf("BuildCostLabel(House) = %q, want %q", got, "House 50W/50S")
	}
	if got := BuildCostLabel(components.StructureShrine); got != "Shrine 0W/100S" {
		t.Errorf("BuildCostLabel(Shrine) = %q, want %q", got, "Shrine 0W/100S")
	}
}

// newTestSite spawns a bare ConstructionSiteComponent entity.
func newTestSite(world *ecs.World, siteType uint32, funded bool) ecs.Entity {
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](world)
	e := world.NewEntity(siteID)
	s := (*components.ConstructionSiteComponent)(world.Get(e, siteID))
	s.WoodRequired, s.StoneRequired = 50, 50
	s.MaxProgress = 100
	s.SiteType = siteType
	if funded {
		s.WoodGathered, s.StoneGathered = 50, 50
	}
	return e
}

func TestSiteToastTracker(t *testing.T) {
	world := ecs.NewWorld()
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](&world)
	structID := ecs.ComponentID[components.StructureComponent](&world)

	var got []string
	push := func(text string, tick uint64) { got = append(got, text) }
	var tr siteToastTracker

	// Baseline poll on the empty world (simulates end of warmup).
	tr.Poll(&world, 1, push)

	// A village-spawned (unfunded) site never toasts: not player-owned.
	vsite := newTestSite(&world, uint32(components.StructureHouse), false)
	tr.Poll(&world, 2, push)
	vs := (*components.ConstructionSiteComponent)(world.Get(vsite, siteID))
	vs.BuilderID = 7
	tr.Poll(&world, 3, push)
	if len(got) != 0 {
		t.Fatalf("village site produced toasts %v, want none", got)
	}

	// A player-funded site (fully funded at first sighting) toasts on builder
	// claim and again on completion.
	psite := newTestSite(&world, uint32(components.StructureTavern), true)
	tr.Poll(&world, 4, push) // registration frame: silent
	if len(got) != 0 {
		t.Fatalf("player site registration produced toasts %v, want none", got)
	}
	ps := (*components.ConstructionSiteComponent)(world.Get(psite, siteID))
	ps.BuilderID = 9
	tr.Poll(&world, 5, push)
	if len(got) != 1 || got[0] != "Builders started on your Tavern." {
		t.Fatalf("after builder claim got %v, want [Builders started on your Tavern.]", got)
	}
	tr.Poll(&world, 6, push) // no repeat
	if len(got) != 1 {
		t.Fatalf("builder-claim toast repeated: %v", got)
	}

	// Completion: ConstructionSystem swaps the site component for a structure.
	world.Remove(psite, siteID)
	world.Add(psite, structID)
	tr.Poll(&world, 7, push)
	if len(got) != 2 || got[1] != "Your Tavern is complete!" {
		t.Fatalf("after completion got %v, want completion toast last", got)
	}

	// Village site completing stays silent.
	world.Remove(vsite, siteID)
	world.Add(vsite, structID)
	tr.Poll(&world, 8, push)
	if len(got) != 2 {
		t.Fatalf("village completion produced toasts: %v", got)
	}

	// A destroyed player site (no structure) is dropped without a toast.
	dsite := newTestSite(&world, uint32(components.StructureFarm), true)
	tr.Poll(&world, 9, push)
	world.RemoveEntity(dsite)
	tr.Poll(&world, 10, push)
	if len(got) != 2 {
		t.Fatalf("destroyed site produced toasts: %v", got)
	}
}

func TestSiteToastTrackerWorldReset(t *testing.T) {
	worldA := ecs.NewWorld()
	ecs.ComponentID[components.ConstructionSiteComponent](&worldA)
	ecs.ComponentID[components.StructureComponent](&worldA)

	var got []string
	push := func(text string, tick uint64) { got = append(got, text) }
	var tr siteToastTracker
	tr.Poll(&worldA, 1, push)

	// Switching worlds (e.g. loading a save) re-baselines: pre-existing
	// funded, claimed sites register silently instead of replaying toasts.
	worldB := ecs.NewWorld()
	siteID := ecs.ComponentID[components.ConstructionSiteComponent](&worldB)
	ecs.ComponentID[components.StructureComponent](&worldB)
	e := newTestSite(&worldB, uint32(components.StructureHouse), true)
	s := (*components.ConstructionSiteComponent)(worldB.Get(e, siteID))
	s.BuilderID = 3

	tr.Poll(&worldB, 1, push)
	if len(got) != 0 {
		t.Fatalf("world switch replayed toasts: %v", got)
	}

	// But a NEW player site in the new world still toasts normally.
	e2 := newTestSite(&worldB, uint32(components.StructureFarm), true)
	tr.Poll(&worldB, 2, push)
	s2 := (*components.ConstructionSiteComponent)(worldB.Get(e2, siteID))
	s2.BuilderID = 4
	tr.Poll(&worldB, 3, push)
	if len(got) != 1 || got[0] != "Builders started on your Farm." {
		t.Fatalf("new-world player site got %v, want farm builder toast", got)
	}
}
