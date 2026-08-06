package systems

import (
	"errors"
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Grand Strategy Phase (spec P2.5): council tests.
// Deterministic, headless coverage of Appoint validation, the Steward's
// periodic treasury bonus in a real ticked world, seat vacancy on death, and
// the GetCouncilBonus read API.

// spawnCapital creates a country capital entity: CapitalComponent +
// CountryComponent + Affiliation(CountryID) + TreasuryComponent.
func spawnCapital(world *ecs.World, countryID uint32, wealth float32) ecs.Entity {
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	countryCompID := ecs.ComponentID[components.CountryComponent](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](world)

	e := world.NewEntity(capitalID, countryCompID, affID, treasuryID)
	(*components.Affiliation)(world.Get(e, affID)).CountryID = countryID
	(*components.TreasuryComponent)(world.Get(e, treasuryID)).Wealth = wealth
	return e
}

// spawnCitizen creates an NPC of the given country.
func spawnCitizen(world *ecs.World, identityID uint64, countryID uint32) ecs.Entity {
	e := spawnDynastyNPC(world, identityID, 1, 30)
	affID := ecs.ComponentID[components.Affiliation](world)
	(*components.Affiliation)(world.Get(e, affID)).CountryID = countryID
	return e
}

// TestAppointValidation verifies capital, seat, citizenship, and
// one-seat-per-NPC rules.
func TestAppointValidation(t *testing.T) {
	world := ecs.NewWorld()

	// No capital for country 5 yet.
	if err := Appoint(&world, 5, components.SeatSteward, 10); !errors.Is(err, ErrNoCapital) {
		t.Errorf("no capital: err = %v, want ErrNoCapital", err)
	}

	capital := spawnCapital(&world, 5, 0)
	spawnCitizen(&world, 10, 5)
	spawnCitizen(&world, 11, 9) // Foreigner

	if err := Appoint(&world, 5, 0, 10); !errors.Is(err, ErrInvalidSeat) {
		t.Errorf("invalid seat: err = %v, want ErrInvalidSeat", err)
	}
	if err := Appoint(&world, 5, components.SeatSteward, 999); !errors.Is(err, ErrCouncilorNotFound) {
		t.Errorf("unknown NPC: err = %v, want ErrCouncilorNotFound", err)
	}
	if err := Appoint(&world, 5, components.SeatSteward, 11); !errors.Is(err, ErrForeignCouncilor) {
		t.Errorf("foreign NPC: err = %v, want ErrForeignCouncilor", err)
	}

	if err := Appoint(&world, 5, components.SeatSteward, 10); err != nil {
		t.Fatalf("valid appointment failed: %v", err)
	}
	councilID := ecs.ComponentID[components.CouncilComponent](&world)
	council := (*components.CouncilComponent)(world.Get(capital, councilID))
	if council.Steward != 10 {
		t.Errorf("Steward = %d, want 10", council.Steward)
	}

	// Moving the same NPC to another seat vacates the old one.
	if err := Appoint(&world, 5, components.SeatSpymaster, 10); err != nil {
		t.Fatalf("re-appointment failed: %v", err)
	}
	council = (*components.CouncilComponent)(world.Get(capital, councilID))
	if council.Steward != 0 || council.Spymaster != 10 {
		t.Errorf("council = {Steward:%d Spymaster:%d}, want {0 10}", council.Steward, council.Spymaster)
	}
}

// TestCouncilStewardIncomeAndVacancy ticks a real world: the Steward grows the
// treasury by StewardIncomePct per cycle, and a dead holder is vacated with no
// further bonus.
func TestCouncilStewardIncomeAndVacancy(t *testing.T) {
	world := ecs.NewWorld()

	capital := spawnCapital(&world, 5, 100)
	steward := spawnCitizen(&world, 10, 5)
	if err := Appoint(&world, 5, components.SeatSteward, 10); err != nil {
		t.Fatalf("Appoint failed: %v", err)
	}

	sys := NewCouncilSystem(&world)
	treasuryID := ecs.ComponentID[components.TreasuryComponent](&world)

	// One tick short of the cycle: nothing happens.
	for i := 0; i < CouncilTickRate-1; i++ {
		sys.Update(&world)
	}
	if w := (*components.TreasuryComponent)(world.Get(capital, treasuryID)).Wealth; w != 100 {
		t.Fatalf("treasury moved early: %v, want 100", w)
	}

	// The cycle tick applies +5%.
	sys.Update(&world)
	if w := (*components.TreasuryComponent)(world.Get(capital, treasuryID)).Wealth; w != 105 {
		t.Fatalf("treasury = %v after steward cycle, want 105", w)
	}

	// The steward dies: seat vacated, no further income.
	world.RemoveEntity(steward)
	for i := 0; i < CouncilTickRate; i++ {
		sys.Update(&world)
	}
	councilID := ecs.ComponentID[components.CouncilComponent](&world)
	council := (*components.CouncilComponent)(world.Get(capital, councilID))
	if council.Steward != 0 {
		t.Errorf("dead steward still seated: %d, want 0", council.Steward)
	}
	if w := (*components.TreasuryComponent)(world.Get(capital, treasuryID)).Wealth; w != 105 {
		t.Errorf("treasury = %v after steward death, want 105 (no bonus)", w)
	}
}

// TestGetCouncilBonus verifies the per-seat bonus read API that diplomacy,
// economy, and plots consume.
func TestGetCouncilBonus(t *testing.T) {
	world := ecs.NewWorld()

	// No capital at all: every bonus is 0.
	if got := GetCouncilBonus(&world, 5, components.SeatSteward); got != 0 {
		t.Errorf("bonus without capital = %d, want 0", got)
	}

	spawnCapital(&world, 5, 0)
	spawnCitizen(&world, 10, 5)
	spawnCitizen(&world, 11, 5)
	spawnCitizen(&world, 12, 5)
	spawnCitizen(&world, 13, 5)

	// Vacant seats yield 0.
	if got := GetCouncilBonus(&world, 5, components.SeatMarshal); got != 0 {
		t.Errorf("vacant marshal bonus = %d, want 0", got)
	}

	for _, tc := range []struct {
		seat  uint8
		npc   uint64
		bonus int
	}{
		{components.SeatSteward, 10, StewardIncomePct},
		{components.SeatMarshal, 11, MarshalWarScoreBonus},
		{components.SeatDiplomat, 12, DiplomatOpinionDrift},
		{components.SeatSpymaster, 13, SpymasterDiscoveryBonus},
	} {
		if err := Appoint(&world, 5, tc.seat, tc.npc); err != nil {
			t.Fatalf("Appoint(seat %d) failed: %v", tc.seat, err)
		}
		if got := GetCouncilBonus(&world, 5, tc.seat); got != tc.bonus {
			t.Errorf("bonus(seat %d) = %d, want %d", tc.seat, got, tc.bonus)
		}
	}
}
