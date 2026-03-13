package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 03.3: MetabolismSystem
// Evaluates all valid NeedsComponent payloads and subtracts dynamic rate variables.
// Deducts Needs.Food based on a formula like: Food -= 0.05 * GeneticHealthModifier

type MetabolismSystem struct {
	filter   ecs.Filter
	calendar *engine.Calendar
}

// NewMetabolismSystem creates a new MetabolismSystem.
func NewMetabolismSystem(world *ecs.World, calendar *engine.Calendar) *MetabolismSystem {
	// Query entities that have both Needs and Genetics
	needsID := ecs.ComponentID[components.Needs](world)
	geneticsID := ecs.ComponentID[components.GenomeComponent](world)
	ruinID := ecs.ComponentID[components.RuinComponent](world)

	// Phase 05.3: Arche-Go Component Filters
	// Explicitly build Without(ruinID) to skip over ruins.
	mask := ecs.All(needsID, geneticsID).Without(ruinID)

	return &MetabolismSystem{
		filter:   &mask,
		calendar: calendar,
	}
}

// Update executes the system logic per tick.
func (s *MetabolismSystem) Update(world *ecs.World) {
	needsID := ecs.ComponentID[components.Needs](world)
	geneticsID := ecs.ComponentID[components.GenomeComponent](world)

	query := world.Query(s.filter)
	for query.Next() {
		needs := (*components.Needs)(query.Get(needsID))
		genetics := (*components.GenomeComponent)(query.Get(geneticsID))

		// Calculate health modifier. Health is 0-100.
		// Higher health means slower metabolism (less food deducted).
		// Modifier ranges from 1.0 (at 100 health) to 2.0 (at 0 health)
		// Math: base_rate * (2.0 - health/100.0)
		healthModifier := 2.0 - (float32(genetics.Health) / 100.0)
		if healthModifier < 0.1 {
			healthModifier = 0.1
		}

		// Phase 13.4: The Seasonal Pulse
		// Mutably scales the NeedsComponent decay matrices (1.5x calorie burn rates globally)
		multiplier := float32(1.0)
		if s.calendar != nil && s.calendar.IsWinter {
			multiplier = 1.5
		}

		// Deduct food
		needs.Food -= (0.05 * healthModifier) * multiplier

		// Ensure it doesn't drop below 0
		if needs.Food < 0 {
			needs.Food = 0
		}

		// Optional: We could deduct rest and safety too, but sticking to Phase 3.3 Food logic
	}
}
