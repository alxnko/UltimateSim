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
	tm       *engine.TickManager
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *MetabolismSystem) IsExpensive() bool {
	return true
}

// NewMetabolismSystem creates a new MetabolismSystem.
func NewMetabolismSystem(world *ecs.World, calendar *engine.Calendar, tm *engine.TickManager) *MetabolismSystem {
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
		tm:       tm,
	}
}

// Update executes the system logic per tick.
func (s *MetabolismSystem) Update(world *ecs.World) {
	needsID := ecs.ComponentID[components.Needs](world)
	geneticsID := ecs.ComponentID[components.GenomeComponent](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	popID := ecs.ComponentID[components.PopulationComponent](world)
	storageID := ecs.ComponentID[components.StorageComponent](world)

	// 1. Process active NPCs (entities with Needs)
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

		// Phase 19.4: Advanced Biology (The Butterfly Effect: Starvation -> Pain -> Collapse)
		if query.Has(vitalsID) {
			vitals := (*components.VitalsComponent)(query.Get(vitalsID))

			if needs.Food == 0 {
				vitals.Pain += 0.5
			} else {
				// Recover pain slowly if fed
				vitals.Pain -= 0.1
				if vitals.Pain < 0 {
					vitals.Pain = 0
				}
			}

			if vitals.Pain > 50.0 {
				vitals.Consciousness -= 0.5
				if vitals.Consciousness < 0 {
					vitals.Consciousness = 0
				}
			} else {
				// Recover consciousness slowly if pain is low
				vitals.Consciousness += 0.2
				if vitals.Consciousness > 100.0 {
					vitals.Consciousness = 100.0
				}
			}
		}
	}

	// 2. Process abstracted Citizens in Settlements (Village Metabolism)
	popFilter := ecs.All(popID, storageID)
	popQuery := world.Query(popFilter)

	multiplier := float32(1.0)
	if s.calendar != nil && s.calendar.IsWinter {
		multiplier = 1.5
	}

	for popQuery.Next() {
		pop := (*components.PopulationComponent)(popQuery.Get(popID))
		storage := (*components.StorageComponent)(popQuery.Get(storageID))

		// Each citizen consumes food. Total consumption based on headcount.
		foodNeeded := float32(len(pop.Citizens)) * 0.05 * multiplier

		if float32(storage.Food) >= foodNeeded {
			storage.Food -= uint32(foodNeeded)
		} else {
			// Starvation: Food storage is empty.
			storage.Food = 0
			// Random death chance during starvation (1% per tick during starvation)
			if len(pop.Citizens) > 0 && engine.GetRandomInt()%100 == 0 {
				pop.Count--
				// Remove random citizen from the abstracted pool
				idx := engine.GetRandomInt() % len(pop.Citizens)
				pop.Citizens[idx] = pop.Citizens[len(pop.Citizens)-1]
				pop.Citizens = pop.Citizens[:len(pop.Citizens)-1]
			}
		}
	}
}
