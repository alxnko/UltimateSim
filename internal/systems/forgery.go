package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 70 - The Deed Forgery Engine
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect
// NPCs to steal the BusinessComponent ownership from lower-intellect owners.

type ForgerySystem struct {
	npcFilter      ecs.Filter
	businessFilter ecs.Filter

	identID    ecs.ID
	despID     ecs.ID
	genID      ecs.ID
	memID      ecs.ID
	businessID ecs.ID

	hooks *engine.SparseHookGraph
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	identID := ecs.ComponentID[components.Identity](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	memID := ecs.ComponentID[components.Memory](world)

	businessID := ecs.ComponentID[components.BusinessComponent](world)

	npcMask := ecs.All(identID, despID, genID, memID)
	busMask := ecs.All(businessID)

	return &ForgerySystem{
		npcFilter:      &npcMask,
		businessFilter: &busMask,
		identID:        identID,
		despID:         despID,
		genID:          genID,
		memID:          memID,
		businessID:     businessID,
		hooks:          hooks,
	}
}

type busData struct {
	entity    ecs.Entity
	ownerID   uint64
	component *components.BusinessComponent
}

type forgerNpcData struct {
	ident     *components.Identity
	desp      *components.DesperationComponent
	gen       *components.GenomeComponent
	mem       *components.Memory
	intellect uint8
}

func (s *ForgerySystem) Update(world *ecs.World) {
	// Step 1: Pre-cache all vulnerable businesses and map owners to their Intellect
	bQuery := world.Query(s.businessFilter)
	businesses := make([]busData, 0, 50)
	ownerIDs := make(map[uint64]bool)

	for bQuery.Next() {
		bus := (*components.BusinessComponent)(bQuery.Get(s.businessID))
		businesses = append(businesses, busData{
			entity:    bQuery.Entity(),
			ownerID:   bus.OwnerID,
			component: bus,
		})
		ownerIDs[bus.OwnerID] = true
	}

	if len(businesses) == 0 {
		return
	}

	// Fetch intellects of current owners and potential forgers
	ownerIntellects := make(map[uint64]uint8)
	var forgers []forgerNpcData

	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(s.identID))
		desp := (*components.DesperationComponent)(npcQuery.Get(s.despID))
		gen := (*components.GenomeComponent)(npcQuery.Get(s.genID))
		mem := (*components.Memory)(npcQuery.Get(s.memID))

		// If this NPC is a business owner, store their intellect
		if ownerIDs[ident.ID] {
			ownerIntellects[ident.ID] = gen.Intellect
		}

		// Identify potential forgers
		if desp.Level > 50 && gen.Intellect > 70 {
			forgers = append(forgers, forgerNpcData{
				ident:     ident,
				desp:      desp,
				gen:       gen,
				mem:       mem,
				intellect: gen.Intellect,
			})
		}
	}

	if len(forgers) == 0 {
		return
	}

	// Step 2: Desperate Geniuses attempt to forge deeds
	forgedBusinesses := make(map[ecs.Entity]bool)

	for i := 0; i < len(forgers); i++ {
		f := &forgers[i]

		// Find a business they can steal
		for j := 0; j < len(businesses); j++ {
			b := &businesses[j]

			// Don't steal a business that was already stolen this tick
			if forgedBusinesses[b.entity] {
				continue
			}

			// Don't steal from yourself
			if b.ownerID == f.ident.ID {
				continue
			}

			targetIntellect := ownerIntellects[b.ownerID]

			// The theft is successful if the forger's intellect exceeds the target's
			if f.intellect > targetIntellect {
				// 1. Swap ownership
				b.component.OwnerID = f.ident.ID
				forgedBusinesses[b.entity] = true

				// 2. Log Memory Event
				event := components.MemoryEvent{
					TargetID:        b.ownerID,
					InteractionType: components.InteractionTheft,
					Value:           int32(b.entity.ID()), // Abstract representation of the stolen item
					TickStamp:       0,
				}
				f.mem.Events[f.mem.Head] = event
				f.mem.Head = (f.mem.Head + 1) % uint8(len(f.mem.Events))

				// 3. Emit massive negative hook via SparseHookGraph against the forger
				if s.hooks != nil {
					s.hooks.AddHook(b.ownerID, f.ident.ID, -100)
				}

				// The forger is satisfied for now (Desperation resets)
				f.desp.Level = 0

				// A forger only steals one business per tick
				break
			}
		}
	}
}
