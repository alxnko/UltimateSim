package systems

import (
	"sort"

	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Evolution: Phase 70 - The Deed Forgery Engine
// ForgerySystem bridges Economy, Genetics, and Justice. Desperate, high-intellect NPCs
// forge deeds to steal Business ownership from lower-intellect owners.

type forgeryBusinessData struct {
	entity  ecs.Entity
	busComp *components.BusinessComponent
}

type ForgerySystem struct {
	world       *ecs.World
	hooks       *engine.SparseHookGraph
	tickCounter uint64

	// Component IDs
	npcID      ecs.ID
	identID    ecs.ID
	genomeID   ecs.ID
	despID     ecs.ID
	memID      ecs.ID
	businessID ecs.ID

	npcFilter      ecs.Filter
	businessFilter ecs.Filter

	// Cache
	businesses []forgeryBusinessData
	ownerIntel map[uint64]uint8
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	npcID := ecs.ComponentID[components.NPC](world)
	identID := ecs.ComponentID[components.Identity](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	businessID := ecs.ComponentID[components.BusinessComponent](world)

	npcMask := filter.All(npcID, identID, genomeID, despID, memID)
	busMask := filter.All(businessID)

	return &ForgerySystem{
		world:          world,
		hooks:          hooks,
		npcID:          npcID,
		identID:        identID,
		genomeID:       genomeID,
		despID:         despID,
		memID:          memID,
		businessID:     businessID,
		npcFilter:      &npcMask,
		businessFilter: &busMask,
		businesses:     make([]forgeryBusinessData, 0, 100),
		ownerIntel:     make(map[uint64]uint8),
	}
}

func (s *ForgerySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Execute on an offset tick to distribute computational load
	if s.tickCounter%100 != 0 {
		return
	}

	// 1. Pre-cache all business entities and their owners
	s.businesses = s.businesses[:0]
	clear(s.ownerIntel)

	busQuery := world.Query(s.businessFilter)
	for busQuery.Next() {
		busComp := (*components.BusinessComponent)(busQuery.Get(s.businessID))
		s.businesses = append(s.businesses, forgeryBusinessData{
			entity:  busQuery.Entity(),
			busComp: busComp,
		})
	}

	// 2. Pre-cache all NPC intellects to avoid nested queries
	// Only cache intellect for NPCs that are currently business owners to save time,
	// or cache all if they might be owners. We'll cache all NPCs just in case.
	npcIntelQuery := world.Query(filter.All(s.identID, s.genomeID))
	for npcIntelQuery.Next() {
		id := (*components.Identity)(npcIntelQuery.Get(s.identID))
		genome := (*components.GenomeComponent)(npcIntelQuery.Get(s.genomeID))
		s.ownerIntel[id.ID] = genome.Intellect
	}

	// Make execution deterministic by sorting cached businesses
	sort.Slice(s.businesses, func(i, j int) bool {
		return s.businesses[i].busComp.OwnerID < s.businesses[j].busComp.OwnerID
	})

	// 3. Iterate over potential forgers
	forgerQuery := world.Query(s.npcFilter)
	for forgerQuery.Next() {
		desp := (*components.DesperationComponent)(forgerQuery.Get(s.despID))

		// Only desperate NPCs resort to forgery
		if desp.Level < 50 {
			continue
		}

		genome := (*components.GenomeComponent)(forgerQuery.Get(s.genomeID))

		// Only high-intellect NPCs can forge deeds
		if genome.Intellect < 120 {
			continue
		}

		ident := (*components.Identity)(forgerQuery.Get(s.identID))
		mem := (*components.Memory)(forgerQuery.Get(s.memID))

		// 4. Find a vulnerable business
		for _, bData := range s.businesses {
			ownerID := bData.busComp.OwnerID

			// Can't forge your own deed
			if ownerID == ident.ID || ownerID == 0 {
				continue
			}

			ownerIntellect, exists := s.ownerIntel[ownerID]
			if !exists {
				continue
			}

			// Forgery succeeds if forger's intellect is significantly higher than owner's
			if genome.Intellect > ownerIntellect && (genome.Intellect - ownerIntellect) > 40 {
				// Execute Forgery
				bData.busComp.OwnerID = ident.ID

				// Log Memory Event
				mem.Events[mem.Head] = components.MemoryEvent{
					TargetID:        ownerID,
					TickStamp:       s.tickCounter,
					InteractionType: components.InteractionTheft,
					LanguageID:      0,
					Value:           0,
				}
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))

				// Generate massive negative hook via SparseHookGraph
				s.hooks.AddHook(ownerID, ident.ID, -100)

				// Once successful, stop looking for businesses to forge this tick for this NPC
				break
			}
		}
	}
}
