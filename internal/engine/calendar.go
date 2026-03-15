package engine

// Phase 13.4: The Seasonal Pulse
// Calendar tracks global ticks and the IsWinter boolean to mathematically influence
// the simulation bounds rules across decoupled systems.

const SeasonDuration = 3600 // Ticks per season. At 60 TPS, this equals 60 seconds of real-time simulation.
const TicksPerDay = 600     // 10 seconds per day. 6 days = 1 season

// Calendar holds the seasonal state of the game world.
// It is passed as a pointer to Systems to ensure Data-Oriented shared state reading.
type Calendar struct {
	Ticks    uint64
	Days     uint64
	Months   uint64 // Conceptually treating a season as a month
	Years    uint64
	IsWinter bool
}

// NewCalendar creates a new initialized Calendar.
func NewCalendar() *Calendar {
	return &Calendar{
		Ticks:    0,
		Days:     1,
		Months:   1,
		Years:    1,
		IsWinter: false, // Start in Summer/Spring
	}
}
