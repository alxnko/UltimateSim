package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 70 - The Deed Forgery Engine
// ForgerySystem bridges Economy, Genetics, and Justice by allowing desperate,
// high-intellect NPCs to steal the BusinessComponent ownership from lower-intellect owners.

type forgeryData struct {
	BusinessEntity ecs.Entity
	OriginalOwner  uint64
	ForgerID       uint64
	ForgerEntity   ecs.Entity
}

type forgeryNpcData struct {
	Entity    ecs.Entity
	ID        uint64
	Intellect uint8
}

type ForgerySystem struct {
	hookGraph   *engine.SparseHookGraph
	tickCounter uint64

	businessFilter ecs.Filter
	npcFilter      ecs.Filter

	busCompID ecs.ID
	idID      ecs.ID
	genomeID  ecs.ID
	despID    ecs.ID
	memID     ecs.ID

	forgeries         []forgeryData
	npcCache          []forgeryNpcData
	ownerIntellectMap map[uint64]uint8
	hasForged         map[uint64]bool
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	busTagID := ecs.ComponentID[components.BusinessEntity](world)
	busCompID := ecs.ComponentID[components.BusinessComponent](world)

	idID := ecs.ComponentID[components.Identity](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	memID := ecs.ComponentID[components.Memory](world)

	busMask := ecs.All(busTagID, busCompID)
	npcMask := ecs.All(idID, genomeID, despID, memID)

	return &ForgerySystem{
		hookGraph:         hooks,
		tickCounter:       0,
		businessFilter:    &busMask,
		npcFilter:         &npcMask,
		busCompID:         busCompID,
		idID:              idID,
		genomeID:          genomeID,
		despID:            despID,
		memID:             memID,
		forgeries:         make([]forgeryData, 0, 100),
		npcCache:          make([]forgeryNpcData, 0, 100),
		ownerIntellectMap: make(map[uint64]uint8),
		hasForged:         make(map[uint64]bool),
	}
}

func (s *ForgerySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Process logic periodically to save CPU
	if s.tickCounter%71 != 0 {
		return
	}

	if s.hookGraph == nil {
		return
	}

	// 1. Clear caches
	s.forgeries = s.forgeries[:0]
	s.npcCache = s.npcCache[:0]
	for k := range s.ownerIntellectMap {
		delete(s.ownerIntellectMap, k)
	}
	for k := range s.hasForged {
		delete(s.hasForged, k)
	}

	// 2. Gather all NPCs
	npcQuery := world.Query(s.npcFilter)
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(s.idID))
		genome := (*components.GenomeComponent)(npcQuery.Get(s.genomeID))
		desp := (*components.DesperationComponent)(npcQuery.Get(s.despID))

		s.ownerIntellectMap[ident.ID] = genome.Intellect

		if desp.Level > 0 {
			s.npcCache = append(s.npcCache, forgeryNpcData{
				Entity:    npcQuery.Entity(),
				ID:        ident.ID,
				Intellect: genome.Intellect,
			})
		}
	}

	// 3. Find Vulnerable Businesses
	busQuery := world.Query(s.businessFilter)
	for busQuery.Next() {
		bus := (*components.BusinessComponent)(busQuery.Get(s.busCompID))

		ownerIntellect, exists := s.ownerIntellectMap[bus.OwnerID]
		if !exists {
			continue // Owner doesn't exist in current NPC pool
		}

		// Look for a forger
		for _, npc := range s.npcCache {
			if npc.ID == bus.OwnerID {
				continue
			}

			if s.hasForged[npc.ID] {
				continue
			}

			if npc.Intellect > ownerIntellect {
				s.forgeries = append(s.forgeries, forgeryData{
					BusinessEntity: busQuery.Entity(),
					OriginalOwner:  bus.OwnerID,
					ForgerID:       npc.ID,
					ForgerEntity:   npc.Entity,
				})
				s.hasForged[npc.ID] = true
				break // Only one forgery per business per tick
			}
		}
	}

	// 4. Apply Forgeries
	for _, fg := range s.forgeries {
		// Update Business Owner
		if world.Alive(fg.BusinessEntity) {
			bus := (*components.BusinessComponent)(world.Get(fg.BusinessEntity, s.busCompID))
			bus.OwnerID = fg.ForgerID
		}

		// Update Forger Memory
		if world.Alive(fg.ForgerEntity) {
			if world.Has(fg.ForgerEntity, s.memID) {
				mem := (*components.Memory)(world.Get(fg.ForgerEntity, s.memID))

				event := components.MemoryEvent{
					TargetID:        fg.OriginalOwner,
					TickStamp:       s.tickCounter,
					InteractionType: 4, // InteractionTheft
					LanguageID:      0,
					Value:           -100,
				}

				mem.Events[mem.Head] = event
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))
			}
		}

		// Generate negative hook
		s.hookGraph.AddHook(fg.OriginalOwner, fg.ForgerID, -100)
	}
}
