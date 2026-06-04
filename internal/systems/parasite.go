package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 69: The Parasitic Symbiosis Engine
// Bridges Biology (Vitals), Justice (Witch Hunt), and Geography.
// Hungry parasites pathfind to healthy NPCs and drain their blood to restore their own food needs.
// This generates a deep negative hook and marks the parasite as Esoteric.

type ParasiteSystem struct {
	parasiteFilter ecs.Filter
	victimFilter   ecs.Filter

	posID      ecs.ID
	needsID    ecs.ID
	vitalsID   ecs.ID
	identID    ecs.ID
	pathID     ecs.ID
	parasiteID ecs.ID

	hooks *engine.SparseHookGraph
}

func NewParasiteSystem(world *ecs.World, hooks *engine.SparseHookGraph) *ParasiteSystem {
	posID := ecs.ComponentID[components.Position](world)
	needsID := ecs.ComponentID[components.Needs](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	pathID := ecs.ComponentID[components.Path](world)
	parasiteID := ecs.ComponentID[components.ParasiteComponent](world)

	// Parasites: Must have Needs, Position, Identity, and Path
	pMask := filter.All(parasiteID, needsID, posID, identID, pathID)
	// Victims: Must have Vitals, Position, and Identity, but NOT be a Parasite
	vMask := filter.All(vitalsID, posID, identID).Without(parasiteID)

	return &ParasiteSystem{
		parasiteFilter: &pMask, // Pass by pointer as per memory
		victimFilter:   &vMask,
		posID:          posID,
		needsID:        needsID,
		vitalsID:       vitalsID,
		identID:        identID,
		pathID:         pathID,
		parasiteID:     parasiteID,
		hooks:          hooks,
	}
}

type victimData struct {
	entity ecs.Entity
	x      float32
	y      float32
	id     uint64
	blood  float32
}

func (s *ParasiteSystem) Update(world *ecs.World) {
	// 1. Pre-cache healthy victims to avoid DOD lock issues
	var victims []victimData

	vQuery := world.Query(s.victimFilter)
	for vQuery.Next() {
		vitals := (*components.VitalsComponent)(vQuery.Get(s.vitalsID))
		if vitals.Blood > 50.0 {
			pos := (*components.Position)(vQuery.Get(s.posID))
			ident := (*components.Identity)(vQuery.Get(s.identID))

			victims = append(victims, victimData{
				entity: vQuery.Entity(),
				x:      pos.X,
				y:      pos.Y,
				id:     ident.ID,
				blood:  vitals.Blood,
			})
		}
	}

	if len(victims) == 0 {
		return // No healthy victims available
	}

	// 2. Iterate hungry parasites
	pQuery := world.Query(s.parasiteFilter)

	for pQuery.Next() {
		needs := (*components.Needs)(pQuery.Get(s.needsID))

		// If the parasite is hungry (Food < 30.0)
		if needs.Food < 30.0 {
			pPos := (*components.Position)(pQuery.Get(s.posID))
			pIdent := (*components.Identity)(pQuery.Get(s.identID))
			path := (*components.Path)(pQuery.Get(s.pathID))

			// Find closest healthy victim
			var bestVictim *victimData
			var bestDistSq float32 = 9999999.0

			for i := range victims {
				v := &victims[i]

				if !world.Alive(v.entity) {
					continue
				}

				// Check if this victim is currently healthy according to the cache
				if v.blood <= 50.0 {
					continue
				}

				dx := pPos.X - v.x
				dy := pPos.Y - v.y
				distSq := dx*dx + dy*dy

				if distSq < bestDistSq {
					bestDistSq = distSq
					bestVictim = v
				}
			}

			if bestVictim == nil {
				continue
			}

			if bestDistSq <= 2.0 {
				// Execute Blood Drain (Symbiosis Attack)
				vVitals := (*components.VitalsComponent)(world.Get(bestVictim.entity, s.vitalsID))

				// Ensure victim hasn't been drained by another parasite this tick
				if vVitals.Blood > 50.0 {
					// Drain blood
					vVitals.Blood -= 40.0
					vVitals.Pain += 40.0 // Massive pain spike (Phase 62 Trigger)

					// Restore parasite's food
					needs.Food += 50.0

					// Generate negative hook from victim to parasite (Phase 23 Trigger)
					if s.hooks != nil {
						s.hooks.AddHook(bestVictim.id, pIdent.ID, -100)
					}

					// Apply Esoteric Trait to the parasite (Phase 49 Witch Hunt Trigger)
					pIdent.BaseTraits |= components.TraitEsoteric

					// Update victim cache so they aren't targeted again this tick
					bestVictim.blood = vVitals.Blood
				}

			} else {
				// Pathfind to victim
				if path.TargetX != bestVictim.x || path.TargetY != bestVictim.y {
					path.TargetX = bestVictim.x
					path.TargetY = bestVictim.y
					path.HasPath = false // Trigger repathing in MovementSystem/WanderSystem
				}
			}
		}
	}
}
