package engine

// Phase 13.4: The Seasonal Pulse
// Calendar tracks global ticks and the IsWinter boolean to mathematically influence
// the simulation bounds rules across decoupled systems.

const SeasonDuration = 3600 // Ticks per season. At 60 TPS, this equals 60 seconds of real-time simulation.

// Calendar holds the seasonal state of the game world.
// It is passed as a pointer to Systems to ensure Data-Oriented shared state reading.
type Calendar struct {
	Ticks    uint64
	IsWinter bool
}

// NewCalendar creates a new initialized Calendar.
func NewCalendar() *Calendar {
	return &Calendar{
		Ticks:    0,
		IsWinter: false, // Start in Summer/Spring
	}
}
