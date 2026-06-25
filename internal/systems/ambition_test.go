package systems

import (
	"reflect"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: AmbitionSystem E2E Tests.
// Deterministic, headless coverage of the pure offer/accept/dismiss helpers and
// the tick-gated AmbitionSystem completion path (wealth -> prestige reward).

// spawnAmbitionPlayer creates a possessed entity carrying the full component set
// the AmbitionSystem reads. CityID/FamilyID seed Affiliation; identityID seeds
// Identity; wealth seeds Needs.Wealth.
func spawnAmbitionPlayer(world *ecs.World, identityID uint64, cityID, familyID uint32, wealth float32) ecs.Entity {
	possessedID := ecs.ComponentID[components.Possessed](world)
	ambID := ecs.ComponentID[components.AmbitionsComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	needsID := ecs.ComponentID[components.Needs](world)
	legacyID := ecs.ComponentID[components.Legacy](world)

	e := world.NewEntity(possessedID, ambID, identID, affID, needsID, legacyID)
	(*components.Identity)(world.Get(e, identID)).ID = identityID
	aff := (*components.Affiliation)(world.Get(e, affID))
	aff.CityID = cityID
	aff.FamilyID = familyID
	(*components.Needs)(world.Get(e, needsID)).Wealth = wealth
	return e
}

// TestGenerateOffersDeterminism verifies repeated calls against the same world
// state yield identical offer slices, and that an active ambition type is not
// re-offered (dedupe).
func TestGenerateOffersDeterminism(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	ambID := ecs.ComponentID[components.AmbitionsComponent](&world)

	// CityID 0 + no admin marker means the Ruler offer is skipped; FamilyID 0
	// skips the Heir offer. Wealth + Builder offers remain (and a Ruler is gated
	// out), so we expect a stable, small offer set.
	player := spawnAmbitionPlayer(&world, 1, 7, 0, 60)

	first := GenerateOffers(&world, hooks, player, 100)
	second := GenerateOffers(&world, hooks, player, 100)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("GenerateOffers not deterministic:\n first=%+v\nsecond=%+v", first, second)
	}

	// With CityID set and no AdministrationMarker, a Ruler offer must be present.
	if !hasAmbitionType(first, components.AmbitionRuler) {
		t.Errorf("expected an AmbitionRuler offer when CityID set and no admin marker; got %+v", first)
	}
	// Wealth goal must be max(100, Wealth*2) = max(100, 120) = 120.
	for _, a := range first {
		if a.Type == components.AmbitionWealth && a.Goal != 120 {
			t.Errorf("wealth offer Goal = %d, want 120", a.Goal)
		}
	}

	// Dedupe: make AmbitionWealth active, then it must not be offered again.
	amb := (*components.AmbitionsComponent)(world.Get(player, ambID))
	amb.Ambitions = []components.Ambition{{Type: components.AmbitionWealth, Goal: 999, Accepted: true}}

	deduped := GenerateOffers(&world, hooks, player, 100)
	if hasAmbitionType(deduped, components.AmbitionWealth) {
		t.Errorf("active AmbitionWealth must not be re-offered; got %+v", deduped)
	}
	// The active ambition is unaffected by generation (offers are returned, not stored).
	if len(amb.Ambitions) != 1 || amb.Ambitions[0].Goal != 999 {
		t.Errorf("GenerateOffers must not mutate active ambitions; got %+v", amb.Ambitions)
	}
}

// TestAcceptOffer verifies Offers[idx] moves into Ambitions with Accepted=true,
// is removed from Offers, and that out-of-range / full-active-list calls fail.
func TestAcceptOffer(t *testing.T) {
	amb := &components.AmbitionsComponent{
		Offers: []components.Ambition{
			{Type: components.AmbitionWealth, Goal: 100},
			{Type: components.AmbitionBuilder, Goal: 3},
		},
	}

	// Out-of-range (negative and too-large) returns false and mutates nothing.
	if AcceptOffer(amb, -1) {
		t.Errorf("AcceptOffer(-1) = true, want false")
	}
	if AcceptOffer(amb, 2) {
		t.Errorf("AcceptOffer(2) on len-2 offers = true, want false")
	}
	if len(amb.Offers) != 2 || len(amb.Ambitions) != 0 {
		t.Fatalf("failed accept mutated state: offers=%d ambitions=%d", len(amb.Offers), len(amb.Ambitions))
	}

	// Accept index 0: Builder offer remains, Wealth becomes an active Accepted ambition.
	if !AcceptOffer(amb, 0) {
		t.Fatalf("AcceptOffer(0) = false, want true")
	}
	if len(amb.Offers) != 1 || amb.Offers[0].Type != components.AmbitionBuilder {
		t.Errorf("after accept, offers = %+v, want [Builder]", amb.Offers)
	}
	if len(amb.Ambitions) != 1 {
		t.Fatalf("after accept, ambitions len = %d, want 1", len(amb.Ambitions))
	}
	if got := amb.Ambitions[0]; got.Type != components.AmbitionWealth || !got.Accepted || got.Goal != 100 {
		t.Errorf("accepted ambition = %+v, want {Wealth, Accepted, Goal 100}", got)
	}

	// Full active list (maxActive=3): accepting is denied even with offers present.
	full := &components.AmbitionsComponent{
		Ambitions: []components.Ambition{
			{Type: components.AmbitionWealth},
			{Type: components.AmbitionBuilder},
			{Type: components.AmbitionHeir},
		},
		Offers: []components.Ambition{{Type: components.AmbitionRuler}},
	}
	if AcceptOffer(full, 0) {
		t.Errorf("AcceptOffer should fail when active list is full (maxActive=3)")
	}
	if len(full.Offers) != 1 || len(full.Ambitions) != 3 {
		t.Errorf("denied accept mutated state: offers=%d ambitions=%d", len(full.Offers), len(full.Ambitions))
	}
}

// TestDismissOffer verifies Offers[idx] is dropped and out-of-range returns false.
func TestDismissOffer(t *testing.T) {
	amb := &components.AmbitionsComponent{
		Offers: []components.Ambition{
			{Type: components.AmbitionWealth},
			{Type: components.AmbitionBuilder},
			{Type: components.AmbitionRuler},
		},
	}

	if DismissOffer(amb, -1) {
		t.Errorf("DismissOffer(-1) = true, want false")
	}
	if DismissOffer(amb, 3) {
		t.Errorf("DismissOffer(3) on len-3 offers = true, want false")
	}
	if len(amb.Offers) != 3 {
		t.Fatalf("failed dismiss mutated offers len = %d, want 3", len(amb.Offers))
	}

	// Drop the middle offer; the rest shift down and nothing moves to Ambitions.
	if !DismissOffer(amb, 1) {
		t.Fatalf("DismissOffer(1) = false, want true")
	}
	if len(amb.Offers) != 2 {
		t.Fatalf("after dismiss, offers len = %d, want 2", len(amb.Offers))
	}
	if amb.Offers[0].Type != components.AmbitionWealth || amb.Offers[1].Type != components.AmbitionRuler {
		t.Errorf("after dismiss(1), offers = %+v, want [Wealth, Ruler]", amb.Offers)
	}
	if len(amb.Ambitions) != 0 {
		t.Errorf("dismiss must not add to active ambitions; got %d", len(amb.Ambitions))
	}
}

// TestRecordPlayerBuild verifies BuiltCount increments and that the active,
// not-yet-done Builder ambition's Progress tracks the new count while completed
// or non-Builder ambitions are left untouched.
func TestRecordPlayerBuild(t *testing.T) {
	amb := &components.AmbitionsComponent{
		BuiltCount: 1,
		Ambitions: []components.Ambition{
			{Type: components.AmbitionBuilder, Goal: 4, Progress: 1, Accepted: true},
			{Type: components.AmbitionBuilder, Goal: 2, Progress: 2, Accepted: true, Done: true}, // already done, untouched
			{Type: components.AmbitionWealth, Goal: 100, Progress: 50, Accepted: true},           // unaffected type
		},
	}

	RecordPlayerBuild(amb)

	if amb.BuiltCount != 2 {
		t.Errorf("BuiltCount = %d, want 2", amb.BuiltCount)
	}
	if amb.Ambitions[0].Progress != 2 {
		t.Errorf("active Builder Progress = %d, want 2", amb.Ambitions[0].Progress)
	}
	if amb.Ambitions[1].Progress != 2 {
		t.Errorf("done Builder Progress = %d, want 2 (must not change)", amb.Ambitions[1].Progress)
	}
	if amb.Ambitions[2].Progress != 50 {
		t.Errorf("Wealth Progress = %d, want 50 (must not change)", amb.Ambitions[2].Progress)
	}

	// A second build keeps bumping the active Builder toward its Goal.
	RecordPlayerBuild(amb)
	if amb.BuiltCount != 3 || amb.Ambitions[0].Progress != 3 {
		t.Errorf("after second build: BuiltCount=%d Progress=%d, want 3 and 3", amb.BuiltCount, amb.Ambitions[0].Progress)
	}
}

// TestAmbitionSystemWealthCompletion drives the full system: an active wealth
// ambition with a small Goal and Needs.Wealth above it must complete on the
// 400th Update tick, flipping Done and awarding +25 prestige exactly once.
func TestAmbitionSystemWealthCompletion(t *testing.T) {
	world := ecs.NewWorld()
	hooks := engine.NewSparseHookGraph()

	ambID := ecs.ComponentID[components.AmbitionsComponent](&world)
	legacyID := ecs.ComponentID[components.Legacy](&world)

	// CityID 0 + FamilyID 0 keeps offer refresh minimal and irrelevant here.
	player := spawnAmbitionPlayer(&world, 42, 0, 0, 200)
	amb := (*components.AmbitionsComponent)(world.Get(player, ambID))
	amb.Ambitions = []components.Ambition{
		{Type: components.AmbitionWealth, Goal: 50, Accepted: true},
	}
	legacy := (*components.Legacy)(world.Get(player, legacyID))
	legacy.Prestige = 10

	sys := NewAmbitionSystem(hooks)

	// The system only acts every 400 ticks (ambitionTickRate). Tick 399 times:
	// nothing should fire yet.
	for i := 0; i < ambitionTickRate-1; i++ {
		sys.Update(&world)
	}
	if amb.Ambitions[0].Done {
		t.Fatalf("ambition completed before the 400th tick")
	}
	if legacy.Prestige != 10 {
		t.Fatalf("prestige changed before the 400th tick: %d", legacy.Prestige)
	}

	// 400th tick: completion fires.
	sys.Update(&world)
	if !amb.Ambitions[0].Done {
		t.Errorf("wealth ambition Done = false after 400 ticks, want true")
	}
	if amb.Ambitions[0].Progress != 200 {
		t.Errorf("wealth Progress = %d, want 200 (= Needs.Wealth)", amb.Ambitions[0].Progress)
	}
	if legacy.Prestige != 35 {
		t.Errorf("prestige = %d after completion, want 35 (10 + 25 reward)", legacy.Prestige)
	}

	// Run a second 400-tick window: a Done ambition must not be re-rewarded.
	for i := 0; i < ambitionTickRate; i++ {
		sys.Update(&world)
	}
	if legacy.Prestige != 35 {
		t.Errorf("prestige = %d after re-completion window, want 35 (reward must apply once)", legacy.Prestige)
	}
}
