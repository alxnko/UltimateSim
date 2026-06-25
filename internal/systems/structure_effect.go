package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/mlange-42/arche/ecs"
)

// Shell Phase: StructureEffectSystem applies the ongoing runtime effects of
// completed structures, satisfying macro/micro parity — each building visibly
// changes the world around it.

const structureEffectTickRate = 200

type structureData struct {
	stype uint32
	dataA uint32
	x, y  float32
}

type StructureEffectSystem struct {
	tick uint64
}

func NewStructureEffectSystem() *StructureEffectSystem { return &StructureEffectSystem{} }

// IsExpensive throttles the system during fast-forward.
func (s *StructureEffectSystem) IsExpensive() bool { return true }

func (s *StructureEffectSystem) Update(world *ecs.World) {
	s.tick++
	if s.tick%structureEffectTickRate != 0 {
		return
	}

	structID := ecs.ComponentID[components.StructureComponent](world)
	posID := ecs.ComponentID[components.Position](world)
	npcID := ecs.ComponentID[components.NPC](world)
	needsID := ecs.ComponentID[components.Needs](world)
	beliefID := ecs.ComponentID[components.BeliefComponent](world)
	villageID := ecs.ComponentID[components.Village](world)
	storID := ecs.ComponentID[components.StorageComponent](world)

	// Pass 1: collect structures.
	var structs []structureData
	q := world.Query(ecs.All(structID, posID))
	for q.Next() {
		st := (*components.StructureComponent)(q.Get(structID))
		pos := (*components.Position)(q.Get(posID))
		structs = append(structs, structureData{st.StructureType, st.DataA, pos.X, pos.Y})
	}
	if len(structs) == 0 {
		return
	}

	// Pass 2: NPC auras (rest from houses/taverns) + shrine belief spread.
	nq := world.Query(ecs.All(npcID, posID))
	for nq.Next() {
		pos := (*components.Position)(nq.Get(posID))
		ent := nq.Entity()
		hasNeeds := world.Has(ent, needsID)
		hasBelief := world.Has(ent, beliefID)
		if !hasNeeds && !hasBelief {
			continue
		}
		for i := range structs {
			st := &structs[i]
			dx := pos.X - st.x
			dy := pos.Y - st.y
			distSq := dx*dx + dy*dy
			switch uint8(st.stype) {
			case components.StructureHouse:
				if hasNeeds && distSq <= 9 {
					addRest((*components.Needs)(nq.Get(needsID)), 5)
				}
			case components.StructureTavern:
				if hasNeeds && distSq <= 25 {
					addRest((*components.Needs)(nq.Get(needsID)), 3)
				}
			case components.StructureShrine:
				if hasBelief && st.dataA != 0 && distSq <= 25 {
					spreadBelief((*components.BeliefComponent)(nq.Get(beliefID)), st.dataA)
				}
			}
		}
	}

	// Pass 3: farms feed the nearest village.
	for i := range structs {
		if uint8(structs[i].stype) != components.StructureFarm {
			continue
		}
		fx, fy := structs[i].x, structs[i].y
		var bestStore *components.StorageComponent
		bestD := float32(25)
		vq := world.Query(ecs.All(villageID, posID, storID))
		for vq.Next() {
			pos := (*components.Position)(vq.Get(posID))
			dx := pos.X - fx
			dy := pos.Y - fy
			d := dx*dx + dy*dy
			if d <= bestD {
				bestD = d
				bestStore = (*components.StorageComponent)(vq.Get(storID))
			}
		}
		if bestStore != nil {
			bestStore.Food += 5
		}
	}
}

func addRest(n *components.Needs, amt float32) {
	n.Rest += amt
	if n.Rest > 100 {
		n.Rest = 100
	}
}

func spreadBelief(b *components.BeliefComponent, beliefID uint32) {
	for i := range b.Beliefs {
		if b.Beliefs[i].BeliefID == beliefID {
			b.Beliefs[i].Weight++
			return
		}
	}
	b.Beliefs = append(b.Beliefs, components.Belief{BeliefID: beliefID, Weight: 1})
}
