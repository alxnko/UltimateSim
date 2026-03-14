package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 23.1: The Blood Feud Engine
// Scans for deep negative hooks between NPCs. If found, one NPC murders the other.
// Murder triggers an immediate generational negative hook inheritance across entire Clans.

type feudNodeData struct {
	entity ecs.Entity
	id     uint64
	clanID uint32
	x      float32
	y      float32
	mem    *components.Memory
}

type BloodFeudSystem struct {
	tickCounter uint64
	hooks       *engine.SparseHookGraph
	filter      ecs.Filter

	// Component IDs mapped once during NewBloodFeudSystem
	posID   ecs.ID
	identID ecs.ID
	affID   ecs.ID
	memID   ecs.ID
	needsID ecs.ID
}

func NewBloodFeudSystem(world *ecs.World, hooks *engine.SparseHookGraph) *BloodFeudSystem {
	posID := ecs.ComponentID[components.Position](world)
	identID := ecs.ComponentID[components.Identity](world)
	affID := ecs.ComponentID[components.Affiliation](world)
	memID := ecs.ComponentID[components.Memory](world)
	needsID := ecs.ComponentID[components.Needs](world)

	mask := ecs.All(posID, identID, affID, memID, needsID)

	return &BloodFeudSystem{
		hooks:   hooks,
		filter:  &mask,
		posID:   posID,
		identID: identID,
		affID:   affID,
		memID:   memID,
		needsID: needsID,
	}
}

func (s *BloodFeudSystem) Update(world *ecs.World) {
	s.tickCounter++

	// Extract NPCs into a flat DOD slice
	query := world.Query(s.filter)
	nodes := make([]feudNodeData, 0, 500)

	for query.Next() {
		pos := (*components.Position)(query.Get(s.posID))
		ident := (*components.Identity)(query.Get(s.identID))
		aff := (*components.Affiliation)(query.Get(s.affID))
		mem := (*components.Memory)(query.Get(s.memID))

		nodes = append(nodes, feudNodeData{
			entity: query.Entity(),
			id:     ident.ID,
			clanID: aff.ClanID,
			x:      pos.X,
			y:      pos.Y,
			mem:    mem,
		})
	}

	deadEntities := make(map[uint64]bool)

	for i := 0; i < len(nodes); i++ {
		killer := nodes[i]

		if deadEntities[killer.id] {
			continue
		}

		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			victim := nodes[j]

			if deadEntities[victim.id] {
				continue
			}

			// Distance check (adjacent tiles)
			dx := killer.x - victim.x
			dy := killer.y - victim.y
			distSq := dx*dx + dy*dy

			if distSq < 2.0 {
				// Evaluate grudge/hook (-50 threshold for murder)
				// Using hooks.GetHook directly
				grudge := s.hooks.GetHook(killer.id, victim.id)
				if grudge <= -50 {
					// Murder logic execution

					// 1. Log InteractionMurder in Killer's memory
					idx := killer.mem.Head
					killer.mem.Events[idx] = components.MemoryEvent{
						TargetID:        victim.id,
						TickStamp:       s.tickCounter,
						InteractionType: components.InteractionMurder,
					}
					killer.mem.Head = (idx + 1) % 50

					// 2. Kill victim by starving them natively
					vNeeds := (*components.Needs)(world.Get(victim.entity, s.needsID))
					vNeeds.Food = 0

					// Prevent double-kills
					deadEntities[victim.id] = true

					// 3. Propagate the Feud across Clans (O(N) iteration over nodes)
					// Victim's Clan now hates Killer and Killer's Clan
					for k := 0; k < len(nodes); k++ {
						bystander := nodes[k]

						// Bystander belongs to Victim's Clan
						if bystander.clanID == victim.clanID && bystander.id != victim.id {
							// Hate the killer specifically
							s.hooks.AddHook(bystander.id, killer.id, -100)

							// Hate the killer's clan members
							if killer.clanID != 0 {
								for m := 0; m < len(nodes); m++ {
									kClanMember := nodes[m]
									if kClanMember.clanID == killer.clanID && kClanMember.id != killer.id {
										s.hooks.AddHook(bystander.id, kClanMember.id, -50)
									}
								}
							}
						}
					}

					break // Killer executed one murder, skip remaining victims this tick
				}
			}
		}
	}
}
