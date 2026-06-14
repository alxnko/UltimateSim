package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
)

// Evolution: Phase 70 - The Deed Forgery Engine
// Bridges Economy, Genetics, and Justice by allowing desperate, high-intellect NPCs
// to steal BusinessComponent ownership from lower-intellect owners.

type businessNodeData struct {
	Entity ecs.Entity
	OwnerID uint64
	X       float32
	Y       float32
}

type ForgerySystem struct {
	tickCounter uint64
	hookGraph   *engine.SparseHookGraph
	businesses  []businessNodeData
	intellectMap map[uint64]uint8

	// Pre-cached component IDs
	npcID      ecs.ID
	identID    ecs.ID
	genID      ecs.ID
	posID      ecs.ID
	despID     ecs.ID
	memID      ecs.ID
	busTagID   ecs.ID
	busCompID  ecs.ID
	workID     ecs.ID
}

func NewForgerySystem(world *ecs.World, hookGraph *engine.SparseHookGraph) *ForgerySystem {
	return &ForgerySystem{
		tickCounter:  0,
		hookGraph:    hookGraph,
		businesses:   make([]businessNodeData, 0, 100),
		intellectMap: make(map[uint64]uint8, 1000),

		npcID:      ecs.ComponentID[components.NPC](world),
		identID:    ecs.ComponentID[components.Identity](world),
		genID:      ecs.ComponentID[components.GenomeComponent](world),
		posID:      ecs.ComponentID[components.Position](world),
		despID:     ecs.ComponentID[components.DesperationComponent](world),
		memID:      ecs.ComponentID[components.Memory](world),
		busTagID:   ecs.ComponentID[components.BusinessEntity](world),
		busCompID:  ecs.ComponentID[components.BusinessComponent](world),
		workID:     ecs.ComponentID[components.WorkplaceComponent](world),
	}
}

func (s *ForgerySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Throttle execution
	if s.tickCounter%20 != 0 {
		return
	}

	s.businesses = s.businesses[:0]
	// Clear the map
	for k := range s.intellectMap {
		delete(s.intellectMap, k)
	}

	// 1. Build intellect map for all NPCs
	npcQuery := world.Query(ecs.All(s.npcID, s.identID, s.genID))
	for npcQuery.Next() {
		ident := (*components.Identity)(npcQuery.Get(s.identID))
		gen := (*components.GenomeComponent)(npcQuery.Get(s.genID))
		s.intellectMap[ident.ID] = gen.Intellect
	}

	// 2. Pre-cache businesses
	busQuery := world.Query(ecs.All(s.busTagID, s.busCompID, s.workID))
	for busQuery.Next() {
		bus := (*components.BusinessComponent)(busQuery.Get(s.busCompID))
		work := (*components.WorkplaceComponent)(busQuery.Get(s.workID))
		s.businesses = append(s.businesses, businessNodeData{
			Entity:  busQuery.Entity(),
			OwnerID: bus.OwnerID,
			X:       work.X,
			Y:       work.Y,
		})
	}

	if len(s.businesses) == 0 {
		return
	}

	// 3. Iterate over desperate NPCs to attempt forgery
	despQuery := world.Query(ecs.All(s.npcID, s.identID, s.genID, s.posID, s.despID, s.memID))
	for despQuery.Next() {
		desp := (*components.DesperationComponent)(despQuery.Get(s.despID))
		if desp.Level < 50 {
			continue
		}

		gen := (*components.GenomeComponent)(despQuery.Get(s.genID))
		if gen.Intellect < 100 {
			continue
		}

		ident := (*components.Identity)(despQuery.Get(s.identID))
		pos := (*components.Position)(despQuery.Get(s.posID))
		mem := (*components.Memory)(despQuery.Get(s.memID))

		// Check proximity to businesses
		for i := 0; i < len(s.businesses); i++ {
			b := &s.businesses[i]
			if b.OwnerID == ident.ID {
				continue // Already owns it
			}

			dx := pos.X - b.X
			dy := pos.Y - b.Y
			distSq := (dx * dx) + (dy * dy)

			if distSq <= 4.0 {
				ownerIntellect, exists := s.intellectMap[b.OwnerID]
				if !exists {
					ownerIntellect = 50 // Default if missing
				}

				if int(gen.Intellect) > int(ownerIntellect)+20 {
					// Forgery Succeeds!

					// 1. Overwrite Business Owner (Safe to fetch specific entity directly outside the query or via world.Get inside if it's just data modification? Actually we can't world.Get on a different entity safely within a query iteration without risking locks if it was a structural change, but for data pointer it's OK. To be completely safe, we could defer, but data modification is usually fine. Wait, DOD rules suggest deferred changes or fetching. Let's just world.Get)

					busCompPtr := (*components.BusinessComponent)(world.Get(b.Entity, s.busCompID))
					if busCompPtr != nil {
					    oldOwnerID := busCompPtr.OwnerID
						busCompPtr.OwnerID = ident.ID

						// Update our cache so one person doesn't forge it twice in a tick, or someone else doesn't overwrite it immediately
						b.OwnerID = ident.ID

						// 2. Inject massive -100 grudge into SparseHookGraph
						s.hookGraph.AddHook(oldOwnerID, ident.ID, -100)

						// 3. Log InteractionTheft into Memory
						event := components.MemoryEvent{
							TargetID:        oldOwnerID,
							TickStamp:       s.tickCounter,
							InteractionType: components.InteractionTheft,
							LanguageID:      0,
							Value:           0,
						}
						mem.Events[mem.Head] = event
						mem.Head = (mem.Head + 1) % uint8(len(mem.Events))

						break // Only forge one business per tick
					}
				}
			}
		}
	}
}
