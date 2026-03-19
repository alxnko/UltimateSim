package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 43: Organic Administration Engine
// LeadershipEmergenceSystem evaluates NPCs within a CityID to determine who has the highest
// volume of powerful positive hooks in the SparseHookGraph. The winner emerges natively as the Administration.

const LeadershipEmergenceTickRate = 500

type npcLeadershipData struct {
	Entity ecs.Entity
	ID     uint64
	CityID uint32
	HasTag bool
}

type LeadershipEmergenceSystem struct {
	hooks     *engine.SparseHookGraph
	tickCount uint64

	// Pre-allocated flat slices to prevent GC pressure and ECS locks
	npcs        []npcLeadershipData
	cityLeaders map[uint32]uint64     // CityID -> Leading NPC ID
	cityScores  map[uint32]int        // CityID -> Leading Score
	cityEntities map[uint32]ecs.Entity // CityID -> Leading NPC Entity

	toRemove []ecs.Entity // NPCs that lost the election
	toAdd    []ecs.Entity // NPCs that won the election
}

// Evolution: Phase 43 - Organic Administration Engine
func NewLeadershipEmergenceSystem(hooks *engine.SparseHookGraph) *LeadershipEmergenceSystem {
	return &LeadershipEmergenceSystem{
		hooks:        hooks,
		npcs:         make([]npcLeadershipData, 0, 1000),
		cityLeaders:  make(map[uint32]uint64),
		cityScores:   make(map[uint32]int),
		cityEntities: make(map[uint32]ecs.Entity),
		toRemove:     make([]ecs.Entity, 0, 50),
		toAdd:        make([]ecs.Entity, 0, 50),
	}
}

func (s *LeadershipEmergenceSystem) Update(world *ecs.World) {
	s.tickCount++

	if s.tickCount%LeadershipEmergenceTickRate != 0 {
		return
	}

	if s.hooks == nil {
		return
	}

	// 1. Reset caches
	s.npcs = s.npcs[:0]
	for k := range s.cityLeaders {
		delete(s.cityLeaders, k)
	}
	for k := range s.cityScores {
		delete(s.cityScores, k)
	}
	for k := range s.cityEntities {
		delete(s.cityEntities, k)
	}
	s.toRemove = s.toRemove[:0]
	s.toAdd = s.toAdd[:0]

	// 2. Pre-cache all valid NPCs residing in a city
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	affilID := ecs.ComponentID[components.Affiliation](world)
	adminMarkerID := ecs.ComponentID[components.AdministrationMarker](world)

	query := world.Query(filter.All(npcID, identID, affilID))

	for query.Next() {
		affil := (*components.Affiliation)(query.Get(affilID))
		if affil.CityID == 0 {
			continue // Skip homeless / wandering NPCs
		}

		ident := (*components.Identity)(query.Get(identID))
		hasTag := query.Has(adminMarkerID)

		s.npcs = append(s.npcs, npcLeadershipData{
			Entity: query.Entity(),
			ID:     ident.ID,
			CityID: affil.CityID,
			HasTag: hasTag,
		})
	}

	// 3. Evaluate Influence Scores per city
	for i := 0; i < len(s.npcs); i++ {
		npc := s.npcs[i]

		// Calculate total positive influence from the SparseHookGraph
		incomingHooks := s.hooks.GetAllIncomingHooks(npc.ID)
		score := 0
		for _, val := range incomingHooks {
			if val > 0 {
				score += val
			}
		}

		// Update the current leader for the city if this NPC has a higher score
		// In case of a tie, the first one evaluated (or the existing leader if we wanted to be sticky) wins.
		// We add a slight bias (+1) if they already have the tag to prevent constant flipping on ties.
		effectiveScore := score
		if npc.HasTag {
			effectiveScore += 1
		}

		currentBest, exists := s.cityScores[npc.CityID]
		if !exists || effectiveScore > currentBest {
			s.cityScores[npc.CityID] = effectiveScore
			s.cityLeaders[npc.CityID] = npc.ID
			s.cityEntities[npc.CityID] = npc.Entity
		} else if exists && effectiveScore == currentBest {
			// In case of a tie, existing leader retains leadership for stability.
			// However, if the new evaluated npc DOES have a tag and the current best does NOT,
			// the tag-holder should win the tie.
			currentBestLeaderID := s.cityLeaders[npc.CityID]
			currentBestLeaderHasTag := false
			for j := 0; j < i; j++ {
				if s.npcs[j].ID == currentBestLeaderID {
					currentBestLeaderHasTag = s.npcs[j].HasTag
					break
				}
			}

			if !currentBestLeaderHasTag && npc.HasTag {
				s.cityScores[npc.CityID] = effectiveScore
				s.cityLeaders[npc.CityID] = npc.ID
				s.cityEntities[npc.CityID] = npc.Entity
			}
		}
	}

	// 4. Determine structural changes
	for i := 0; i < len(s.npcs); i++ {
		npc := s.npcs[i]
		isWinner := (s.cityLeaders[npc.CityID] == npc.ID)

		if npc.HasTag && !isWinner {
			// Dethroned
			s.toRemove = append(s.toRemove, npc.Entity)
		} else if !npc.HasTag && isWinner {
			// Crowned
			s.toAdd = append(s.toAdd, npc.Entity)
		}
	}

	// 5. Apply structural changes safely outside the main query loop
	for _, e := range s.toRemove {
		if world.Alive(e) {
			world.Remove(e, adminMarkerID)
		}
	}

	for _, e := range s.toAdd {
		if world.Alive(e) {
			world.Add(e, adminMarkerID)
		}
	}
}
