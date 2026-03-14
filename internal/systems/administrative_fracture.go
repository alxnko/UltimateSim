package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 16.4: Administrative Reach & Friction
// AdministrativeFractureSystem evaluates the physical distance of all sub-cities
// from their Country Capital. If a village exceeds the MaxAdministrativeRange,
// it unilaterally withdraws from the Country to save on tax costs.

const (
	AdministrativeFractureTickRate = 1000 // Perform analysis every 1000 ticks
	MaxAdministrativeRange         = 150.0 // Maximum allowed distance from capital
)

type AdministrativeFractureSystem struct {
	world     *ecs.World
	tickStamp uint64

	// Component IDs cached
	villageID ecs.ID
	affilID   ecs.ID
	posID     ecs.ID
	countryID ecs.ID
	capitalID ecs.ID
	jurID     ecs.ID
}

// NewAdministrativeFractureSystem initializes the administrative fracture evaluation.
func NewAdministrativeFractureSystem(world *ecs.World) *AdministrativeFractureSystem {
	return &AdministrativeFractureSystem{
		world:     world,
		tickStamp: 0,
		villageID: ecs.ComponentID[components.Village](world),
		affilID:   ecs.ComponentID[components.Affiliation](world),
		posID:     ecs.ComponentID[components.Position](world),
		countryID: ecs.ComponentID[components.CountryComponent](world),
		capitalID: ecs.ComponentID[components.CapitalComponent](world),
		jurID:     ecs.ComponentID[components.JurisdictionComponent](world),
	}
}

type capitalData struct {
	pos        *components.Position
	corruption uint32
}

// Update executes the core loop.
func (s *AdministrativeFractureSystem) Update(world *ecs.World) {
	s.tickStamp++

	// Only run analysis periodically to save CPU
	if s.tickStamp%AdministrativeFractureTickRate != 0 {
		return
	}

	// 1. Build a flat map of Capital data for O(1) distance matching
	capitalDataMap := make(map[uint32]capitalData)

	capitalQuery := s.world.Query(filter.All(s.capitalID, s.countryID, s.posID, s.affilID))
	for capitalQuery.Next() {
		affil := (*components.Affiliation)(capitalQuery.Get(s.affilID))
		pos := (*components.Position)(capitalQuery.Get(s.posID))

		var corruption uint32 = 0
		if s.world.Has(capitalQuery.Entity(), s.jurID) {
			jur := (*components.JurisdictionComponent)(capitalQuery.Get(s.jurID))
			corruption = jur.Corruption
		}

		capitalDataMap[affil.CountryID] = capitalData{
			pos:        pos,
			corruption: corruption,
		}
	}

	maxDistSq := float32(MaxAdministrativeRange * MaxAdministrativeRange)

	// 2. Iterate over all Villages with Affiliation and Position
	villageQuery := s.world.Query(filter.All(s.villageID, s.affilID, s.posID))
	for villageQuery.Next() {
		affil := (*components.Affiliation)(villageQuery.Get(s.affilID))

		// If the village belongs to a Country
		if affil.CountryID != 0 {
			if capData, exists := capitalDataMap[affil.CountryID]; exists {
				pos := (*components.Position)(villageQuery.Get(s.posID))

				// Calculate distance squared to capital
				dx := pos.X - capData.pos.X
				dy := pos.Y - capData.pos.Y
				distSq := dx*dx + dy*dy

				// Phase 22.1: The Corruption Engine
				effectiveDistSq := distSq * (1.0 + float32(capData.corruption)*0.1)

				// Fracture Logic: Unilaterally withdraw from Union/Country
				if effectiveDistSq > maxDistSq {
					affil.CountryID = 0
				}
			} else {
				// Capital doesn't exist anymore, Country is broken
				affil.CountryID = 0
			}
		}
	}
}
