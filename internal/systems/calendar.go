package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 13.4: The Seasonal Pulse
// CalendarSystem is a global tick modifier affecting the primary bounds rules.

// CalendarSystem increments the global tick counter and toggles the IsWinter boolean.
type CalendarSystem struct {
	calendar *engine.Calendar
}

// NewCalendarSystem creates a new CalendarSystem bound to the shared global Calendar.
func NewCalendarSystem(calendar *engine.Calendar) *CalendarSystem {
	return &CalendarSystem{
		calendar: calendar,
	}
}

// Update evaluates seasonal state per tick.
func (s *CalendarSystem) Update(world *ecs.World) {
	if s.calendar == nil {
		return
	}

	s.calendar.Ticks++

	// Toggle season exactly on boundaries.
	if s.calendar.Ticks%engine.SeasonDuration == 0 {
		s.calendar.IsWinter = !s.calendar.IsWinter
	}
}
