package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 64: The Physical Combat Engine
// Bridges Blood Feuds (Intent) to Advanced Biology (Vitals), executing physical damage over time.

type combatNodeData struct {
	Entity   ecs.Entity
	ID       uint64
	X        float32
	Y        float32
	Vitals   *components.VitalsComponent
}

type CombatSystem struct {
	filter ecs.Filter

	combatID ecs.ID
	posID    ecs.ID
	identID  ecs.ID
	vitalsID ecs.ID
	genomeID ecs.ID
	equipID  ecs.ID
}

func NewCombatSystem(world *ecs.World) *CombatSystem {
	combatID := ecs.ComponentID[components.CombatMarker](world)
	posID := ecs.ComponentID[components.Position](world)
	identID := ecs.ComponentID[components.Identity](world)
	vitalsID := ecs.ComponentID[components.VitalsComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	equipID := ecs.ComponentID[components.EquipmentComponent](world)

	mask := filter.All(combatID, posID, identID, vitalsID, genomeID)

	return &CombatSystem{
		filter:   mask,
		combatID: combatID,
		posID:    posID,
		identID:  identID,
		vitalsID: vitalsID,
		genomeID: genomeID,
		equipID:  equipID,
	}
}

func (s *CombatSystem) Update(world *ecs.World) {
	// 1. Pre-cache potential victims (entities with Vitals) for O(1) distance and stat checks.
	victimQuery := world.Query(filter.All(s.vitalsID, s.posID, s.identID))
	victims := make(map[uint64]combatNodeData)

	for victimQuery.Next() {
		pos := (*components.Position)(victimQuery.Get(s.posID))
		ident := (*components.Identity)(victimQuery.Get(s.identID))
		vitals := (*components.VitalsComponent)(victimQuery.Get(s.vitalsID))

		victims[ident.ID] = combatNodeData{
			Entity:   victimQuery.Entity(),
			ID:       ident.ID,
			X:        pos.X,
			Y:        pos.Y,
			Vitals:   vitals,
		}
	}
	// The query automatically unlocks the world after finishing the loop.

	// Now iterate over attackers with CombatMarker
	attackerQuery := world.Query(s.filter)
	var finishedCombats []ecs.Entity

	for attackerQuery.Next() {
		combat := (*components.CombatMarker)(attackerQuery.Get(s.combatID))
		pos := (*components.Position)(attackerQuery.Get(s.posID))
		vitals := (*components.VitalsComponent)(attackerQuery.Get(s.vitalsID))
		genome := (*components.GenomeComponent)(attackerQuery.Get(s.genomeID))

		victim, exists := victims[combat.TargetID]
		if !exists || victim.Vitals.Blood <= 0 {
			// Victim is dead or gone, finish combat
			finishedCombats = append(finishedCombats, attackerQuery.Entity())
			continue
		}

		// Check distance
		dx := pos.X - victim.X
		dy := pos.Y - victim.Y
		distSq := dx*dx + dy*dy

		if distSq < 2.0 {
			// Melee range! Execute attack if enough stamina
			if vitals.Stamina >= 10.0 {
				vitals.Stamina -= 10.0 // Cost of attack

				// Calculate damage
				damage := float32(genome.Strength) * 0.1 // Base damage

				if attackerQuery.Has(s.equipID) {
					equip := (*components.EquipmentComponent)(attackerQuery.Get(s.equipID))
					if equip.Equipped {
						// Add weapon prestige to damage as an abstraction for weapon quality
						damage += float32(equip.Weapon.Prestige) * 0.5
					}
				}

				// Inflict damage (Pain and Blood loss)
				victim.Vitals.Pain += damage
				victim.Vitals.Blood -= damage

				if victim.Vitals.Blood <= 0 {
					victim.Vitals.Blood = 0
					finishedCombats = append(finishedCombats, attackerQuery.Entity())
				}
			}
		}
	}

	// Remove finished combats (world is unlocked here)
	for _, e := range finishedCombats {
		if world.Alive(e) && world.Has(e, s.combatID) {
			world.Remove(e, s.combatID)
		}
	}
}
