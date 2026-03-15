package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 29.1: Geopolitical Resource Wars
// ResourceWarSystem causes starving Countries to declare war on wealthy neighbors.
// It bridges macro-economics to the Justice and Blood Feud engines by distributing
// massive negative hooks to all citizens of the invading country against the target country.

type capitalWarData struct {
	entity    ecs.Entity
	countryID uint32
	x         float32
	y         float32
	foodPrice float32
	foodStock uint32
	hasWar    bool
}

type ResourceWarSystem struct {
	tickCounter uint64
	hooks       *engine.SparseHookGraph

	// Component IDs
	posID       ecs.ID
	affID       ecs.ID
	capID       ecs.ID
	marketID    ecs.ID
	storageID   ecs.ID
	warCompID   ecs.ID
	npcID       ecs.ID
	identID     ecs.ID
}

func NewResourceWarSystem(world *ecs.World, hooks *engine.SparseHookGraph) *ResourceWarSystem {
	return &ResourceWarSystem{
		hooks:       hooks,
		posID:       ecs.ComponentID[components.Position](world),
		affID:       ecs.ComponentID[components.Affiliation](world),
		capID:       ecs.ComponentID[components.CapitalComponent](world),
		marketID:    ecs.ComponentID[components.MarketComponent](world),
		storageID:   ecs.ComponentID[components.StorageComponent](world),
		warCompID:   ecs.ComponentID[components.WarTrackerComponent](world),
		npcID:       ecs.ComponentID[components.NPC](world),
		identID:     ecs.ComponentID[components.Identity](world),
	}
}

func (s *ResourceWarSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Only evaluate macro-politics occasionally
	if s.tickCounter%500 != 0 {
		return
	}

	// 1. Gather all Capital data into a flat slice
	capQuery := world.Query(ecs.All(s.capID, s.posID, s.affID, s.marketID, s.storageID))
	var capitals []capitalWarData

	for capQuery.Next() {
		pos := (*components.Position)(capQuery.Get(s.posID))
		aff := (*components.Affiliation)(capQuery.Get(s.affID))
		market := (*components.MarketComponent)(capQuery.Get(s.marketID))
		storage := (*components.StorageComponent)(capQuery.Get(s.storageID))

		// Check if already at war
		hasWar := world.Has(capQuery.Entity(), s.warCompID)

		// If at war, ensure we only attack active wars, but for simplicity
		// we just skip initiating new wars if one is already active.

		capitals = append(capitals, capitalWarData{
			entity:    capQuery.Entity(),
			countryID: aff.CountryID,
			x:         pos.X,
			y:         pos.Y,
			foodPrice: market.FoodPrice,
			foodStock: storage.Food,
			hasWar:    hasWar,
		})
	}

	if len(capitals) < 2 {
		return // Cannot have a war with less than two countries
	}

	// 2. Evaluate Starvation vs Wealth
	// We want to avoid nested queries or modifying the world during this loop.
	var newWars []struct{ Attacker, Defender capitalWarData }

	for i := 0; i < len(capitals); i++ {
		attacker := capitals[i]

		// Must not already be at war, and must be starving (extreme famine)
		if attacker.hasWar || attacker.foodPrice < 8.0 {
			continue
		}

		for j := 0; j < len(capitals); j++ {
			if i == j {
				continue
			}

			defender := capitals[j]

			// Defender must be wealthy
			if defender.foodStock < 1000 {
				continue
			}

			// Check distance. Geopolitical distance threshold (RadiusSquared 2500 = 50 tiles)
			dx := attacker.x - defender.x
			dy := attacker.y - defender.y
			distSq := dx*dx + dy*dy

			if distSq <= 2500.0 {
				// Declare War!
				newWars = append(newWars, struct{ Attacker, Defender capitalWarData }{attacker, defender})

				// Mark attacker as having a war to prevent multiple war declarations in the same loop
				capitals[i].hasWar = true
				break // One war per starving country
			}
		}
	}

	if len(newWars) == 0 {
		return
	}

	// 3. Apply structural changes and distribute Hooks (The Butterfly Effect)
	// We need to map NPCs to their CountryID for fast distribution
	type npcData struct {
		id        uint64
		countryID uint32
	}

	npcQuery := world.Query(ecs.All(s.npcID, s.identID, s.affID))
	var npcs []npcData
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(s.identID))
		aff := (*components.Affiliation)(npcQuery.Get(s.affID))

		if aff.CountryID != 0 {
			npcs = append(npcs, npcData{
				id:        ident.ID,
				countryID: aff.CountryID,
			})
		}
	}

	for _, war := range newWars {
		// Add WarTrackerComponent to the Attacker Capital
		if !world.Has(war.Attacker.entity, s.warCompID) {
			world.Add(war.Attacker.entity, s.warCompID)
			warComp := (*components.WarTrackerComponent)(world.Get(war.Attacker.entity, s.warCompID))
			warComp.TargetCountryID = war.Defender.countryID
			warComp.Active = true
		}

		// Cross-reference attackers against defenders and seed Hooks
		// This mathematically forces BloodFeudSystem to trigger
		for _, attackerNPC := range npcs {
			if attackerNPC.countryID == war.Attacker.countryID {
				for _, defenderNPC := range npcs {
					if defenderNPC.countryID == war.Defender.countryID {
						// Add a massive -100 hook: attacker NPC hates defender NPC
						if s.hooks != nil {
							s.hooks.AddHook(attackerNPC.id, defenderNPC.id, -100)
						}
					}
				}
			}
		}
	}
}
