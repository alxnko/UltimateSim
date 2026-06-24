package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 71: The Macabre Survival Engine (CannibalismSystem)
// Bridges Biology (Starvation), Economy (Desperation), Sanitation (Corpses),
// Psychology (Stress), and Parasitic Symbiosis (Phase 69).

type CannibalismSystem struct {
	npcFilter    ecs.Filter
	corpseFilter ecs.Filter

	pathQueue *engine.PathRequestQueue

	corpses  []cannibalCorpseData
	consumed []ecs.Entity
}

type cannibalCorpseData struct {
	entity ecs.Entity
	x      float32
	y      float32
}

func NewCannibalismSystem(world *ecs.World, pathQueue *engine.PathRequestQueue) *CannibalismSystem {
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	corpseID := ecs.ComponentID[components.CorpseComponent](world)

	npcMask := filter.All(posID, needsID, despID, sanityID)
	corpseMask := filter.All(posID, corpseID)

	return &CannibalismSystem{
		npcFilter:    npcMask,
		corpseFilter: corpseMask,
		pathQueue:    pathQueue,
		corpses:      make([]cannibalCorpseData, 0, 100),
		consumed:     make([]ecs.Entity, 0, 20),
	}
}

func (s *CannibalismSystem) Update(world *ecs.World) {
	s.corpses = s.corpses[:0]
	s.consumed = s.consumed[:0]

	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	sanityID := ecs.ComponentID[components.SanityComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	memID := ecs.ComponentID[components.Memory](world)
	parasiteID := ecs.ComponentID[components.ParasiteComponent](world)
	pathID := ecs.ComponentID[components.Path](world)

	// Pre-cache corpses
	corpseQuery := world.Query(s.corpseFilter)
	for corpseQuery.Next() {
		pos := (*components.Position)(corpseQuery.Get(posID))
		s.corpses = append(s.corpses, cannibalCorpseData{
			entity: corpseQuery.Entity(),
			x:      pos.X,
			y:      pos.Y,
		})
	}

	if len(s.corpses) == 0 {
		return
	}

	// Iterate starving, desperate NPCs
	npcQuery := world.Query(s.npcFilter)
	var newlyInfected []ecs.Entity

	for npcQuery.Next() {
		needs := (*components.Needs)(npcQuery.Get(needsID))
		desp := (*components.DesperationComponent)(npcQuery.Get(despID))

		// Check thresholds (same logic as DesperationSystem catalyst roughly)
		if needs.Food >= 30.0 || desp.Level < 50 {
			continue
		}

		pos := (*components.Position)(npcQuery.Get(posID))
		sanity := (*components.SanityComponent)(npcQuery.Get(sanityID))

		var bestCorpse *cannibalCorpseData
		var bestDistSq float32 = 9999999.0

		for i := 0; i < len(s.corpses); i++ {
			c := &s.corpses[i]

			// Check if already consumed this tick
			alreadyConsumed := false
			for _, e := range s.consumed {
				if e == c.entity {
					alreadyConsumed = true
					break
				}
			}
			if alreadyConsumed {
				continue
			}

			dx := pos.X - c.x
			dy := pos.Y - c.y
			distSq := dx*dx + dy*dy

			if distSq < bestDistSq {
				bestDistSq = distSq
				bestCorpse = c
			}
		}

		if bestCorpse == nil {
			continue
		}

		if bestDistSq <= 2.0 {
			// Consume corpse
			s.consumed = append(s.consumed, bestCorpse.entity)

			needs.Food += 50.0
			if needs.Food > 100.0 {
				needs.Food = 100.0
			}

			sanity.Stress += 50.0
			if sanity.Stress > sanity.MaxStress {
				sanity.Stress = sanity.MaxStress
			}

			desp.Level = 0

			if npcQuery.Has(memID) {
				mem := (*components.Memory)(npcQuery.Get(memID))
				event := components.MemoryEvent{
					TargetID:        0, // Self/Esoteric
					InteractionType: components.InteractionEsoteric,
					Value:           -50,
					TickStamp:       0,
				}
				mem.Events[mem.Head] = event
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))
			}

			if npcQuery.Has(identID) {
				ident := (*components.Identity)(npcQuery.Get(identID))
				ident.BaseTraits |= components.TraitEsoteric
			}

			if !npcQuery.Has(parasiteID) {
				newlyInfected = append(newlyInfected, npcQuery.Entity())
			}

		} else {
			// Pathfind
			if s.pathQueue != nil && npcQuery.Has(identID) {
				ident := (*components.Identity)(npcQuery.Get(identID))

				if npcQuery.Has(pathID) {
					path := (*components.Path)(npcQuery.Get(pathID))
					path.TargetX = bestCorpse.x
					path.TargetY = bestCorpse.y
					path.HasPath = false
				}

				s.pathQueue.Enqueue(engine.PathRequest{
					EntityID: ident.ID,
					StartX:   pos.X,
					StartY:   pos.Y,
					TargetX:  bestCorpse.x,
					TargetY:  bestCorpse.y,
					IsNaval:  false,
				})
			}
		}
	}

	for _, e := range newlyInfected {
		if world.Alive(e) {
			world.Add(e, parasiteID)
			parasite := (*components.ParasiteComponent)(world.Get(e, parasiteID))
			parasite.BloodSatiety = 100.0
			parasite.IsHidden = true
		}
	}

	for _, e := range s.consumed {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}
}
