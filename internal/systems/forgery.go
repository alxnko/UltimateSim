package systems

import (
	"github.com/ALXNKO/UltimateSim/internal/components"
	"github.com/ALXNKO/UltimateSim/internal/engine"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/filter"
)

// Phase 70: The Deed Forgery Engine
// ForgerySystem bridges Economy, Biology/Genetics, and Justice.
// Desperate NPCs with high Intellect attempt to steal businesses from owners with lower Intellect.

type businessData struct {
	entity  ecs.Entity
	ownerID uint64
	x       float32
	y       float32
}

type ForgerySystem struct {
	npcFilter      ecs.Filter
	businessFilter ecs.Filter
	ownerFilter    ecs.Filter

	hookGraph *engine.SparseHookGraph

	tickCounter uint64

	// Pre-allocated flat slices and maps to prevent Arche-Go locks and GC overhead
	businesses   []businessData
	ownerIntels  map[uint64]uint8
	forgeryQueue []struct {
		businessEnt ecs.Entity
		newOwnerID  uint64
		oldOwnerID  uint64
	}
}

// IsExpensive returns true to throttle this system during fast-forward.
func (s *ForgerySystem) IsExpensive() bool {
	return true
}

// NewForgerySystem creates a new ForgerySystem.
func NewForgerySystem(world *ecs.World, hooks *engine.SparseHookGraph) *ForgerySystem {
	// NPC Filter: needs to be desperate and have intellect
	npcID := ecs.ComponentID[components.NPC](world)
	posID := ecs.ComponentID[components.Position](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)
	identID := ecs.ComponentID[components.Identity](world)
	memID := ecs.ComponentID[components.Memory](world)

	npcF := filter.All(npcID, posID, despID, genID, identID, memID)

	// Business Filter: physical location and ownership
	busTagID := ecs.ComponentID[components.BusinessEntity](world)
	busCompID := ecs.ComponentID[components.BusinessComponent](world)
	wpID := ecs.ComponentID[components.WorkplaceComponent](world)

	busF := filter.All(busTagID, busCompID, wpID)

	// Owner Filter: mapping owner ID to Intellect
	ownF := filter.All(identID, genID)

	return &ForgerySystem{
		npcFilter:      &npcF,
		businessFilter: &busF,
		ownerFilter:    &ownF,
		hookGraph:      hooks,
		businesses:     make([]businessData, 0, 100),
		ownerIntels:    make(map[uint64]uint8, 100),
		forgeryQueue:   make([]struct{
			businessEnt ecs.Entity
			newOwnerID  uint64
			oldOwnerID  uint64
		}, 0, 10),
	}
}

// Update executes the system logic per tick.
func (s *ForgerySystem) Update(world *ecs.World) {
	s.tickCounter++

	// Offset execution to spread load
	if s.tickCounter%50 != 0 {
		return
	}

	busCompID := ecs.ComponentID[components.BusinessComponent](world)
	wpID := ecs.ComponentID[components.WorkplaceComponent](world)

	identID := ecs.ComponentID[components.Identity](world)
	genID := ecs.ComponentID[components.GenomeComponent](world)

	posID := ecs.ComponentID[components.Position](world)
	despID := ecs.ComponentID[components.DesperationComponent](world)

	s.businesses = s.businesses[:0]
	// Cannot reuse map directly without clearing, but since it's an offset loop,
	// we will clear it.
	for k := range s.ownerIntels {
		delete(s.ownerIntels, k)
	}
	s.forgeryQueue = s.forgeryQueue[:0]

	// 1. Cache all Business entities
	bQuery := world.Query(s.businessFilter)
	for bQuery.Next() {
		bus := (*components.BusinessComponent)(bQuery.Get(busCompID))
		wp := (*components.WorkplaceComponent)(bQuery.Get(wpID))

		s.businesses = append(s.businesses, businessData{
			entity:  bQuery.Entity(),
			ownerID: bus.OwnerID,
			x:       wp.X,
			y:       wp.Y,
		})
	}
	// bQuery auto-closes

	if len(s.businesses) == 0 {
		return
	}

	// 2. Cache all Owner Intellects
	oQuery := world.Query(s.ownerFilter)
	for oQuery.Next() {
		ident := (*components.Identity)(oQuery.Get(identID))
		gen := (*components.GenomeComponent)(oQuery.Get(genID))
		s.ownerIntels[ident.ID] = gen.Intellect
	}
	// oQuery auto-closes

	// 3. Evaluate desperate NPCs for forgery
	nQuery := world.Query(s.npcFilter)
	for nQuery.Next() {
		desp := (*components.DesperationComponent)(nQuery.Get(despID))
		if desp.Level < 50 {
			continue // Not desperate enough
		}

		gen := (*components.GenomeComponent)(nQuery.Get(genID))
		if gen.Intellect < 120 {
			continue // Not smart enough to forge deeds
		}

		pos := (*components.Position)(nQuery.Get(posID))
		ident := (*components.Identity)(nQuery.Get(identID))

		// Search for nearby businesses
		for _, b := range s.businesses {
			if b.ownerID == ident.ID || b.ownerID == 0 {
				continue // Already owns it, or no owner
			}

			distSq := (pos.X-b.x)*(pos.X-b.x) + (pos.Y-b.y)*(pos.Y-b.y)
			if distSq <= 4.0 { // Must be physically at the business
				targetIntellect := s.ownerIntels[b.ownerID]

				// Forgery succeeds if forger is smarter than the owner
				if gen.Intellect > targetIntellect {
					// Add to structural queue to prevent nested Arche-Go modifications
					s.forgeryQueue = append(s.forgeryQueue, struct{
						businessEnt ecs.Entity
						newOwnerID  uint64
						oldOwnerID  uint64
					}{
						businessEnt: b.entity,
						newOwnerID:  ident.ID,
						oldOwnerID:  b.ownerID,
					})

					break // One forgery per NPC per evaluation
				}
			}
		}
	}
	// nQuery auto-closes

	// 4. Apply structural/state changes safely outside the loop
	for _, forgery := range s.forgeryQueue {
		// Verify the business still exists
		if world.Alive(forgery.businessEnt) && world.Has(forgery.businessEnt, busCompID) {
			bus := (*components.BusinessComponent)(world.Get(forgery.businessEnt, busCompID))

			// Only update if the owner hasn't changed in the meantime
			if bus.OwnerID == forgery.oldOwnerID {
				bus.OwnerID = forgery.newOwnerID

				// Find the new owner entity to apply state changes
				identID := ecs.ComponentID[components.Identity](world)
				forgerEntQuery := world.Query(filter.All(identID))
				var forgerEnt ecs.Entity
				var found bool
				for forgerEntQuery.Next() {
					id := (*components.Identity)(forgerEntQuery.Get(identID))
					if id.ID == forgery.newOwnerID {
						forgerEnt = forgerEntQuery.Entity()
						found = true
						break
					}
				}
				forgerEntQuery.Close() // Close before using the entity

				if found {
					// Reduce desperation
					despID := ecs.ComponentID[components.DesperationComponent](world)
					if world.Has(forgerEnt, despID) {
						desp := (*components.DesperationComponent)(world.Get(forgerEnt, despID))
						desp.Level = 0
					}

					// Log InteractionTheft into Memory
					memID := ecs.ComponentID[components.Memory](world)
					if world.Has(forgerEnt, memID) {
						mem := (*components.Memory)(world.Get(forgerEnt, memID))
						idx := mem.Head % 50
						mem.Events[idx] = components.MemoryEvent{
							TargetID:        forgery.oldOwnerID,
							TickStamp:       s.tickCounter,
							InteractionType: components.InteractionTheft,
							LanguageID:      0,
							Value:           0,
						}
						mem.Head++
					}
				}

				// 5. Generate negative hook from the victim to the forger
				if s.hookGraph != nil {
					s.hookGraph.AddHook(forgery.oldOwnerID, forgery.newOwnerID, -100)
				}
			}
		}
	}
}
