package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 70 - The Deed Forgery Engine
// ForgerySystem bridges Economy, Genetics, and Justice.
// Desperate NPCs with high intellect (>= 100) steal BusinessComponent ownership
// from lower-intellect owners. Generates a massive -100 grudge and logs an InteractionTheft.

type ForgerySystem struct {
	tickCounter uint64
	hooks       *engine.SparseHookGraph

	forgerFilter ecs.Filter
	ownerFilter  ecs.Filter

	identID      ecs.ID
	despID       ecs.ID
	genomeID     ecs.ID
	memID        ecs.ID
	busID        ecs.ID
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	identID := ecs.ComponentID[components.Identity](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	busID := ecs.ComponentID[components.BusinessComponent](world)

	fMask := ecs.All(identID, despID, genomeID, memID)
	oMask := ecs.All(busID)

	return &ForgerySystem{
		tickCounter: 0,
		hooks:       hooks,
		forgerFilter: &fMask,
		ownerFilter:  &oMask,
		identID:      identID,
		despID:       despID,
		genomeID:     genomeID,
		memID:        memID,
		busID:        busID,
	}
}

type businessOwnerData struct {
	businessEnt ecs.Entity
	ownerID     uint64
	intellect   uint8
}

func (s *ForgerySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle to offset ticks
	if s.tickCounter%13 != 0 {
		return
	}

	if s.hooks == nil {
		return
	}

	// 1. Gather all Businesses and map them to their owner's intellect
	businesses := make([]businessOwnerData, 0)

	ownerQuery := world.Query(s.ownerFilter)
	for ownerQuery.Next() {
		bus := (*components.BusinessComponent)(ownerQuery.Get(s.busID))
		businesses = append(businesses, businessOwnerData{
			businessEnt: ownerQuery.Entity(),
			ownerID:     bus.OwnerID,
			intellect:   0, // We will map intellect in a separate pass
		})
	}

	// Helper to find owner intellect by ownerID
	ownerIntellects := make(map[uint64]uint8)

	// Create a filter to quickly find the identities and genomes of all owners
	ownerIdentFilter := ecs.All(s.identID, s.genomeID)
	identQuery := world.Query(ownerIdentFilter)
	for identQuery.Next() {
		ident := (*components.Identity)(identQuery.Get(s.identID))
		genome := (*components.GenomeComponent)(identQuery.Get(s.genomeID))
		ownerIntellects[ident.ID] = genome.Intellect
	}

	// Update the businesses array with the found intellects
	for i := range businesses {
		if intel, ok := ownerIntellects[businesses[i].ownerID]; ok {
			businesses[i].intellect = intel
		}
	}

	// Track stolen businesses to prevent multiple forgeries on the same business in one tick
	stolenThisTick := make(map[ecs.Entity]bool)

	// 2. Iterate forgers
	forgerQuery := world.Query(s.forgerFilter)
	for forgerQuery.Next() {
		desp := (*components.DesperationComponent)(forgerQuery.Get(s.despID))
		genome := (*components.GenomeComponent)(forgerQuery.Get(s.genomeID))

		if desp.Level < 50 || genome.Intellect < 100 {
			continue
		}

		ident := (*components.Identity)(forgerQuery.Get(s.identID))
		mem := (*components.Memory)(forgerQuery.Get(s.memID))

		// Find a vulnerable business
		var targetBus *businessOwnerData

		for i := range businesses {
			b := &businesses[i]
			// Cannot steal from yourself
			if b.ownerID == ident.ID {
				continue
			}
			// Cannot steal the same business twice in a tick
			if stolenThisTick[b.businessEnt] {
				continue
			}

			// Intellect check
			if genome.Intellect > b.intellect {
				targetBus = b
				break
			}
		}

		if targetBus != nil {
			// Execute forgery
			stolenThisTick[targetBus.businessEnt] = true

			// The actual structural reassignment is done directly on the component
			// since it's just a pointer dereference within the query lock.
			// Wait, the business component is not queried in this forger loop,
			// but we can fetch it via world.Get
			busComp := (*components.BusinessComponent)(world.Get(targetBus.businessEnt, s.busID))

			originalOwnerID := targetBus.ownerID
			busComp.OwnerID = ident.ID

			// Update the business struct for next iterations in the same tick
			targetBus.ownerID = ident.ID
			targetBus.intellect = genome.Intellect

			// Generate the grudge
			s.hooks.AddHook(originalOwnerID, ident.ID, -100)

			// Log the theft in memory
			for i := 0; i < len(mem.Events); i++ {
				if mem.Events[i].InteractionType == 0 { // Empty slot
					mem.Events[i] = components.MemoryEvent{
						TargetID:        originalOwnerID,
						TickStamp:       s.tickCounter,
						InteractionType: components.InteractionTheft,
					}
					break
				}
			}
		}
	}
}
