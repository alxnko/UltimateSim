package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 70: Deed Forgery Engine
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect NPCs
// to steal the BusinessComponent ownership from lower-intellect owners.

type forgeryChange struct {
	forger    ecs.Entity
	targetBiz ecs.Entity
	ownerID   uint64
	forgerID  uint64
	stealVal  float32
}

type bizData struct {
	Entity  ecs.Entity
	OwnerID uint64
}

type ForgerySystem struct {
	forgerFilter ecs.Filter
	bizFilter    ecs.Filter
	npcFilter    ecs.Filter
	hooks        *engine.SparseHookGraph
	changes      []forgeryChange
	businesses   []bizData
	intellectMap map[uint64]uint8
}

func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	// Forger filter: Needs Desperation, Genome, Memory, Identity
	despID := ecs.ComponentID[components.DesperationComponent](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)
	memID := ecs.ComponentID[components.Memory](world)
	idID := ecs.ComponentID[components.Identity](world)
	f1 := filter.All(despID, genomeID, memID, idID)

	// Business filter
	bizID := ecs.ComponentID[components.BusinessComponent](world)
	bizTagID := ecs.ComponentID[components.BusinessEntity](world)
	f2 := filter.All(bizID, bizTagID)

	// General NPC filter for Intellect cache
	f3 := filter.All(idID, genomeID)

	return &ForgerySystem{
		forgerFilter: &f1,
		bizFilter:    &f2,
		npcFilter:    &f3,
		hooks:        hooks,
		changes:      make([]forgeryChange, 0, 100),
		businesses:   make([]bizData, 0, 100),
		intellectMap: make(map[uint64]uint8),
	}
}

func (s *ForgerySystem) Update(world *ecs.World) {
	// Step 1: Cache NPC intellects
	idID := ecs.ComponentID[components.Identity](world)
	genomeID := ecs.ComponentID[components.GenomeComponent](world)

	npcQuery := world.Query(s.npcFilter)
	clear(s.intellectMap) // reuse the map
	for npcQuery.Next() {
		id := (*components.Identity)(npcQuery.Get(idID))
		genome := (*components.GenomeComponent)(npcQuery.Get(genomeID))
		s.intellectMap[id.ID] = genome.Intellect
	}

	// Step 2: Cache target businesses
	bizID := ecs.ComponentID[components.BusinessComponent](world)
	bizQuery := world.Query(s.bizFilter)

	s.businesses = s.businesses[:0]

	for bizQuery.Next() {
		biz := (*components.BusinessComponent)(bizQuery.Get(bizID))
		s.businesses = append(s.businesses, bizData{
			Entity:  bizQuery.Entity(),
			OwnerID: biz.OwnerID,
		})
	}

	s.changes = s.changes[:0]

	// Step 3: Find forgers
	despID := ecs.ComponentID[components.DesperationComponent](world)
	forgerQuery := world.Query(s.forgerFilter)

	for forgerQuery.Next() {
		desp := (*components.DesperationComponent)(forgerQuery.Get(despID))

		// Skip if not desperate
		if desp.Level == 0 {
			continue
		}

		genome := (*components.GenomeComponent)(forgerQuery.Get(genomeID))
		id := (*components.Identity)(forgerQuery.Get(idID))

		// Try to find a target business
		for _, b := range s.businesses {
			// Skip if already owns it
			if b.OwnerID == id.ID {
				continue
			}

			ownerIntellect, ok := s.intellectMap[b.OwnerID]
			// Only steal if we outsmart them and they exist
			if ok && genome.Intellect > ownerIntellect {
				s.changes = append(s.changes, forgeryChange{
					forger:    forgerQuery.Entity(),
					targetBiz: b.Entity,
					ownerID:   b.OwnerID,
					forgerID:  id.ID,
					stealVal:  1.0, // Arbitrary interaction value
				})
				// One forgery per tick per NPC
				break
			}
		}
	}

	// Step 4: Apply structural changes
	memID := ecs.ComponentID[components.Memory](world)

	for _, change := range s.changes {
		// Update business ownership
		biz := (*components.BusinessComponent)(world.Get(change.targetBiz, bizID))
		if biz != nil && biz.OwnerID == change.ownerID {
			biz.OwnerID = change.forgerID

			// Apply SparseHookGraph penalty (-100) from victim to forger
			s.hooks.AddHook(change.ownerID, change.forgerID, -100)

			// Log InteractionTheft in forger's Memory
			mem := (*components.Memory)(world.Get(change.forger, memID))
			if mem != nil {
				event := components.MemoryEvent{
					TargetID:        change.ownerID,
					InteractionType: components.InteractionTheft,
					Value:           int32(change.stealVal),
					TickStamp:       0,
				}
				mem.Events[mem.Head] = event
				mem.Head = (mem.Head + 1) % uint8(len(mem.Events))
			}
		}
	}
}
