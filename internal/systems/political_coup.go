package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 51 - The Debt-Trap Political Coup Engine
// PoliticalCoupSystem continuously maps the negative hooks of active Administrators.
// If an Administrator owes extreme "Debt Hooks" (e.g., massive negative value)
// to a single entity, that entity executes the collateral, staging a bloodless coup
// and forcibly taking the AdministrationMarker.

type coupNodeData struct {
	Entity ecs.Entity
	ID     uint64
	CityID uint32
}

type PoliticalCoupSystem struct {
	hooks        *engine.SparseHookGraph
	tickCounter  uint64
	activeRulers []coupNodeData
	allNPCs      []coupNodeData
	toRemove     []ecs.Entity
	toAdd        []ecs.Entity
}

// NewPoliticalCoupSystem creates a new PoliticalCoupSystem.
func NewPoliticalCoupSystem(hooks *engine.SparseHookGraph) *PoliticalCoupSystem {
	return &PoliticalCoupSystem{
		hooks:        hooks,
		activeRulers: make([]coupNodeData, 0, 50),
		allNPCs:      make([]coupNodeData, 0, 500),
		toRemove:     make([]ecs.Entity, 0, 10),
		toAdd:        make([]ecs.Entity, 0, 10),
	}
}

// Update evaluates political debt leverage.
func (s *PoliticalCoupSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle to run every 100 ticks to avoid O(N^2) overhead per frame.
	if s.tickCounter%100 != 0 {
		return
	}

	if s.hooks == nil {
		return
	}

	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](world)

	s.activeRulers = s.activeRulers[:0]
	s.allNPCs = s.allNPCs[:0]
	s.toRemove = s.toRemove[:0]
	s.toAdd = s.toAdd[:0]

	// 1. Gather all NPCs and flag active rulers
	query := world.Query(filter.All(npcID, identID, affilID))

	for query.Next() {
		ident := (*components.Identity)(query.Get(identID))
		affil := (*components.Affiliation)(query.Get(affilID))

		if affil.CityID == 0 {
			continue // Skip homeless
		}

		node := coupNodeData{
			Entity: query.Entity(),
			ID:     ident.ID,
			CityID: affil.CityID,
		}

		s.allNPCs = append(s.allNPCs, node)

		if world.Has(query.Entity(), adminMarkerID) {
			s.activeRulers = append(s.activeRulers, node)
		}
	}

	// 2. Evaluate vulnerability of active Rulers
	for i := 0; i < len(s.activeRulers); i++ {
		ruler := s.activeRulers[i]

		// The ruler's incoming hooks represent people's opinions of them.
		// A massive negative hook from a single source represents leveraged debt/blackmail.
		incomingHooks := s.hooks.GetAllIncomingHooks(ruler.ID)

		var bestUsurperID uint64 = 0
		var worstGrudge int = 0 // Negative values

		for originID, hookVal := range incomingHooks {
			if hookVal <= -200 { // The catastrophic debt threshold
				if hookVal < worstGrudge {
					worstGrudge = hookVal
					bestUsurperID = originID
				}
			}
		}

		// 3. Execute the Coup
		if bestUsurperID != 0 {
			// Find the usurper entity in the same city
			var usurperEntity ecs.Entity
			found := false

			for j := 0; j < len(s.allNPCs); j++ {
				npc := s.allNPCs[j]
				if npc.ID == bestUsurperID {
					// The usurper must reside in the same city to actually seize local power.
					if npc.CityID == ruler.CityID {
						usurperEntity = npc.Entity
						found = true
					}
					break
				}
			}

			if found {
				// Stage the structural changes
				s.toRemove = append(s.toRemove, ruler.Entity)
				s.toAdd = append(s.toAdd, usurperEntity)

				// Clear the debt hooks to represent the collateral being executed.
				s.hooks.AddHook(bestUsurperID, ruler.ID, -worstGrudge) // Neutralize the exact grudge
			}
		}
	}

	// 4. Safely apply structural changes outside the iterator loop
	for _, e := range s.toRemove {
		if world.Alive(e) && world.Has(e, adminMarkerID) {
			world.Remove(e, adminMarkerID)
		}
	}

	for _, e := range s.toAdd {
		if world.Alive(e) && !world.Has(e, adminMarkerID) {
			world.Add(e, adminMarkerID)
		}
	}
}
