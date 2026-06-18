package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 70: Deed Forgery Engine (ForgerySystem)
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect NPCs
// to steal the BusinessComponent ownership from lower-intellect owners.

type forgeryBusinessData struct {
	Entity         ecs.Entity
	OwnerID        uint64
	OwnerIntellect uint8
}

type structuralForgeryChange struct {
	BusinessEntity ecs.Entity
	NewOwnerID     uint64
	OldOwnerID     uint64
	ForgerEntity   ecs.Entity
}

type ForgerySystem struct {
	hookGraph      *engine.SparseHookGraph
	businessFilter ecs.Filter
	npcFilter      ecs.Filter
	changes        []structuralForgeryChange
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	busTagID := ecs.ComponentID[components.BusinessEntity](world)
	busCompID := ecs.ComponentID[components.BusinessComponent](world)
	bMask := filter.All(busTagID, busCompID)

	npcTagID := ecs.ComponentID[components.NPC](world)
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	npcMask := filter.All(npcTagID, idID, genID, despID, memID)

	return &ForgerySystem{
		hookGraph:      hooks,
		businessFilter: &bMask,
		npcFilter:      &npcMask,
		changes:        make([]structuralForgeryChange, 0, 10),
	}
}

func (s *ForgerySystem) Update(world *ecs.World) {
	if s.hookGraph == nil {
		return
	}

	busCompID := ecs.ComponentID[components.BusinessComponent](world)
	idID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)

	s.changes = s.changes[:0]

	// 1. Pre-cache all NPC intellects to avoid inner-loop map queries.
	ownerIntellectCache := make(map[uint64]uint8)
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(idID))
		gen := (*components.GenomeComponent)(npcQuery.Get(genID))
		ownerIntellectCache[ident.ID] = gen.Intellect
	}

	// 2. Pre-cache businesses.
	businesses := make([]forgeryBusinessData, 0, 50)
	busQuery := world.Query(s.businessFilter)
	for busQuery.Next() {
		busComp := (*components.BusinessComponent)(busQuery.Get(busCompID))

		intellect, exists := ownerIntellectCache[busComp.OwnerID]
		if !exists {
			intellect = 0 // Default if owner not found
		}

		businesses = append(businesses, forgeryBusinessData{
			Entity:         busQuery.Entity(),
			OwnerID:        busComp.OwnerID,
			OwnerIntellect: intellect,
		})
	}

	if len(businesses) == 0 {
		return
	}

	// 3. Identify forgeries.
	stolenThisTick := make(map[ecs.Entity]bool)

	npcQuery2 := world.Query(s.npcFilter)
	for npcQuery2.Next() {
		desp := (*components.DesperationComponent)(npcQuery2.Get(despID))
		if desp.Level == 0 {
			continue // Not desperate
		}

		gen := (*components.GenomeComponent)(npcQuery2.Get(genID))
		ident := (*components.Identity)(npcQuery2.Get(idID))

		for i := 0; i < len(businesses); i++ {
			b := &businesses[i]

			if b.OwnerID == ident.ID {
				continue
			}

			if stolenThisTick[b.Entity] {
				continue
			}

			// Core logic: High-intellect desperate NPC steals from lower-intellect owner
			if gen.Intellect > b.OwnerIntellect {
				s.changes = append(s.changes, structuralForgeryChange{
					BusinessEntity: b.Entity,
					NewOwnerID:     ident.ID,
					OldOwnerID:     b.OwnerID,
					ForgerEntity:   npcQuery2.Entity(),
				})
				stolenThisTick[b.Entity] = true
				break // One successful forgery per NPC per tick
			}
		}
	}

	// 4. Apply structural changes and memory logging outside of iteration.
	for _, c := range s.changes {
		if world.Alive(c.BusinessEntity) {
			bComp := (*components.BusinessComponent)(world.Get(c.BusinessEntity, busCompID))
			bComp.OwnerID = c.NewOwnerID

			// Massive negative hook from victim to forger
			s.hookGraph.AddHook(c.OldOwnerID, c.NewOwnerID, -100)

			// Log the theft in forger's memory
			if world.Alive(c.ForgerEntity) {
				mem := (*components.Memory)(world.Get(c.ForgerEntity, memID))
				mem.Events[mem.Head] = components.MemoryEvent{
					TargetID:        c.OldOwnerID,
					InteractionType: components.InteractionTheft,
					TickStamp:       0,
					Value:           int32(c.BusinessEntity.ID()),
				}
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))
			}
		}
	}
}
