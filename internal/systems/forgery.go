package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Phase 70: Deed Forgery Engine
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect NPCs
// to steal the BusinessComponent ownership from lower-intellect owners.

type ForgerySystem struct {
	desperateFilter ecs.Filter
	businessFilter  ecs.Filter
	npcFilter       ecs.Filter

	identID  ecs.ID
	genomeID ecs.ID
	despID   ecs.ID
	busID    ecs.ID
	busEntID ecs.ID
	memID    ecs.ID
	npcID    ecs.ID

	hooks *engine.SparseHookGraph
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	identID := ecs.ComponentID[components.Identity](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	busID := ecs.ComponentID[components.BusinessComponent](world)
	busEntID := ecs.ComponentID[components.BusinessEntity](world)
	memID := ecs.ComponentID[components.Memory](world)
	npcID := ecs.ComponentID[components.NPC](world)

	dMask := ecs.All(identID, genomeID, despID, memID, npcID)
	f := ecs.Filter(&dMask)

	bMask := ecs.All(busEntID, busID)
	bf := ecs.Filter(&bMask)

	nMask := ecs.All(identID, genomeID, npcID)
	nf := ecs.Filter(&nMask)

	return &ForgerySystem{
		desperateFilter: f,
		businessFilter:  bf,
		npcFilter:       nf,
		identID:         identID,
		genomeID:        genomeID,
		despID:          despID,
		busID:           busID,
		busEntID:        busEntID,
		memID:           memID,
		npcID:           npcID,
		hooks:           hooks,
	}
}

type intelData struct {
	intellect uint8
	entity    ecs.Entity
}

func (s *ForgerySystem) Update(world *ecs.World) {
	// Pre-cache all NPCs' intellects
	intellects := make(map[uint64]intelData)
	nQuery := world.Query(s.npcFilter)
	for nQuery.Next() {
		ident := (*components.Identity)(nQuery.Get(s.identID))
		genome := (*components.GenomeComponent)(nQuery.Get(s.genomeID))
		intellects[ident.ID] = intelData{
			intellect: genome.Intellect,
			entity:    nQuery.Entity(),
		}
	}

	// Iterate over desperate NPCs
	dQuery := world.Query(s.desperateFilter)

	// Collect desperate NPCs in a slice
	type desperateNPC struct {
		id        uint64
		intellect uint8
		entity    ecs.Entity
		memory    *components.Memory
	}
	var desperates []desperateNPC

	for dQuery.Next() {
		ident := (*components.Identity)(dQuery.Get(s.identID))
		genome := (*components.GenomeComponent)(dQuery.Get(s.genomeID))
		mem := (*components.Memory)(dQuery.Get(s.memID))
		desperates = append(desperates, desperateNPC{
			id:        ident.ID,
			intellect: genome.Intellect,
			entity:    dQuery.Entity(),
			memory:    mem,
		})
	}

	// Iterate over businesses
	bQuery := world.Query(s.businessFilter)
	type businessData struct {
		entity  ecs.Entity
		ownerID uint64
		comp    *components.BusinessComponent
	}
	var businesses []businessData
	for bQuery.Next() {
		bus := (*components.BusinessComponent)(bQuery.Get(s.busID))
		businesses = append(businesses, businessData{
			entity:  bQuery.Entity(),
			ownerID: bus.OwnerID,
			comp:    bus,
		})
	}

	// Now match desperate NPCs to vulnerable businesses
	for i := range desperates {
		d := desperates[i]

		for j := range businesses {
			b := businesses[j]

			// Skip if owner is already the desperate NPC
			if b.ownerID == d.id {
				continue
			}

			// Get owner's intellect
			ownerIntel, ok := intellects[b.ownerID]
			if !ok {
				continue // Owner not found/dead
			}

			if d.intellect > ownerIntel.intellect {
				// Forgery successful
				originalOwnerID := b.ownerID
				b.comp.OwnerID = d.id
				businesses[j].ownerID = d.id // Update cached owner

				// Generate negative hook from original owner to forger
				if s.hooks != nil {
					s.hooks.AddHook(originalOwnerID, d.id, -100)
				}

				// Log InteractionTheft in forger's Memory
				event := components.MemoryEvent{
					TargetID:        originalOwnerID,
					InteractionType: components.InteractionTheft,
					TickStamp:       0, // Or whatever tick we have
				}
				d.memory.Events[d.memory.Head] = event
				d.memory.Head = (d.memory.Head + 1) % uint8(len(d.memory.Events))

				break // One business per desperate NPC per tick
			}
		}
	}
}
