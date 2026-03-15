package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 26.1: Caravan Banditry & Supply Chain Collapse
// BanditrySystem converts desperate NPCs into bandits and triggers robbery events on local Caravans.

type BanditrySystem struct {
	npcFilter     ecs.Filter
	caravanFilter ecs.Filter
	toRemove      []ecs.Entity
	toPunish      []ecs.Entity
}

func NewBanditrySystem(world *ecs.World) *BanditrySystem {
	// NPC dependencies: Needs, Position, Desperation, Memory, JobComponent
	needsID := ecs.ComponentID[components.Needs](world)
	posID := ecs.ComponentID[components.Position](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	jobID := ecs.ComponentID[components.JobComponent](world)

	npcMask := ecs.All(needsID, posID, despID, memID, jobID)

	// Caravan dependencies: Position, Caravan (Tag), Payload
	cPosID := ecs.ComponentID[components.Position](world)
	caravanID := ecs.ComponentID[components.Caravan](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	caravanMask := ecs.All(cPosID, caravanID, payloadID)

	return &BanditrySystem{
		npcFilter:     &npcMask,
		caravanFilter: &caravanMask,
		toRemove:      make([]ecs.Entity, 0, 100),
		toPunish:      make([]ecs.Entity, 0, 100),
	}
}

func (s *BanditrySystem) Update(world *ecs.World) {
	s.toRemove = s.toRemove[:0]
	s.toPunish = s.toPunish[:0]

	// Step 1: Pre-cache Caravans into a flat slice for DOD O(1) matching
	cPosID := ecs.ComponentID[components.Position](world)
	payloadID := ecs.ComponentID[components.Payload](world)

	caravanQuery := world.Query(s.caravanFilter)

	type cData struct {
		Entity  ecs.Entity
		X       float32
		Y       float32
		Payload *components.Payload
	}

	caravans := make([]cData, 0, 100)

	for caravanQuery.Next() {
		pos := (*components.Position)(caravanQuery.Get(cPosID))
		payload := (*components.Payload)(caravanQuery.Get(payloadID))

		caravans = append(caravans, cData{
			Entity:  caravanQuery.Entity(),
			X:       pos.X,
			Y:       pos.Y,
			Payload: payload,
		})
	}

	// Step 2: Iterate over NPCs
	needsID := ecs.ComponentID[components.Needs](world)
	posID := ecs.ComponentID[components.Position](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	jobID := ecs.ComponentID[components.JobComponent](world)

	npcQuery := world.Query(s.npcFilter)

	for npcQuery.Next() {
		job := (*components.JobComponent)(npcQuery.Get(jobID))
		desp := (*components.DesperationComponent)(npcQuery.Get(despID))

		// Convert to bandit if sufficiently desperate and not a guard
		if desp.Level >= 50 && job.JobID != components.JobGuard {
			job.JobID = components.JobBandit
		}

		if job.JobID == components.JobBandit {
			pos := (*components.Position)(npcQuery.Get(posID))

			// Find nearest caravan
			var bestC *cData
			var bestDist float32 = 9999999.0
			var bestIdx int = -1

			for i := 0; i < len(caravans); i++ {
				c := &caravans[i]

				// Skip if already marked for removal
				marked := false
				for _, rm := range s.toRemove {
					if rm == c.Entity {
						marked = true
						break
					}
				}
				if marked {
					continue
				}

				dx := pos.X - c.X
				dy := pos.Y - c.Y
				distSq := (dx * dx) + (dy * dy)

				if distSq < bestDist {
					bestDist = distSq
					bestC = c
					bestIdx = i
				}
			}

			if bestC != nil && bestDist < 2.0 {
				// Execute robbery
				needs := (*components.Needs)(npcQuery.Get(needsID))
				mem := (*components.Memory)(npcQuery.Get(memID))

				// Take food
				needs.Food += float32(bestC.Payload.Food)
				// We could take other resources and convert to wealth, but sticking to basics.
				needs.Wealth += float32(bestC.Payload.Iron) + float32(bestC.Payload.Stone) + float32(bestC.Payload.Wood)

				desp.Level = 0

				// Log crime
				event := components.MemoryEvent{
					TargetID:        0, // Target is caravan entity which isn't an NPC, so 0 is fine
					InteractionType: components.InteractionTheft,
					Value:           int32(bestC.Payload.Food),
					TickStamp:       0,
				}
				mem.Events[mem.Head] = event
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))

				// Mark for removal and punishment
				s.toRemove = append(s.toRemove, bestC.Entity)
				s.toPunish = append(s.toPunish, npcQuery.Entity())

				// Remove caravan from local list to prevent multiple bandits hitting the same caravan in one tick
				if bestIdx != -1 {
					caravans = append(caravans[:bestIdx], caravans[bestIdx+1:]...)
				}
			}
		}
	}

	// Step 3: Clean up and apply structural changes outside the query
	for _, e := range s.toRemove {
		if world.Alive(e) {
			world.RemoveEntity(e)
		}
	}

	crimeID := ecs.ComponentID[components.CrimeMarker](world)

	for _, e := range s.toPunish {
		if world.Alive(e) {
			if !world.Has(e, crimeID) {
				world.Add(e, crimeID)
			}
			cm := (*components.CrimeMarker)(world.Get(e, crimeID))
			cm.CrimeLevel = 1
			cm.Bounty = 250 // High bounty for banditry
		}
	}
}
