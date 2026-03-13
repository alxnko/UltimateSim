package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.4: Administrative Reach & Friction
// FractureLogicSystem evaluates if a sub-city is too far from its Country's Capital
// to benefit from the macro-state (e.g., Defense Pacts). If the distance exceeds
// MaxAdministrativeReach, the city unilaterally withdraws from the Union/Country to save on tax costs.

const (
	FractureLogicTickRate    = 1000 // Perform analysis every 1000 ticks
	MaxAdministrativeReach   = 150  // Maximum distance before administrative friction forces a fracture
)

type FractureLogicSystem struct {
	world     *ecs.World
	tickStamp uint64

	// Pre-allocated DOD map for O(1) Capital lookup by CountryID
	capitals map[uint32]*components.Position
}

// NewFractureLogicSystem initializes the fracture logic system.
func NewFractureLogicSystem(world *ecs.World) *FractureLogicSystem {
	return &FractureLogicSystem{
		world:     world,
		tickStamp: 0,
		capitals:  make(map[uint32]*components.Position, 50), // Pre-allocated capacity
	}
}

// Update executes the fracture evaluation logic.
func (s *FractureLogicSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Only run deep analysis periodically to save CPU
	if s.tickStamp%FractureLogicTickRate != 0 {
		return
	}

	countryID := ecs.ComponentID[components.CountryComponent](world)
	capitalID := ecs.ComponentID[components.CapitalComponent](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	posID := ecs.ComponentID[components.Position](world)
	villageID := ecs.ComponentID[components.Village](world)

	// Step 1: Build O(1) lookup map of Capital Positions
	// Clear the map but retain its underlying memory allocation
	clear(s.capitals)

	capitalQuery := world.Query(filter.All(countryID, capitalID, affilID, posID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(affilID))
		pos := (*components.Position)(capitalQuery.Get(posID))

		s.capitals[affil.CountryID] = pos
	}

	// Step 2: Iterate over all Villages (sub-cities) to calculate distance to their Capital
	villageFilter := filter.All(villageID, affilID, posID).Without(capitalID)
	villageQuery := world.Query(&villageFilter)

	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(affilID))

		// If the village is not part of a Country, skip
		if affil.CountryID == 0 {
			continue
		}

		// Find the Capital's position
		if capitalPos, exists := s.capitals[affil.CountryID]; exists {
			pos := (*components.Position)(villageQuery.Get(posID))

			// Distance Check (Basic math to preserve determinism & DOD speed)
			dx := pos.X - capitalPos.X
			dy := pos.Y - capitalPos.Y
			distSq := dx*dx + dy*dy

			// If distance exceeds administrative reach, fracture!
			if distSq > MaxAdministrativeReach*MaxAdministrativeReach {
				// Withdraw from the macro-state
				affil.CountryID = 0
			}
		} else {
			// If the CountryID is invalid (no capital found), fracture out of safety
			affil.CountryID = 0
		}
	}
}
