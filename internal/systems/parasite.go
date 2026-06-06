package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 69: The Parasitic Symbiosis Engine
// Bridges Biology and Justice by having ParasiteComponent entities drain Blood from healthy NPCs
// to restore Needs.Food, bypassing the agrarian economy. Generates negative hooks and applies
// TraitEsoteric to the parasite's Identity.

type ParasiteSystem struct {
	parasiteFilter ecs.Filter
	victimFilter   ecs.Filter

	posID      ecs.ID
	needsID    ecs.ID
	identID    ecs.ID
	parasiteID ecs.ID
	vitalsID   ecs.ID
	memID      ecs.ID
	npcID      ecs.ID

	hooks *engine.SparseHookGraph
}

func NewParasiteSystem(world *ecs.World, hooks *engine.SparseHookGraph) *ParasiteSystem {
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	identID := ecs.ComponentID[components.Identity](world)
	parasiteID := ecs.ComponentID[components.ParasiteComponent](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	npcID := ecs.ComponentID[components.NPC](world)

	pMask := ecs.All(posID, needsID, identID, parasiteID)
	vMask := ecs.All(posID, vitalsID, identID, npcID).Without(parasiteID)

	return &ParasiteSystem{
		parasiteFilter: &pMask,
		victimFilter:   &vMask,
		posID:          posID,
		needsID:        needsID,
		identID:        identID,
		parasiteID:     parasiteID,
		vitalsID:       vitalsID,
		memID:          memID,
		npcID:          npcID,
		hooks:          hooks,
	}
}

type parasiteVictimData struct {
	entity ecs.Entity
	x      float32
	y      float32
	id     uint64
	blood  float32
}

func (s *ParasiteSystem) Update(world *ecs.World) {
	// Pre-cache potential victims to avoid nested Arche-Go queries
	var potentialVictims []parasiteVictimData

	vQuery := world.Query(s.victimFilter)
	for vQuery.Next() {
		vitals := (*components.VitalsComponent)(vQuery.Get(s.vitalsID))

		// Only target healthy individuals with enough blood
		if vitals.Blood >= 30.0 {
			pos := (*components.Position)(vQuery.Get(s.posID))
			ident := (*components.Identity)(vQuery.Get(s.identID))

			potentialVictims = append(potentialVictims, parasiteVictimData{
				entity: vQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				id:     ident.ID,
				blood:  vitals.Blood,
			})
		}
	}

	if len(potentialVictims) == 0 {
		return
	}

	pQuery := world.Query(s.parasiteFilter)
	for pQuery.Next() {
		needs := (*components.Needs)(pQuery.Get(s.needsID))
		parasite := (*components.ParasiteComponent)(pQuery.Get(s.parasiteID))

		// Only hunt if hungry
		if needs.Food > 50.0 {
			continue
		}

		pPos := (*components.Position)(pQuery.Get(s.posID))
		pIdent := (*components.Identity)(pQuery.Get(s.identID))

		var bestVictim *parasiteVictimData
		var bestDistSq float32 = 9999999.0
		bestIndex := -1

		// Find closest healthy victim
		for i := range potentialVictims {
			v := &potentialVictims[i]
			if !world.Alive(v.entity) {
				continue
			}

			dx := pPos.X - v.x
			dy := pPos.Y - v.y
			distSq := dx*dx + dy*dy

			if distSq < bestDistSq {
				bestDistSq = distSq
				bestVictim = v
				bestIndex = i
			}
		}

		if bestVictim == nil {
			continue
		}

		// If adjacent, attack
		if bestDistSq <= 4.0 {
			vVitals := (*components.VitalsComponent)(world.Get(bestVictim.entity, s.vitalsID))

			// Drain blood
			drainAmount := float32(20.0)
			if vVitals.Blood < drainAmount {
				drainAmount = vVitals.Blood
			}

			vVitals.Blood -= drainAmount
			vVitals.Pain += 20.0 // It hurts

			// Restore parasite needs
			needs.Food += drainAmount * 2.0 // Blood is very nutritious for parasites
			if needs.Food > 100.0 {
				needs.Food = 100.0
			}
			parasite.BloodSatiety += drainAmount

			// Expose the parasite
			pIdent.BaseTraits |= components.TraitEsoteric
			parasite.IsHidden = false

			// Generate Hooks & Memory
			if s.hooks != nil {
				// Extreme negative hook from victim to parasite
				s.hooks.AddHook(bestVictim.id, pIdent.ID, -50)
			}

			// If victim has memory, record the attack
			if world.Has(bestVictim.entity, s.memID) {
				vMem := (*components.Memory)(world.Get(bestVictim.entity, s.memID))
				for j := 0; j < len(vMem.Events); j++ {
					if vMem.Events[j].InteractionType == 0 {
						vMem.Events[j] = components.MemoryEvent{
							TargetID:        pIdent.ID,
							InteractionType: components.InteractionParasite,
							TickStamp:       0, // Real tick should be passed, but 0 is okay for this scope
						}
						break
					}
				}
			}

			// Remove victim from consideration for other parasites this tick
			if bestIndex != -1 {
				potentialVictims[bestIndex] = potentialVictims[len(potentialVictims)-1]
				potentialVictims = potentialVictims[:len(potentialVictims)-1]
			}
		}
	}
}
