package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase (Playable Game Shell): E2E tests for ruler law actions.
// Run headless: go test ./internal/systems/ -run TestLawActions -count=2

// newLawTestCapital spawns a capital entity with the components FindCapitalOf
// and DeclareWar expect, returning the entity.
func newLawTestCapital(world *ecs.World, countryID uint32) ecs.Entity {
	capID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	villageID := ecs.ComponentID[components.Village](world)

	entity := world.NewEntity(capID, countryCompID, affID, villageID)
	aff := (*components.Affiliation)(world.Get(entity, affID))
	aff.CountryID = countryID
	return entity
}

// TestLawActions_LawBitToggleRoundTrip verifies that toggling a law twice
// returns the jurisdiction to its exact original state and that IsIllegal
// reads the correct bit for each interaction type.
func TestLawActions_LawBitToggleRoundTrip(t *testing.T) {
	jur := &components.JurisdictionComponent{
		IllegalActionIDs: uint32(1) << components.InteractionTheft,
	}

	// Pre-existing law reads correctly; unrelated bits read as legal.
	if !IsIllegal(jur, components.InteractionTheft) {
		t.Fatalf("expected InteractionTheft to be illegal")
	}
	if IsIllegal(jur, components.InteractionMurder) {
		t.Fatalf("expected InteractionMurder to be legal initially")
	}

	// Toggle on: murder becomes illegal, theft untouched.
	ToggleLaw(jur, components.InteractionMurder)
	if !IsIllegal(jur, components.InteractionMurder) {
		t.Fatalf("expected InteractionMurder to be illegal after toggle")
	}
	if !IsIllegal(jur, components.InteractionTheft) {
		t.Fatalf("toggle of murder must not clear theft bit")
	}

	// Toggle off: round-trips to the exact original bitmask.
	ToggleLaw(jur, components.InteractionMurder)
	if IsIllegal(jur, components.InteractionMurder) {
		t.Fatalf("expected InteractionMurder to be legal after round-trip")
	}
	if jur.IllegalActionIDs != uint32(1)<<components.InteractionTheft {
		t.Fatalf("expected bitmask round-trip, got %b", jur.IllegalActionIDs)
	}

	// Nil safety: helpers must be no-ops on missing components.
	if IsIllegal(nil, components.InteractionTheft) {
		t.Fatalf("nil jurisdiction must report legal")
	}
	ToggleLaw(nil, components.InteractionTheft) // must not panic
}

// TestLawActions_ContrabandBitToggleRoundTrip verifies contraband flags flip
// independently and round-trip to the original bitmask.
func TestLawActions_ContrabandBitToggleRoundTrip(t *testing.T) {
	c := &components.ContrabandComponent{}

	if HasContraband(c, components.ItemIron) {
		t.Fatalf("expected ItemIron legal on empty mask")
	}

	ToggleContraband(c, components.ItemIron)
	ToggleContraband(c, components.ItemFood)
	if !HasContraband(c, components.ItemIron) || !HasContraband(c, components.ItemFood) {
		t.Fatalf("expected ItemIron and ItemFood flagged, got %b", c.Contraband)
	}
	if HasContraband(c, components.ItemWood) {
		t.Fatalf("ItemWood must stay legal, got %b", c.Contraband)
	}

	ToggleContraband(c, components.ItemIron)
	ToggleContraband(c, components.ItemFood)
	if c.Contraband != 0 {
		t.Fatalf("expected empty mask after round-trip, got %b", c.Contraband)
	}

	// Nil safety.
	if HasContraband(nil, components.ItemIron) {
		t.Fatalf("nil contraband must report legal")
	}
	ToggleContraband(nil, components.ItemIron) // must not panic
}

// TestLawActions_DebasementClamp verifies SetDebasement accumulates deltas and
// clamps the result into the 0..0.9 range at both ends.
func TestLawActions_DebasementClamp(t *testing.T) {
	c := &components.CountryComponent{}

	SetDebasement(c, 0.5)
	if c.Debasement != 0.5 {
		t.Fatalf("expected 0.5, got %f", c.Debasement)
	}

	// Upper clamp.
	SetDebasement(c, 1.0)
	if c.Debasement != MaxDebasement {
		t.Fatalf("expected upper clamp %f, got %f", MaxDebasement, c.Debasement)
	}

	// Lower clamp.
	SetDebasement(c, -5.0)
	if c.Debasement != 0 {
		t.Fatalf("expected lower clamp 0, got %f", c.Debasement)
	}

	// Nil safety.
	SetDebasement(nil, 0.5) // must not panic
}

// TestLawActions_FindCapitalOf verifies the capital lookup matches on
// Affiliation.CountryID and reports missing countries.
func TestLawActions_FindCapitalOf(t *testing.T) {
	world := ecs.NewWorld()

	cap1 := newLawTestCapital(&world, 1)
	cap2 := newLawTestCapital(&world, 2)

	// A plain village (no CapitalComponent/CountryComponent) must never match.
	affID := ecs.ComponentID[components.Affiliation](&world)
	villageID := ecs.ComponentID[components.Village](&world)
	plain := world.NewEntity(affID, villageID)
	plainAff := (*components.Affiliation)(world.Get(plain, affID))
	plainAff.CountryID = 3

	found, ok := FindCapitalOf(&world, 1)
	if !ok || found != cap1 {
		t.Fatalf("expected capital of country 1, got %v ok=%v", found, ok)
	}

	found, ok = FindCapitalOf(&world, 2)
	if !ok || found != cap2 {
		t.Fatalf("expected capital of country 2, got %v ok=%v", found, ok)
	}

	if _, ok := FindCapitalOf(&world, 3); ok {
		t.Fatalf("plain village must not be returned as a capital")
	}
	if _, ok := FindCapitalOf(&world, 99); ok {
		t.Fatalf("expected no capital for unknown country 99")
	}
}

// TestLawActions_ListCountriesSorted verifies distinct CountryIDs are returned
// ascending regardless of creation order, deduplicated, and skipping 0.
func TestLawActions_ListCountriesSorted(t *testing.T) {
	world := ecs.NewWorld()

	// Deliberately unsorted creation order with a duplicate and an
	// unaffiliated (CountryID 0) capital.
	newLawTestCapital(&world, 7)
	newLawTestCapital(&world, 2)
	newLawTestCapital(&world, 5)
	newLawTestCapital(&world, 2) // Duplicate country
	newLawTestCapital(&world, 0) // Unaffiliated, must be skipped

	got := ListCountries(&world)
	want := []uint32{2, 5, 7}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}

	// Empty world yields an empty (non-nil-safe to range) list.
	emptyWorld := ecs.NewWorld()
	if list := ListCountries(&emptyWorld); len(list) != 0 {
		t.Fatalf("expected empty country list, got %v", list)
	}
}

// TestLawActions_DeclareWarAndPeace verifies the full war lifecycle: declare
// adds an active tracker, re-declare re-targets, peace deactivates without
// removing the component, and invalid targets (self, 0) are rejected.
func TestLawActions_DeclareWarAndPeace(t *testing.T) {
	world := ecs.NewWorld()
	warCompID := ecs.ComponentID[components.WarTrackerComponent](&world)

	capital := newLawTestCapital(&world, 1)
	newLawTestCapital(&world, 2)
	newLawTestCapital(&world, 3)

	// Self-war and zero-target must error and add no tracker.
	if err := DeclareWar(&world, capital, 1); err == nil {
		t.Fatalf("expected error declaring war on own country")
	}
	if err := DeclareWar(&world, capital, 0); err == nil {
		t.Fatalf("expected error declaring war on country 0")
	}
	if world.Has(capital, warCompID) {
		t.Fatalf("failed declarations must not attach WarTrackerComponent")
	}

	// Valid declaration adds the tracker.
	if err := DeclareWar(&world, capital, 2); err != nil {
		t.Fatalf("unexpected declare error: %v", err)
	}
	if !world.Has(capital, warCompID) {
		t.Fatalf("expected WarTrackerComponent on capital")
	}
	warComp := (*components.WarTrackerComponent)(world.Get(capital, warCompID))
	if warComp.TargetCountryID != 2 || !warComp.Active {
		t.Fatalf("expected active war on country 2, got target=%d active=%v",
			warComp.TargetCountryID, warComp.Active)
	}

	// Re-declaring against a new target reuses the existing component.
	if err := DeclareWar(&world, capital, 3); err != nil {
		t.Fatalf("unexpected re-declare error: %v", err)
	}
	warComp = (*components.WarTrackerComponent)(world.Get(capital, warCompID))
	if warComp.TargetCountryID != 3 || !warComp.Active {
		t.Fatalf("expected re-targeted war on country 3, got target=%d active=%v",
			warComp.TargetCountryID, warComp.Active)
	}

	// Peace flips Active off but keeps the component archetype stable.
	MakePeace(&world, capital)
	if !world.Has(capital, warCompID) {
		t.Fatalf("MakePeace must keep WarTrackerComponent attached")
	}
	warComp = (*components.WarTrackerComponent)(world.Get(capital, warCompID))
	if warComp.Active {
		t.Fatalf("expected war inactive after MakePeace")
	}
	if warComp.TargetCountryID != 3 {
		t.Fatalf("MakePeace must not erase last war target, got %d", warComp.TargetCountryID)
	}

	// MakePeace on a capital without a tracker is a safe no-op.
	other := newLawTestCapital(&world, 4)
	MakePeace(&world, other)
	if world.Has(other, warCompID) {
		t.Fatalf("MakePeace must not add WarTrackerComponent")
	}

	// Dead entity guards.
	world.RemoveEntity(other)
	MakePeace(&world, other) // must not panic
	if err := DeclareWar(&world, other, 2); err == nil {
		t.Fatalf("expected error declaring war from a dead capital")
	}
}
