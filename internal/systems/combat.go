package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Phase 64: The Physical Combat Engine
// Evaluates active CombatMarkers, physically depletes vitals based on DOD patterns,
// and breaks intent when the target dies or moves out of range.

type CombatSystem struct {
	combatFilter ecs.Filter
	targetFilter ecs.Filter

	combatID ecs.ID
	vitalsID ecs.ID
	posID    ecs.ID
	identID  ecs.ID
}

func NewCombatSystem(world *ecs.World) *CombatSystem {
	combatID := ecs.ComponentID[components.CombatMarker](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	identID := ecs.ComponentID[components.Identity](world)

	cMask := ecs.All(combatID, vitalsID, posID)
	tMask := ecs.All(identID, vitalsID, posID)

	return &CombatSystem{
		combatFilter: &cMask,
		targetFilter: &tMask,
		combatID:     combatID,
		vitalsID:     vitalsID,
		posID:        posID,
		identID:      identID,
	}
}

type targetData struct {
	entity ecs.Entity
	x      float32
	y      float32
	vitals *components.VitalsComponent
}

func (s *CombatSystem) Update(world *ecs.World) {
	// Pre-cache potential targets into an O(1) map for DOD performance
	targets := make(map[uint64]targetData)
	tQuery := world.Query(s.targetFilter)
	for tQuery.Next() {
		ident := (*components.Identity)(tQuery.Get(s.identID))
		pos := (*components.Position)(tQuery.Get(s.posID))
		vitals := (*components.VitalsComponent)(tQuery.Get(s.vitalsID))

		targets[ident.ID] = targetData{
			entity: tQuery.Entity(),
			x:      pos.X,
			y:      pos.Y,
			vitals: vitals,
		}
	}

	markersToRemove := make([]ecs.Entity, 0, 10)

	cQuery := world.Query(s.combatFilter)
	for cQuery.Next() {
		combat := (*components.CombatMarker)(cQuery.Get(s.combatID))
		attackerPos := (*components.Position)(cQuery.Get(s.posID))
		attackerVitals := (*components.VitalsComponent)(cQuery.Get(s.vitalsID))

		target, exists := targets[combat.TargetID]
		if !exists || !world.Alive(target.entity) || target.vitals.Blood <= 0 {
			// Target is dead or invalid, end combat
			markersToRemove = append(markersToRemove, cQuery.Entity())
			continue
		}

		// Distance check (combat range)
		dx := attackerPos.X - target.x
		dy := attackerPos.Y - target.y
		distSq := dx*dx + dy*dy

		if distSq > 4.0 {
			// Target escaped, end combat
			markersToRemove = append(markersToRemove, cQuery.Entity())
			continue
		}

		// Execute physical combat
		// Attacker expends stamina
		attackerVitals.Stamina -= 5.0
		if attackerVitals.Stamina < 0 {
			attackerVitals.Stamina = 0
		}

		// Target takes damage
		target.vitals.Blood -= 10.0
		if target.vitals.Blood < 0 {
			target.vitals.Blood = 0
		}

		// Target suffers pain (which bridges to Psychological Stress Engine)
		target.vitals.Pain += 15.0
		if target.vitals.Pain > 100.0 {
			target.vitals.Pain = 100.0
		}

		// If target dies from this blow, mark for combat removal
		if target.vitals.Blood <= 0 {
			markersToRemove = append(markersToRemove, cQuery.Entity())
		}
	}

	for _, e := range markersToRemove {
		if world.Alive(e) && world.Has(e, s.combatID) {
			world.Remove(e, s.combatID)
		}
	}
}
