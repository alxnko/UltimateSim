package systems

import (
	"testing"

	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

func TestNewCalendarSystem(t *testing.T) {
	cal := engine.NewCalendar()
	sys := NewCalendarSystem(cal)

	if sys == nil {
		t.Fatal("NewCalendarSystem returned nil")
	}
	if sys.calendar != cal {
		t.Errorf("expected calendar %p, got %p", cal, sys.calendar)
	}
}

func TestCalendarSystem_Update(t *testing.T) {
	world := ecs.NewWorld()
	cal := engine.NewCalendar()
	sys := NewCalendarSystem(cal)

	// Test 1: Nil calendar doesn't panic
	sysNil := NewCalendarSystem(nil)
	sysNil.Update(&world)

	// Test 2: Ticks increment
	sys.Update(&world)
	if cal.Ticks != 1 {
		t.Errorf("expected Ticks 1, got %d", cal.Ticks)
	}

	// Test 3: Day increment
	// TicksPerDay is 600. Ticks is currently 1.
	for i := 0; i < engine.TicksPerDay-1; i++ {
		sys.Update(&world)
	}
	if cal.Ticks != uint64(engine.TicksPerDay) {
		t.Errorf("expected Ticks %d, got %d", engine.TicksPerDay, cal.Ticks)
	}
	if cal.Days != 2 {
		t.Errorf("expected Day 2, got %d", cal.Days)
	}

	// Test 4: Month increment and IsWinter toggle
	// Currently Ticks = 600, Days = 2, Months = 1, Years = 1, IsWinter = false.
	// To get to next month: need to finish Day 6 and hit tick 3600.
	// We need 3600 - 600 = 3000 more ticks.
	for i := 0; i < 5*engine.TicksPerDay; i++ {
		sys.Update(&world)
	}
	if cal.Ticks != uint64(6*engine.TicksPerDay) {
		t.Errorf("expected Ticks %d, got %d", 6*engine.TicksPerDay, cal.Ticks)
	}
	if cal.Months != 2 {
		t.Errorf("expected Month 2, got %d", cal.Months)
	}
	if cal.Days != 1 {
		t.Errorf("expected Day 1, got %d", cal.Days)
	}
	if !cal.IsWinter {
		t.Errorf("expected IsWinter to be true at Ticks %d", cal.Ticks)
	}

	// Test 5: Year increment
	// Months is 2. Need to finish Month 4 and hit next year boundary.
	// Months 2, 3, 4 are 3600 ticks each.
	// Ticks currently 3600. We need 3 * 3600 = 10800 more ticks to reach Ticks = 14400.
	for i := 0; i < 3*engine.SeasonDuration; i++ {
		sys.Update(&world)
	}
	if cal.Ticks != 14400 {
		t.Errorf("expected Ticks 14400, got %d", cal.Ticks)
	}
	if cal.Years != 2 {
		t.Errorf("expected Year 2, got %d", cal.Years)
	}
	if cal.Months != 1 {
		t.Errorf("expected Month 1, got %d", cal.Months)
	}

	// Test 6: IsWinter toggled again (F->T->F->T->F)
	// Toggles at 3600 (T), 7200 (F), 10800 (T), 14400 (F)
	if cal.IsWinter {
		t.Errorf("expected IsWinter to be false at Ticks %d", cal.Ticks)
	}
}
