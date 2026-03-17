package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 40.2 - The Ruins Resettlement Engine
// Homeless or wandering NPCs (Affiliation.CityID == 0) naturally seek out and claim
// abandoned RuinComponent entities. If they idle at the location for long enough,
// they clear the RuinComponent, restore the Village and Needs components, and functionally
// bring the dead settlement back to life, inheriting residual resources and linking their
// CityID to the reborn village.

type RuinResettlementSystem struct {
	tickCounter uint64

	// Component IDs
	npcID     ecs.ID
	posID     ecs.ID
	slID      ecs.ID
	affID     ecs.ID
	ruinID    ecs.ID
	villageID ecs.ID
	storageID ecs.ID
	popID     ecs.ID
	marketID  ecs.ID
}

func NewRuinResettlementSystem(world *ecs.World) *RuinResettlementSystem {
	return &RuinResettlementSystem{
		tickCounter: 0,
		npcID:       ecs.ComponentID[components.NPC](world),
		posID:       ecs.ComponentID[components.Position](world),
		slID:        ecs.ComponentID[components.SettlementLogic](world),
		affID:       ecs.ComponentID[components.Affiliation](world),
		ruinID:      ecs.ComponentID[components.RuinComponent](world),
		villageID:   ecs.ComponentID[components.Village](world),
		storageID:   ecs.ComponentID[components.StorageComponent](world),
		popID:       ecs.ComponentID[components.PopulationComponent](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
	}
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *RuinResettlementSystem) IsExpensive() bool {
	return true
}

func (s *RuinResettlementSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Only evaluate occasionally to save CPU cycles
	if s.tickCounter%50 != 0 {
		return
	}

	// Phase 1: Pre-cache active Ruins into a flat DOD slice
	ruinQuery := world.Query(ecs.All(s.ruinID, s.posID))

	type ruinData struct {
		entity ecs.Entity
		x      float32
		y      float32
	}
	ruins := make([]ruinData, 0, 100)

	for ruinQuery.Next() {
		pos := (*components.Position)(ruinQuery.Get(s.posID))
		ruins = append(ruins, ruinData{
			entity: ruinQuery.Entity(),
			x:      pos.X,
			y:      pos.Y,
		})
	}

	if len(ruins) == 0 {
		return
	}

	// Phase 2: Find homeless NPCs attempting to settle
	npcQuery := world.Query(ecs.All(s.npcID, s.posID, s.slID, s.affID))

	type resettlementAction struct {
		npcEnt  ecs.Entity
		ruinEnt ecs.Entity
	}
	actions := make([]resettlementAction, 0, 10)

	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))

		// Ensure NPC is homeless
		if aff.CityID != 0 {
			continue
		}

		sl := (*components.SettlementLogic)(npcQuery.Get(s.slID))

		// If they have been idling (or are close to triggering a new settlement)
		if sl.TicksAtZeroVelocity >= 500 { // Resettling is faster (500) than starting from scratch (1000)
			pos := (*components.Position)(npcQuery.Get(s.posID))

			// Check if they are standing on top of a Ruin
			for _, r := range ruins {
				dx := pos.X - r.x
				dy := pos.Y - r.y
				distSq := dx*dx + dy*dy

				if distSq < 1.0 {
					// Natively discovered!
					actions = append(actions, resettlementAction{
						npcEnt:  npcQuery.Entity(),
						ruinEnt: r.entity,
					})
					break
				}
			}
		}
	}

	// Phase 3: Execute Resettlements outside of queries
	for _, action := range actions {
		if !world.Alive(action.ruinEnt) || !world.Alive(action.npcEnt) || !world.Has(action.ruinEnt, s.ruinID) {
			continue
		}

		// Remove Ruin Component
		world.Remove(action.ruinEnt, s.ruinID)

		// Restore Settlement Components
		if !world.Has(action.ruinEnt, s.villageID) {
			world.Add(action.ruinEnt, s.villageID)
		}

		if !world.Has(action.ruinEnt, s.popID) {
			world.Add(action.ruinEnt, s.popID)
		}
		pop := (*components.PopulationComponent)(world.Get(action.ruinEnt, s.popID))
		pop.Count = 1

		if !world.Has(action.ruinEnt, s.storageID) {
			world.Add(action.ruinEnt, s.storageID)
		}
		storage := (*components.StorageComponent)(world.Get(action.ruinEnt, s.storageID))
		// Reclaimed ruins offer a slight foundational bonus over fresh settlements
		storage.Wood += 50
		storage.Stone += 100
		storage.Food += 50

		if !world.Has(action.ruinEnt, s.marketID) {
			world.Add(action.ruinEnt, s.marketID)
			market := (*components.MarketComponent)(world.Get(action.ruinEnt, s.marketID))
			market.FoodPrice = 1.0
			market.WoodPrice = 1.0
			market.StonePrice = 1.0
			market.IronPrice = 1.0
		}

		// Re-assign Needs if missing (Ruins lose Needs to avoid Metabolism loops)
		needsID := ecs.ComponentID[components.Needs](world)
		if !world.Has(action.ruinEnt, needsID) {
			world.Add(action.ruinEnt, needsID)
		}

		// Add City Affiliation tracking
		if !world.Has(action.ruinEnt, s.affID) {
			world.Add(action.ruinEnt, s.affID)
		}
		ruinAff := (*components.Affiliation)(world.Get(action.ruinEnt, s.affID))

		// The ruin needs a valid CityID. We can assign it a new ID based on entity ID
		ruinCityID := uint32(action.ruinEnt.ID() + 10000)
		ruinAff.CityID = ruinCityID

		// Link the homeless NPC to this new reborn city
		npcAff := (*components.Affiliation)(world.Get(action.npcEnt, s.affID))
		npcAff.CityID = ruinCityID

		// Reset their settlement logic
		sl := (*components.SettlementLogic)(world.Get(action.npcEnt, s.slID))
		sl.TicksAtZeroVelocity = 0
	}
}
