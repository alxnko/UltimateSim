package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 54 - The Radicalization Engine (Echo Chamber)
// This system bridges Phase 09.3 (Geography: FootTraffic/Desire Paths) with Phase 07.5 (Memetics: Beliefs) and Phase 33.1 (Cultural Friction).
// If a Village is geographically isolated (very low FootTraffic meaning no caravans or visitors),
// the dominant belief in the village naturally amplifies over time without external preachers, leading to radicalization.

// cache values, not component pointers — GC corruption class, see banditry.go
// (s.npcs persists across ticks; BeliefComponent is re-fetched via the entity handle at use time)
type npcData struct {
	entity ecs.Entity
	cityID uint32
}

type EchoChamberSystem struct {
	mapGrid          *engine.MapGrid
	tickCounter      uint64
	isolatedVillages map[uint32]bool
	npcs             []npcData
	cityBeliefCounts map[uint32]map[uint32]int32
}

// NewEchoChamberSystem creates a new EchoChamberSystem.
func NewEchoChamberSystem(mapGrid *engine.MapGrid) *EchoChamberSystem {
	return &EchoChamberSystem{
		mapGrid:          mapGrid,
		tickCounter:      0,
		isolatedVillages: make(map[uint32]bool),
		npcs:             make([]npcData, 0, 100),
		cityBeliefCounts: make(map[uint32]map[uint32]int32),
	}
}

// Update executes the system logic per tick.
func (s *EchoChamberSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle to every 100 ticks to represent long-term cultural shifts
	if s.tickCounter%100 != 0 {
		return
	}

	if s.mapGrid == nil {
		return
	}

	villageID := ecs.ComponentID[components.Village](world)
	posID := ecs.ComponentID[components.Position](world)
	affID := ecs.ComponentID[components.Affiliation](world)

	// Step 1: Identify geographically isolated villages
	clear(s.isolatedVillages)

	villFilter := ecs.All(villageID, posID, affID)
	villQuery := world.Query(villFilter)

	width := s.mapGrid.Width
	height := s.mapGrid.Height

	for villQuery.Next() {
		pos := (*components.Position)(villQuery.Get(posID))
		aff := (*components.Affiliation)(villQuery.Get(affID))

		tileX := int(pos.X)
		tileY := int(pos.Y)

		if tileX >= 0 && tileX < width && tileY >= 0 && tileY < height {
			index := tileY*width + tileX
			traffic := s.mapGrid.TileStates[index].FootTraffic

			// If FootTraffic is very low, it's an isolated Echo Chamber
			if traffic < 50 {
				s.isolatedVillages[aff.CityID] = true
			}
		}
	}

	if len(s.isolatedVillages) == 0 {
		return
	}

	// Step 2: Extract active NPCs in isolated villages and calculate dominant belief per village
	npcID := ecs.ComponentID[components.NPC](world)
	beliefID := ecs.ComponentID[components.BeliefComponent](world)

	npcFilter := ecs.All(npcID, affID, beliefID)
	npcQuery := world.Query(npcFilter)

	// Pre-cache all NPC entities and their beliefs to avoid nested queries
	s.npcs = s.npcs[:0]
	// Clear nested maps
	for k := range s.cityBeliefCounts {
		clear(s.cityBeliefCounts[k])
	}

	for npcQuery.Next() {
		aff := (*components.Affiliation)(npcQuery.Get(affID))

		if !s.isolatedVillages[aff.CityID] {
			continue
		}

		beliefs := (*components.BeliefComponent)(npcQuery.Get(beliefID))

		s.npcs = append(s.npcs, npcData{
			entity: npcQuery.Entity(),
			cityID: aff.CityID,
		})

		if s.cityBeliefCounts[aff.CityID] == nil {
			s.cityBeliefCounts[aff.CityID] = make(map[uint32]int32)
		}

		// Sum weights to find the cultural consensus
		for _, b := range beliefs.Beliefs {
			s.cityBeliefCounts[aff.CityID][b.BeliefID] += b.Weight
		}
	}

	// Determine dominant belief per isolated city
	cityDominantBelief := make(map[uint32]uint32)
	for cityID, counts := range s.cityBeliefCounts {
		var domBelief uint32 = 0
		var maxWeight int32 = -1

		for bID, weight := range counts {
			if weight > maxWeight || (weight == maxWeight && bID < domBelief) {
				maxWeight = weight
				domBelief = bID
			}
		}

		// Only establish a dominant belief if there is a clear consensus (preventing noise amplification)
		if maxWeight > 0 {
			cityDominantBelief[cityID] = domBelief
		}
	}

	// Step 3: Apply the Echo Chamber effect (amplify dominant, decay minorities)
	for _, n := range s.npcs {
		domBelief, exists := cityDominantBelief[n.cityID]
		if !exists {
			continue
		}

		// Re-fetch via entity handle at use time (cached pointers do not survive archetype moves)
		if !world.Alive(n.entity) {
			continue
		}
		npcBeliefs := (*components.BeliefComponent)(world.Get(n.entity, beliefID))

		foundDominant := false

		for i := 0; i < len(npcBeliefs.Beliefs); i++ {
			b := &npcBeliefs.Beliefs[i]
			if b.BeliefID == domBelief {
				// Amplify dominant belief
				if b.Weight < 200 { // Allow extreme radicalization
					b.Weight += 5
				}
				foundDominant = true
			} else {
				// Decay competing beliefs due to social pressure
				if b.Weight > 0 {
					b.Weight -= 2
				}
			}
		}

		// If they didn't hold the dominant belief at all, they start adopting it due to peer pressure
		if !foundDominant {
			npcBeliefs.Beliefs = append(npcBeliefs.Beliefs, components.Belief{
				BeliefID: domBelief,
				Weight:   1,
			})
		}
	}
}
